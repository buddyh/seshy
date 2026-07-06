package retention

import (
	"strings"
	"testing"
)

func TestInspectClaude(t *testing.T) {
	cases := []struct {
		name      string
		settings  string // "" = no file
		effective string
		source    string
		atRisk    bool
		wantErr   bool
	}{
		{name: "no file", effective: "30d", source: "default", atRisk: true},
		{name: "no key", settings: `{"model": "opus"}`, effective: "30d", source: "default", atRisk: true},
		{name: "configured", settings: `{"cleanupPeriodDays": 365}`, effective: "365d", source: "configured", atRisk: true},
		{name: "short", settings: `{"cleanupPeriodDays": 5}`, effective: "5d", source: "configured", atRisk: true},
		{name: "off convention", settings: `{"cleanupPeriodDays": 99999}`, effective: "off", source: "disabled", atRisk: false},
		{name: "non-numeric", settings: `{"cleanupPeriodDays": "abc"}`, effective: "30d", wantErr: true},
		{name: "malformed file", settings: `{not json`, effective: "30d", wantErr: true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			home := t.TempDir()
			if tc.settings != "" {
				writeFile(t, home, ".claude/settings.json", tc.settings, tMid)
			}
			p := inspectClaude(home)
			if p.Kind != AutoDelete {
				t.Errorf("kind = %s, want auto-delete", p.Kind)
			}
			if p.Effective != tc.effective {
				t.Errorf("effective = %q, want %q", p.Effective, tc.effective)
			}
			if (p.Err != nil) != tc.wantErr {
				t.Errorf("err = %v, wantErr %t", p.Err, tc.wantErr)
			}
			if !tc.wantErr {
				if p.Source != tc.source {
					t.Errorf("source = %q, want %q", p.Source, tc.source)
				}
				if p.AtRisk != tc.atRisk {
					t.Errorf("atRisk = %t, want %t", p.AtRisk, tc.atRisk)
				}
			}
		})
	}
}

func TestInspectGemini(t *testing.T) {
	cases := []struct {
		name      string
		settings  string
		effective string
		source    string
		atRisk    bool
		wantErr   bool
		wantNote  string
	}{
		{name: "no file", effective: "30d", source: "default", atRisk: true},
		{name: "no key", settings: `{"general": {}}`, effective: "30d", source: "default", atRisk: true},
		{name: "disabled", settings: `{"general": {"sessionRetention": {"enabled": false}}}`,
			effective: "off", source: "disabled", atRisk: false},
		{name: "disabled with age", settings: `{"general": {"sessionRetention": {"enabled": false, "maxAge": "90d"}}}`,
			effective: "off", source: "disabled", atRisk: false},
		{name: "days", settings: `{"general": {"sessionRetention": {"enabled": true, "maxAge": "90d"}}}`,
			effective: "90d", source: "configured", atRisk: true},
		{name: "months", settings: `{"general": {"sessionRetention": {"maxAge": "12m"}}}`,
			effective: "12m", source: "configured", atRisk: true},
		{name: "weeks", settings: `{"general": {"sessionRetention": {"maxAge": "8w"}}}`,
			effective: "8w", source: "configured", atRisk: true},
		{name: "bad unit", settings: `{"general": {"sessionRetention": {"maxAge": "90x"}}}`,
			effective: "30d", wantErr: true},
		{name: "non-object retention", settings: `{"general": {"sessionRetention": "30d"}}`,
			effective: "30d", wantErr: true},
		{name: "max count noted", settings: `{"general": {"sessionRetention": {"maxAge": "60d", "maxCount": 50}}}`,
			effective: "60d", source: "configured", atRisk: true, wantNote: "caps at 50 sessions"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			home := t.TempDir()
			if tc.settings != "" {
				writeFile(t, home, ".gemini/settings.json", tc.settings, tMid)
			}
			p := inspectGemini(home)
			if p.Effective != tc.effective {
				t.Errorf("effective = %q, want %q", p.Effective, tc.effective)
			}
			if (p.Err != nil) != tc.wantErr {
				t.Errorf("err = %v, wantErr %t", p.Err, tc.wantErr)
			}
			if !tc.wantErr {
				if p.Source != tc.source {
					t.Errorf("source = %q, want %q", p.Source, tc.source)
				}
				if p.AtRisk != tc.atRisk {
					t.Errorf("atRisk = %t, want %t", p.AtRisk, tc.atRisk)
				}
			}
			if tc.wantNote != "" && !strings.Contains(p.Note, tc.wantNote) {
				t.Errorf("note = %q, want it to mention %q", p.Note, tc.wantNote)
			}
		})
	}
}

func TestInspectDroid(t *testing.T) {
	cases := []struct {
		name      string
		settings  string
		effective string
		source    string
	}{
		{name: "no file", effective: "local: forever · cloud: Factory-managed", source: "default"},
		{name: "cloud days", settings: `{"sessionRetentionDays": 90}`,
			effective: "local: forever · cloud: 90d", source: "configured"},
		{name: "sync off", settings: `{"cloudSessionSync": false, "sessionRetentionDays": 90}`,
			effective: "local: forever · cloud: sync off", source: "configured"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			home := t.TempDir()
			if tc.settings != "" {
				writeFile(t, home, ".factory/settings.json", tc.settings, tMid)
			}
			p := inspectDroid(home)
			if p.Kind != CloudManaged {
				t.Errorf("kind = %s, want cloud-managed", p.Kind)
			}
			if p.AtRisk {
				t.Error("droid must never be at-risk: local files are kept")
			}
			if p.Effective != tc.effective {
				t.Errorf("effective = %q, want %q", p.Effective, tc.effective)
			}
			if p.Source != tc.source {
				t.Errorf("source = %q, want %q", p.Source, tc.source)
			}
		})
	}
}

func TestKeepForeverSources(t *testing.T) {
	home := t.TempDir()
	for _, key := range []string{"codex", "grok", "pi", "opencode", "agy", "cursor", "copilot"} {
		for _, s := range Sources {
			if s.Key != key {
				continue
			}
			p := s.Inspect(home)
			if p.Kind != KeepForever || p.AtRisk || p.Effective != "forever" {
				t.Errorf("%s: policy = %+v, want keep-forever / not at-risk", key, p)
			}
		}
	}
}
