package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/buddyh/seshy/internal/agents"
	"github.com/buddyh/seshy/internal/config"
	"github.com/buddyh/seshy/internal/render"
	"github.com/buddyh/seshy/internal/tui"
	"github.com/spf13/cobra"
)

var (
	flagCwd    string
	flagFormat string
	flagAll    bool
	flagNum    int
	flagAgent  string
)

func resolveTarget(args []string) string {
	p := flagCwd
	if len(args) > 0 && args[0] != "" {
		p = args[0]
	}
	if p == "" {
		p, _ = os.Getwd()
	}
	if abs, err := filepath.Abs(p); err == nil {
		p = abs
	}
	if real, err := filepath.EvalSymlinks(p); err == nil {
		p = real
	}
	return p
}

func isTTY() bool {
	fi, err := os.Stdout.Stat()
	return err == nil && fi.Mode()&os.ModeCharDevice != 0
}

func collect(target string) []agents.Session {
	ss := agents.Collect(target, flagNum, flagAll)
	if flagAgent != "" {
		var f []agents.Session
		for _, s := range ss {
			if s.Tool == flagAgent {
				f = append(f, s)
			}
		}
		ss = f
	}
	return ss
}

// pickFormat resolves the output format (explicit flag, else table on a TTY / JSON when piped).
func pickFormat() string {
	if flagFormat != "" {
		return flagFormat
	}
	if isTTY() {
		return "table"
	}
	return "json"
}

// renderList writes sessions in the requested (or auto) format.
func renderList(target string, ss []agents.Session) {
	switch pickFormat() {
	case "json":
		render.JSON(os.Stdout, target, ss)
	case "ndjson":
		render.NDJSON(os.Stdout, ss)
	default:
		render.List(os.Stdout, target, ss, flagAll)
	}
}

// filterAgent narrows ss to flagAgent when set.
func filterAgent(ss []agents.Session) []agents.Session {
	if flagAgent == "" {
		return ss
	}
	var f []agents.Session
	for _, s := range ss {
		if s.Tool == flagAgent {
			f = append(f, s)
		}
	}
	return f
}

// renderGlobal writes the global "most recent across all repos" view.
func renderGlobal(num int) {
	ss := filterAgent(agents.CollectGlobal(num))
	switch pickFormat() {
	case "json":
		render.JSON(os.Stdout, "", ss)
	case "ndjson":
		render.NDJSON(os.Stdout, ss)
	default:
		render.Recent(os.Stdout, ss)
	}
}

// loadFilter applies the user's config to the session filter. A bad config
// file is reported but non-fatal — seshy falls back to showing everything.
func loadFilter() {
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintln(os.Stderr, "seshy: ignoring bad config:", err)
		return
	}
	agents.Filter.HideClaudeHeadless = cfg.HideClaudeHeadless
	agents.Filter.HideCodexExec = cfg.HideCodexExec
}

// version is overridden at release time via -ldflags "-X main.version=<tag>".
var version = "0.1.0-dev"

