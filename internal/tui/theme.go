package tui

import (
	"image/color"
	"strings"

	"charm.land/lipgloss/v2"
)

// Synthwave / outrun palette.
const (
	swPink    = "#FF6AC1"
	swMagenta = "#F92AAD"
	swPurple  = "#B026FF"
	swViolet  = "#9D4EDD"
	swCyan    = "#2DE2E6"
	swAqua    = "#05D9E8"
	swIndigo  = "#2B213A"
	swSurface = "#241B2F" // footer / chrome backdrop
	swSel     = "#37265B" // selected-row backdrop (clearly above terminal black)
	swOrange  = "#FF8E42"
	swYellow  = "#FFD319"
	swLav     = "#B8A6D9"
	swFog     = "#6C5C8A"
	swInk     = "#F5EEFF" // brightest foreground (selected text)
	swEdge    = "#3A1C63" // dark bevel edge (bottom/right borders)
)

var (
	stPath    = lipgloss.NewStyle().Foreground(lipgloss.Color(swAqua)).Italic(true)
	stMuted   = lipgloss.NewStyle().Foreground(lipgloss.Color(swLav))
	stFog     = lipgloss.NewStyle().Foreground(lipgloss.Color(swFog))
	stCyan    = lipgloss.NewStyle().Foreground(lipgloss.Color(swCyan))
	stPink    = lipgloss.NewStyle().Foreground(lipgloss.Color(swPink))
	stYellow  = lipgloss.NewStyle().Foreground(lipgloss.Color(swYellow))
	stSecHead = lipgloss.NewStyle().Foreground(lipgloss.Color(swPink)).Bold(true)
	stKey     = lipgloss.NewStyle().Foreground(lipgloss.Color(swCyan)).Bold(true)

	// Selected-row treatment: a bright bar, a raised backdrop, and accented columns
	// so the cursor is unmistakable and its preview pops above the rest.
	stSelBar  = lipgloss.NewStyle().Foreground(lipgloss.Color(swCyan)).Bold(true)
	stSelRow  = lipgloss.NewStyle().Background(lipgloss.Color(swSel))
	stSelText = lipgloss.NewStyle().Background(lipgloss.Color(swSel)).Foreground(lipgloss.Color(swInk)).Bold(true)
	stSelAge  = lipgloss.NewStyle().Background(lipgloss.Color(swSel)).Foreground(lipgloss.Color(swCyan))
	stSelDim  = lipgloss.NewStyle().Background(lipgloss.Color(swSel)).Foreground(lipgloss.Color(swLav))

	// Unselected columns: the repo/id reads as structure, the preview stays
	// legible — focus comes from the contrast against the selected row. (The age
	// column is tinted per-row by recency; see ageColor.)
	stID   = lipgloss.NewStyle().Foreground(lipgloss.Color(swFog))
	stPrev = lipgloss.NewStyle().Foreground(lipgloss.Color(swLav))

	// Resume command rendered as a copy-ready code chip in the preview card.
	stResumeCmd = lipgloss.NewStyle().
			Foreground(lipgloss.Color(swCyan)).
			Background(lipgloss.Color(swSurface)).
			Bold(true)
)

// neon stop sequence (loops back so a horizontal scroll is seamless).
func neonStops() []color.Color {
	return []color.Color{
		lipgloss.Color(swMagenta), lipgloss.Color(swPink), lipgloss.Color(swOrange),
		lipgloss.Color(swYellow), lipgloss.Color(swCyan), lipgloss.Color(swAqua),
		lipgloss.Color(swViolet), lipgloss.Color(swPurple), lipgloss.Color(swMagenta),
	}
}

// neonRamp returns n colors blended across the looping neon stops.
func neonRamp(n int) []color.Color {
	if n < 2 {
		n = 2
	}
	return lipgloss.Blend1D(n, neonStops()...)
}

// ---- big animated banner ----

var (
	figLines = strings.Split(figArt, "\n")
	figW     = func() int {
		w := 0
		for _, l := range figLines {
			if n := len([]rune(l)); n > w {
				w = n
			}
		}
		return w
	}()
)

// castShadow draws an offset dark ghost behind the wordmark. Off for fonts that
// are already dimensional (isometric), where it would clutter the geometry.
const castShadow = false

func bannerHeight() int {
	if castShadow {
		return len(figLines) + 1
	}
	return len(figLines)
}

var shadowColor = lipgloss.Color("#160B28") // deep cast shadow

func glyphAt(y, x int) rune {
	if y < 0 || y >= len(figLines) {
		return ' '
	}
	r := []rune(figLines[y])
	if x < 0 || x >= len(r) {
		return ' '
	}
	return r[x]
}

// embossShade shades a block face with a smooth top-lit → dark vertical depth ramp.
func embossShade(base color.Color, y, h int) color.Color {
	if h <= 1 {
		return base
	}
	f := float64(y) / float64(h-1) // 0 at top .. 1 at base
	amt := 0.48 - 0.92*f           // +sheen on top, −shadow at the base
	if amt >= 0 {
		return lipgloss.Lighten(base, amt)
	}
	return lipgloss.Darken(base, -amt)
}

// banner renders the ANSI Shadow wordmark: solid block faces (█) get the bright,
// top-lit neon gradient (scrolling by phase); the box-drawing shadow corners get a
// darkened bevel — giving the heavy 3D depth without an extra cast shadow.
func banner(phase int) string {
	h := len(figLines)
	ramp := neonRamp(figW * 2)
	rl := len(ramp)
	face := lipgloss.NewStyle().Bold(true)
	var b strings.Builder
	for y := 0; y < h; y++ {
		for x := 0; x < figW; x++ {
			r := glyphAt(y, x)
			if r == ' ' {
				b.WriteByte(' ')
				continue
			}
			base := ramp[((x+phase)%rl+rl)%rl]
			var c color.Color
			if r == '█' {
				c = embossShade(base, y, h) // lit block face
			} else {
				c = lipgloss.Darken(base, 0.58) // beveled shadow corner
			}
			b.WriteString(face.Foreground(c).Render(string(r)))
		}
		if y < h-1 {
			b.WriteByte('\n')
		}
	}
	return b.String()
}

// logoLine is the compact one-row wordmark for short terminals.
func logoLine(phase int) string {
	word := []rune("✦ seshy ✦")
	ramp := neonRamp(len(word) * 2)
	rl := len(ramp)
	var b strings.Builder
	for i, r := range word {
		c := ramp[((i+phase)%rl+rl)%rl]
		b.WriteString(lipgloss.NewStyle().Foreground(c).Bold(true).Render(string(r)))
	}
	return b.String()
}

// rule draws an animated neon horizon line.
func rule(width, phase int) string {
	if width < 1 {
		return ""
	}
	ramp := neonRamp(width * 2)
	rl := len(ramp)
	var b strings.Builder
	for x := 0; x < width; x++ {
		c := lipgloss.Darken(ramp[((x+phase)%rl+rl)%rl], 0.15)
		b.WriteString(lipgloss.NewStyle().Foreground(c).Render("─"))
	}
	return b.String()
}
