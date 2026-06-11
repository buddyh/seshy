// Package tui implements the interactive split-pane session picker (Bubble Tea v2).
package tui

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"syscall"
	"time"

	"charm.land/bubbles/v2/list"
	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"
	"charm.land/glamour/v2"
	"charm.land/lipgloss/v2"
	"github.com/buddyh/seshy/internal/agents"
	"github.com/buddyh/seshy/internal/config"
	"github.com/buddyh/seshy/internal/render"
)

func labelStyle(tool string) lipgloss.Style {
	hex := "#FFFFFF"
	if m, ok := agents.Metas[tool]; ok {
		hex = m.Hex
	}
	return lipgloss.NewStyle().Foreground(lipgloss.Color(hex)).Bold(true)
}

func short(id string) string {
	if len(id) > 8 {
		return id[:8]
	}
	return id
}

func truncate(s string, w int) string {
	if w <= 1 {
		return ""
	}
	r := []rune(s)
	if len(r) <= w {
		return s
	}
	return string(r[:w-1]) + "…"
}

// ---- list item ----

type sessionItem struct{ s agents.Session }

func (i sessionItem) FilterValue() string {
	return agents.Label(i.s.Tool) + " " + filepath.Base(i.s.Dir) + " " + i.s.Preview + " " + i.s.ID
}

// ---- delegate (one compact glowing row) ----

type delegate struct{ global bool }

func (d delegate) Height() int                         { return 1 }
func (d delegate) Spacing() int                        { return 0 }
func (d delegate) Update(tea.Msg, *list.Model) tea.Cmd { return nil }

// column widths shared by every row so the preview always starts flush.
const (
	colAge = 8 // right-aligned relative age
	colLbl = 8 // agent label
)

// ageColor tints the age column by recency: fresh sessions glow, stale ones fade.
func ageColor(t time.Time) string {
	switch d := time.Since(t); {
	case d < time.Hour:
		return swCyan
	case d < 24*time.Hour:
		return swAqua
	case d < 7*24*time.Hour:
		return swLav
	default:
		return swFog
	}
}

func (d delegate) idWidth() int {
	if d.global {
		return 18 // repo basename across repos
	}
	return 8 // short session id within one dir
}

func (d delegate) Render(w io.Writer, m list.Model, index int, item list.Item) {
	it, ok := item.(sessionItem)
	if !ok {
		return
	}
	s := it.s
	width := m.Width()
	idW := d.idWidth()

	age := fmt.Sprintf("%*s", colAge, render.HumanAge(s.Mtime))
	lbl := fmt.Sprintf("%-*s", colLbl, agents.Label(s.Tool))
	idRaw := short(s.ID)
	if d.global {
		idRaw = filepath.Base(s.Dir) // show the repo, not the id, across repos
	}
	id := fmt.Sprintf("%-*s", idW, truncate(idRaw, idW))

	prev := s.Preview
	if prev == "" {
		prev = "—"
	}
	// layout: lead(2) age(colAge) gap(2) lbl(colLbl) gap(1) id(idW) gap(2) preview
	prefixW := 2 + colAge + 2 + colLbl + 1 + idW + 2
	prevW := max(4, width-prefixW)
	prev = truncate(prev, prevW)

	if index == m.Index() {
		bg := lipgloss.Color(swSel)
		content := fmt.Sprintf(" %s  %s %s  %s",
			stSelAge.Render(age),
			labelStyle(s.Tool).Background(bg).Render(lbl),
			stSelDim.Render(id),
			stSelText.Render(prev))
		used := 2 + colAge + 2 + colLbl + 1 + idW + 2 + len([]rune(prev))
		if pad := width - used; pad > 0 {
			content += stSelRow.Render(strings.Repeat(" ", pad))
		}
		fmt.Fprint(w, stSelBar.Render("▌")+content)
		return
	}
	fmt.Fprintf(w, "  %s  %s %s  %s",
		lipgloss.NewStyle().Foreground(lipgloss.Color(ageColor(s.Mtime))).Render(age),
		labelStyle(s.Tool).Render(lbl), stID.Render(id), stPrev.Render(prev))
}