func main() {
	root := &cobra.Command{
		Use:           "seshy [path]",
		Short:         "Recall & resume AI coding-agent sessions for a directory",
		Version:       version,
		Args:          cobra.MaximumNArgs(1),
		SilenceUsage:  true,
		SilenceErrors: true,
		PersistentPreRun: func(cmd *cobra.Command, args []string) {
			loadFilter()
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			target := resolveTarget(args)
			// Interactive picker when attached to a terminal and no explicit format.
			if flagFormat == "" && isTTY() {
				if err := tui.Run(target, flagNum, flagAll, flagAgent, false); err != nil {
					// Never leave the user stranded: fall back to the plain list.
					fmt.Fprintln(os.Stderr, "seshy: picker unavailable, listing instead:", err)
					renderList(target, collect(target))
				}
				return nil
			}
			renderList(target, collect(target))
			return nil
		},
	}
	root.PersistentFlags().StringVarP(&flagCwd, "cwd", "C", "", "target directory (default: cwd)")
	root.PersistentFlags().StringVarP(&flagFormat, "format", "o", "", "output: table|json|ndjson")
	root.PersistentFlags().BoolVar(&flagAll, "all", false, "include subdirectories")
	root.PersistentFlags().IntVarP(&flagNum, "num", "n", 10, "max sessions per agent")
	root.PersistentFlags().StringVar(&flagAgent, "agent", "", "filter to one agent")

	listCmd := &cobra.Command{
		Use:   "list [path]",
		Short: "List sessions (table for humans, JSON when piped)",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			target := resolveTarget(args)
			renderList(target, collect(target))
			return nil
		},
	}

	summaryCmd := &cobra.Command{
		Use:   "summary [path]",
		Short: "Compact digest of a project's sessions (agent-friendly)",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			target := resolveTarget(args)
			asJSON := flagFormat == "json" || (flagFormat == "" && !isTTY())
			render.Summary(os.Stdout, target, collect(target), asJSON)
			return nil
		},
	}

	lastCmd := &cobra.Command{
		Use:   "last [path]",
		Short: "Resume the most recent session in the directory",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			target := resolveTarget(args)
			ss := collect(target)
			if len(ss) == 0 {
				return fmt.Errorf("no agent sessions in %s", target)
			}
			return agents.Resume(ss[0])
		},
	}

	allCmd := &cobra.Command{
		Use:   "all",
		Short: "Most recent sessions across every repo on the machine",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			num := flagNum
			if !cmd.Flags().Changed("num") {
				num = 20 // a fuller default for the global view
			}
			if flagFormat == "" && isTTY() {
				if err := tui.Run("", num, false, flagAgent, true); err != nil {
					fmt.Fprintln(os.Stderr, "seshy: picker unavailable, listing instead:", err)
					renderGlobal(num)
				}
				return nil
			}
			renderGlobal(num)
			return nil
		},
	}

	sessionsCmd := &cobra.Command{
		Use:   "sessions",
		Short: "Every session across all agents and repos as JSON (an index for tooling)",
		Long: "Print every discovered session across all agents and directories as JSON — " +
			"path, agent, dir, mtime, id, and resume command — uncapped and without reading " +
			"file contents. A fast index for search skills and pipelines (use --agent to scope).",
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			ss := filterAgent(agents.CollectIndex())
			if flagFormat == "ndjson" {
				render.NDJSON(os.Stdout, ss)
			} else {
				render.JSON(os.Stdout, "", ss)
			}
			return nil
		},
	}

	var flagISearch, flagRegex bool
	searchCmd := &cobra.Command{
		Use:   "search <pattern> [path]",
		Short: "Search session contents across agents (excerpt + resume per hit)",
		Long: "Search the contents of past sessions across every agent for a pattern. " +
			"Global by default; pass a path (or -C) to scope to a directory. Prints a table " +
			"with an excerpt and resume command per hit, or JSON when piped (--format json).",
		Args: cobra.RangeArgs(1, 2),
		RunE: func(cmd *cobra.Command, args []string) error {
			o := agents.SearchOpts{
				Pattern:    args[0],
				Agent:      flagAgent,
				IgnoreCase: flagISearch,
				Regex:      flagRegex,
				Sub:        flagAll,
				Limit:      flagNum,
			}
			if !cmd.Flags().Changed("num") {
				o.Limit = 25
			}
			if len(args) > 1 {
				o.Dir = resolveTarget(args[1:])
			} else if flagCwd != "" {
				o.Dir = resolveTarget(nil)
			}
			matches, err := agents.Search(o)
			if err != nil {
				return err
			}
			render.Matches(os.Stdout, matches, pickFormat())
			return nil
		},
	}
	searchCmd.Flags().BoolVarP(&flagISearch, "ignore-case", "i", false, "case-insensitive match")
	searchCmd.Flags().BoolVar(&flagRegex, "regex", false, "treat the pattern as a regular expression")

	configCmd := &cobra.Command{
		Use:   "config",
		Short: "Show seshy settings (and where they live)",
		Long: "Show seshy's settings and config-file path. Use 'seshy config set " +
			"<key> <true|false>' to change one. Keys: hideClaudeHeadless (hide " +
			"claude -p / SDK sessions), hideCodexExec (hide `codex exec` sessions).",
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load()
			if err != nil {
				return err
			}
			fmt.Println("config:", config.Path())
			fmt.Printf("  hideClaudeHeadless = %t\n", cfg.HideClaudeHeadless)
			fmt.Printf("  hideCodexExec      = %t\n", cfg.HideCodexExec)
			return nil
		},
	}
	configSetCmd := &cobra.Command{
		Use:   "set <key> <true|false>",
		Short: "Set a config value (hideClaudeHeadless | hideCodexExec)",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load()
			if err != nil {
				return err
			}
			var b bool
			switch strings.ToLower(args[1]) {
			case "true", "1", "yes", "on":
				b = true
			case "false", "0", "no", "off":
				b = false
			default:
				return fmt.Errorf("value must be true or false, got %q", args[1])
			}
			switch args[0] {
			case "hideClaudeHeadless":
				cfg.HideClaudeHeadless = b
			case "hideCodexExec":
				cfg.HideCodexExec = b
			default:
				return fmt.Errorf("unknown key %q (want hideClaudeHeadless or hideCodexExec)", args[0])
			}
			if err := config.Save(cfg); err != nil {
				return err
			}
			fmt.Printf("set %s = %t  (%s)\n", args[0], b, config.Path())
			return nil
		},
	}
	configCmd.AddCommand(configSetCmd)

	root.AddCommand(listCmd, summaryCmd, lastCmd, allCmd, sessionsCmd, searchCmd, configCmd, wrappedCmd)
	if err := root.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}