// ---- model ----

type model struct {
	target      string
	list        list.Model
	vp          viewport.Model
	w, h        int
	phase       int
	ready       bool
	showPreview bool
	global      bool
	lastSel     string
	chosen      *agents.Session

	// Collection params, kept so the `h` key can re-collect with a flipped filter.
	// (pageSize below doubles as the per-agent / first-page count.)
	allSub       bool
	agent        string
	hideHeadless bool

	// Paged loading for the global view: all holds the full newest-first index
	// (previews unfilled past shown); the list grows a page at a time as the
	// cursor nears the end so we only read previews for sessions actually viewed.
	all      []agents.Session
	shown    int
	pageSize int
}

type tickMsg time.Time

func tick() tea.Cmd {
	return tea.Tick(time.Second/30, func(t time.Time) tea.Msg { return tickMsg(t) })
}

const shimmerStep = 1 // gradient columns advanced per tick

func (m model) Init() tea.Cmd { return tick() }

func (m model) bigHeader() bool { return m.h >= 16 && m.w >= figW+2 }

func (m model) headerHeight() int {
	if m.bigHeader() {
		return bannerHeight() + 2 // subtitle + neon rule
	}
	return 2 // logo line + rule
}

func (m model) headerView() string {
	path := m.target
	if m.global {
		path = "most recent · all repositories"
		if len(m.all) > 0 && m.shown < len(m.all) {
			path += fmt.Sprintf(" · showing %d of %d", m.shown, len(m.all))
		}
	}
	div := rule(m.w, m.phase)
	if m.bigHeader() {
		return lipgloss.JoinVertical(lipgloss.Left,
			banner(m.phase),
			stPath.Render(path),
			div)
	}
	return lipgloss.JoinVertical(lipgloss.Left,
		logoLine(m.phase)+"  "+stPath.Render(path),
		div)
}

// bevelPanel is a rounded box with a raised look: lit top/left edges, dark
// bottom/right edges (a cast-shadow bevel). The focused panel gets a brighter
// cyan edge so the active pane is obvious. w,h are CONTENT dimensions.
func bevelPanel(w, h int, focused bool) lipgloss.Style {
	edge := lipgloss.Color(swViolet)
	if focused {
		edge = lipgloss.Color(swCyan)
	}
	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderTopForeground(edge).
		BorderLeftForeground(edge).
		BorderRightForeground(lipgloss.Color(swEdge)).
		BorderBottomForeground(lipgloss.Color(swEdge)).
		Width(w).Height(h)
}

func footerBar(w int, right string, hideHeadless bool) string {
	sep := stFog.Render("  ")
	headless := stKey.Render("h") + " " + stMuted.Render("headless")
	if hideHeadless {
		headless = stKey.Render("h") + " " + stCyan.Render("headless hidden")
	}
	hints := strings.Join([]string{
		stKey.Render("↑↓") + " " + stMuted.Render("move"),
		stKey.Render("/") + " " + stMuted.Render("filter"),
		stKey.Render("enter") + " " + stMuted.Render("resume"),
		stKey.Render("p") + " " + stMuted.Render("preview"),
		headless,
		stKey.Render("q") + " " + stMuted.Render("quit"),
	}, sep)
	bar := lipgloss.NewStyle().Background(lipgloss.Color(swSurface))
	left := " " + hints
	if right == "" {
		return bar.Width(w).Render(left)
	}
	r := stFog.Background(lipgloss.Color(swSurface)).Render(right + " ")
	gap := w - lipgloss.Width(left) - lipgloss.Width(r)
	if gap < 1 {
		return bar.Width(w).Render(left)
	}
	return bar.Render(left) + bar.Render(strings.Repeat(" ", gap)) + bar.Render(r)
}

// panelInnerH is the content height inside the body panels (minus the footer bar).
func (m model) panelInnerH() int {
	h := m.h - m.headerHeight() - 1 - 2 // footer(1) + panel border(2)
	if h < 1 {
		h = 1
	}
	return h
}

func (m *model) layout() {
	innerH := m.panelInnerH()
	if m.showPreview {
		listTotal := m.w * 44 / 100
		if listTotal < 26 {
			listTotal = 26
		}
		prevTotal := m.w - listTotal
		m.list.SetSize(listTotal-2, innerH)
		m.vp.SetWidth(prevTotal - 2)
		m.vp.SetHeight(innerH)
	} else {
		m.list.SetSize(m.w-2, innerH)
	}
}

// appendPage fills previews for the next pageSize sessions and appends them to
// the list. Returns nil when everything is already shown.
func (m *model) appendPage() tea.Cmd {
	if m.shown >= len(m.all) {
		return nil
	}
	end := m.shown + m.pageSize
	if end > len(m.all) {
		end = len(m.all)
	}
	batch := m.all[m.shown:end] // FillPreviews mutates the backing array in place
	agents.FillPreviews(batch)
	items := m.list.Items()
	for i := range batch {
		items = append(items, sessionItem{batch[i]})
	}
	m.shown = end
	return m.list.SetItems(items)
}

// collectSessions gathers the session list for the model's current scope and
// the active agents.Filter (newest-first). Mirrors the selection logic in Run
// so the `h` toggle re-collects identically.
func (m *model) collectSessions() []agents.Session {
	var ss []agents.Session
	if m.global {
		ss = agents.CollectIndex()
	} else {
		ss = agents.Collect(m.target, m.pageSize, m.allSub)
	}
	if m.agent != "" {
		f := ss[:0]
		for _, s := range ss {
			if s.Tool == m.agent {
				f = append(f, s)
			}
		}
		ss = f
	}
	sort.SliceStable(ss, func(a, b int) bool { return ss[a].Mtime.After(ss[b].Mtime) })
	return ss
}

// seedItems installs ss as the list, filling previews for the first page only
// (the global view pages the rest in lazily as the cursor moves).
func (m *model) seedItems(ss []agents.Session) tea.Cmd {
	m.all = ss
	first := len(ss)
	if m.global && m.pageSize < first {
		first = m.pageSize
	}
	seed := ss[:first]
	if m.global {
		agents.FillPreviews(seed)
	}
	items := make([]list.Item, first)
	for i := range seed {
		items[i] = sessionItem{seed[i]}
	}
	m.shown = first
	m.lastSel = ""
	return m.list.SetItems(items)
}

// toggleHeadless flips hiding of headless/automated sessions (Claude -p / SDK
// and Codex exec), persists it to config, and re-collects with the new filter.
func (m *model) toggleHeadless() tea.Cmd {
	m.hideHeadless = !m.hideHeadless
	agents.Filter.HideClaudeHeadless = m.hideHeadless
	agents.Filter.HideCodexExec = m.hideHeadless
	if cfg, err := config.Load(); err == nil { // best-effort persist
		cfg.HideClaudeHeadless = m.hideHeadless
		cfg.HideCodexExec = m.hideHeadless
		config.Save(cfg)
	}
	cmd := m.seedItems(m.collectSessions())
	for m.shown < m.panelInnerH()+m.pageSize && m.shown < len(m.all) {
		m.appendPage()
	}
	return cmd
}

func (m *model) syncPreview() {
	it, ok := m.list.SelectedItem().(sessionItem)
	if !ok {
		m.vp.SetContent(stFog.Render("  no session"))
		return
	}
	if it.s.ID == m.lastSel {
		return
	}
	m.lastSel = it.s.ID
	m.vp.SetContent(buildPreview(it.s, m.vp.Width()-2))
	m.vp.GotoTop()
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tickMsg:
		m.phase += shimmerStep
		return m, tick()
	case tea.WindowSizeMsg:
		m.w, m.h = msg.Width, msg.Height
		m.ready = true
		m.layout()
		m.lastSel = ""
		// Load enough pages to cover the visible area on first layout / resize.
		for m.shown < m.panelInnerH()+m.pageSize && m.shown < len(m.all) {
			m.appendPage()
		}
	case tea.KeyPressMsg:
		if m.list.FilterState() != list.Filtering {
			switch msg.String() {
			case "q", "ctrl+c":
				return m, tea.Quit
			case "enter":
				if it, ok := m.list.SelectedItem().(sessionItem); ok {
					c := it.s
					m.chosen = &c
				}
				return m, tea.Quit
			case "p":
				m.showPreview = !m.showPreview
				m.layout()
				m.lastSel = ""
			case "h":
				return m, m.toggleHeadless()
			case "ctrl+d":
				m.vp.HalfPageDown()
			case "ctrl+u":
				m.vp.HalfPageUp()
			}
		}
	}
	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	cmds := []tea.Cmd{cmd}
	// Lazily load the next page as the cursor nears the end of what's loaded.
	if m.list.FilterState() != list.Filtering && m.list.Index() >= m.shown-2 {
		if c := m.appendPage(); c != nil {
			cmds = append(cmds, c)
		}
	}
	if m.ready && m.showPreview {
		m.syncPreview()
	}
	return m, tea.Batch(cmds...)
}

func (m model) View() tea.View {
	if !m.ready {
		return tea.NewView(logoLine(0) + stMuted.Render("  loading…"))
	}
	header := m.headerView()
	ph := m.panelInnerH() + 2 // total panel box height (lipgloss is border-box)

	var body string
	if m.showPreview {
		body = lipgloss.JoinHorizontal(lipgloss.Top,
			bevelPanel(m.list.Width()+2, ph, true).Render(m.list.View()),
			bevelPanel(m.vp.Width()+2, ph, false).Render(m.vp.View()))
	} else {
		body = bevelPanel(m.w, ph, true).Render(m.list.View())
	}

	right := ""
	if n := len(m.list.Items()); n > 0 {
		right = fmt.Sprintf("%d/%d", m.list.Index()+1, n)
	}
	v := tea.NewView(lipgloss.JoinVertical(lipgloss.Left, header, body, footerBar(m.w, right, m.hideHeadless)))
	v.AltScreen = true
	return v
}

// ---- preview content ----

func renderMarkdown(md string, width int) string {
	if width < 10 {
		width = 10
	}
	r, err := glamour.NewTermRenderer(glamour.WithStandardStyle("dark"), glamour.WithWordWrap(width))
	if err != nil {
		return md
	}
	out, err := r.Render(md)
	if err != nil {
		return md
	}
	return out
}

// keyVal renders an aligned "  label   value" metadata line.
func keyVal(label, value string) string {
	return stFog.Render(fmt.Sprintf("  %-7s ", label)) + value
}

func buildPreview(s agents.Session, width int) string {
	if width < 10 {
		width = 10
	}
	d := agents.LoadDetail(s)
	var b strings.Builder

	// Header: agent chip + relative age, then a dim divider.
	b.WriteString(" " + labelStyle(s.Tool).Render(agents.Label(s.Tool)) + "  " +
		stPink.Render(render.HumanAge(s.Mtime)) + "\n")
	b.WriteString(stFog.Render(strings.Repeat("─", width)) + "\n")

	// The hero action: a copy-ready resume command in a code-chip.
	cmd := strings.Join(agents.ResumeArgv(s), " ")
	b.WriteString(keyVal("resume", stResumeCmd.Render(" "+truncate(cmd, max(8, width-12))+" ")) + "\n")
	b.WriteString(keyVal("dir", stMuted.Render(truncate(s.Dir, max(8, width-10)))) + "\n")
	b.WriteString(keyVal("id", stFog.Render(truncate(s.ID, max(8, width-10)))) + "\n")

	// Badge row: model · cost · turns · tasks, each in its own accent.
	var badges []string
	if s.Model != "" {
		badges = append(badges, stCyan.Render(s.Model))
	}
	if s.Cost > 0 {
		badges = append(badges, stYellow.Render(fmt.Sprintf("$%.2f", s.Cost)))
	}
	if d.Turns > 0 {
		badges = append(badges, stMuted.Render(fmt.Sprintf("%d turns", d.Turns)))
	} else if s.Msgs > 0 {
		badges = append(badges, stMuted.Render(fmt.Sprintf("%d msgs", s.Msgs)))
	}
	if d.TaskTotal > 0 {
		badges = append(badges, lipgloss.NewStyle().Foreground(lipgloss.Color(swCyan)).
			Render(fmt.Sprintf("✓ %d/%d tasks", d.TaskDone, d.TaskTotal)))
	}
	if len(badges) > 0 {
		b.WriteString("  " + strings.Join(badges, stFog.Render("  ·  ")) + "\n")
	}
	b.WriteString("\n")

	if d.FirstPrompt != "" {
		b.WriteString(section("first prompt") + "\n")
		b.WriteString(wrap(d.FirstPrompt, width) + "\n\n")
	}
	if d.LastMessage != "" {
		head := "last message"
		if d.LastRole != "" {
			head = "last · " + d.LastRole
		}
		b.WriteString(section(head) + "\n")
		b.WriteString(wrap(d.LastMessage, width) + "\n")
	}
	if d.TaskMD != "" {
		b.WriteString(section("tasks") + "\n")
		b.WriteString(renderMarkdown(d.TaskMD, width))
	}
	return b.String()
}

// section renders a uppercased section header with the neon bar.
func section(label string) string {
	return stSelBar.Render("▌") + " " + stSecHead.Render(strings.ToUpper(label))
}

func wrap(s string, width int) string {
	return lipgloss.NewStyle().Width(width).Foreground(lipgloss.Color("#E6DCFF")).Render(s)
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// ---- entry ----

// Run shows the picker; on Enter it execs the chosen session's resume command.
// When global, it lists the most-recent sessions across every repo instead of
// just target, and rows show the repo rather than the session id.
func Run(target string, num int, all bool, agent string, global bool) error {
	if num < 1 {
		num = 1
	}
	l := list.New(nil, delegate{global: global}, 40, 20)
	l.SetShowTitle(false)
	l.SetShowStatusBar(false)
	l.SetShowHelp(false)
	l.SetFilteringEnabled(true)
	l.FilterInput.Prompt = "filter › "

	m := model{
		target: target, vp: viewport.New(), global: global, list: l,
		allSub: all, agent: agent, pageSize: num,
		hideHeadless: agents.Filter.HideClaudeHeadless || agents.Filter.HideCodexExec,
	}
	// collectSessions reads the active agents.Filter (seeded from config); the `h`
	// key flips it and re-collects. The first page's previews are filled here, the
	// rest paged in as the cursor moves (global view only).
	m.seedItems(m.collectSessions())

	final, err := tea.NewProgram(m).Run()
	if err != nil {
		return err
	}
	fm, ok := final.(model)
	if !ok || fm.chosen == nil {
		return nil
	}
	argv := agents.ResumeArgv(*fm.chosen)
	if argv == nil {
		return fmt.Errorf("no resume command for %s", fm.chosen.Tool)
	}
	os.Chdir(fm.chosen.Dir)
	bin, err := exec.LookPath(argv[0])
	if err != nil {
		fmt.Println(strings.Join(argv, " "))
		return nil
	}
	return syscall.Exec(bin, argv, os.Environ())
}
