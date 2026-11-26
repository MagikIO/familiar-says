// Package stage is the real-time rendering layer for the live familiar.
//
// Where package canvas composes a single static picture out of lipgloss-styled
// cells, stage is built for redrawing many times a second: cells carry a small
// comparable Paint instead of a full lipgloss.Style, and rendering coalesces
// runs of identically painted cells so a frame costs a handful of style calls
// rather than one per cell.
package stage

import (
	"strings"
	"sync"

	"github.com/charmbracelet/lipgloss"
	"github.com/mattn/go-runewidth"
)

// cont marks the second cell of a double-width rune.
const cont rune = -1

// Paint is the visual style of a cell. It is deliberately tiny and comparable
// so that adjacent cells can be merged into one styled run.
type Paint struct {
	FG   string // "", "#RRGGBB", or an ANSI 256 index such as "213"
	Bold bool
	Dim  bool
}

// Cell is one character position. A zero rune means "transparent".
type Cell struct {
	R rune
	P Paint
}

// Grid is a W×H buffer of cells.
type Grid struct {
	W, H  int
	cells []Cell
}

// NewGrid returns a fully transparent grid. Non-positive sizes are clamped to 1.
func NewGrid(w, h int) *Grid {
	if w < 1 {
		w = 1
	}
	if h < 1 {
		h = 1
	}
	return &Grid{W: w, H: h, cells: make([]Cell, w*h)}
}

func (g *Grid) in(x, y int) bool { return x >= 0 && y >= 0 && x < g.W && y < g.H }

// At returns the cell at (x, y); out-of-range reads are transparent.
func (g *Grid) At(x, y int) Cell {
	if !g.in(x, y) {
		return Cell{}
	}
	return g.cells[y*g.W+x]
}

// Set writes a single-width rune. Out-of-range writes are ignored.
func (g *Grid) Set(x, y int, r rune, p Paint) {
	if !g.in(x, y) {
		return
	}
	i := y*g.W + x
	// Overwriting half of a wide rune would leave a dangling half behind.
	if g.cells[i].R == cont && x > 0 {
		g.cells[i-1] = Cell{R: ' ', P: g.cells[i-1].P}
	}
	if x+1 < g.W && g.cells[i+1].R == cont {
		g.cells[i+1] = Cell{R: ' ', P: g.cells[i+1].P}
	}
	g.cells[i] = Cell{R: r, P: p}
}

// Text draws s starting at (x, y) and returns the number of columns used.
// Double-width runes take two cells; zero-width runes are dropped.
func (g *Grid) Text(x, y int, s string, p Paint) int {
	col := x
	for _, r := range s {
		w := runewidth.RuneWidth(r)
		switch {
		case w <= 0:
			continue
		case w == 1:
			g.Set(col, y, r, p)
		default:
			if g.in(col, y) && g.in(col+1, y) {
				g.Set(col, y, r, p)
				g.Set(col+1, y, ' ', p)
				g.cells[y*g.W+col+1].R = cont
			} else {
				g.Set(col, y, ' ', p)
				g.Set(col+1, y, ' ', p)
			}
		}
		col += w
	}
	return col - x
}

// Blit draws src onto g with its top-left corner at (x, y). Transparent source
// cells leave g untouched.
func (g *Grid) Blit(src *Grid, x, y int) {
	if src == nil {
		return
	}
	for sy := 0; sy < src.H; sy++ {
		dy := y + sy
		if dy < 0 || dy >= g.H {
			continue
		}
		for sx := 0; sx < src.W; sx++ {
			c := src.cells[sy*src.W+sx]
			if c.R == 0 {
				continue
			}
			dx := x + sx
			if dx < 0 || dx >= g.W {
				continue
			}
			if c.R == cont {
				g.cells[dy*g.W+dx] = c
				continue
			}
			g.Set(dx, dy, c.R, c.P)
		}
	}
}

// mirror maps glyphs to their horizontal reflection so flipped art still reads.
var mirror = map[rune]rune{
	'/': '\\', '\\': '/',
	'(': ')', ')': '(',
	'<': '>', '>': '<',
	'[': ']', ']': '[',
	'{': '}', '}': '{',
	'd': 'b', 'b': 'd',
	'p': 'q', 'q': 'p',
	'`': '\'', '\'': '`',
	'╱': '╲', '╲': '╱',
}

// Flip returns a horizontally mirrored copy of the grid.
func (g *Grid) Flip() *Grid {
	out := NewGrid(g.W, g.H)
	for y := 0; y < g.H; y++ {
		row := out.cells[y*g.W : (y+1)*g.W]
		for x := 0; x < g.W; x++ {
			c := g.cells[y*g.W+x]
			if m, ok := mirror[c.R]; ok {
				c.R = m
			}
			row[g.W-1-x] = c
		}
		// A reversed wide rune now has its continuation on the wrong side.
		for x := 0; x+1 < g.W; x++ {
			if row[x].R == cont {
				row[x], row[x+1] = row[x+1], row[x]
				x++
			}
		}
	}
	return out
}

var (
	styleMu    sync.Mutex
	styleCache = map[Paint]lipgloss.Style{}
)

func styleFor(p Paint) lipgloss.Style {
	styleMu.Lock()
	defer styleMu.Unlock()
	if s, ok := styleCache[p]; ok {
		return s
	}
	s := lipgloss.NewStyle()
	if p.FG != "" {
		s = s.Foreground(lipgloss.Color(p.FG))
	}
	if p.Bold {
		s = s.Bold(true)
	}
	if p.Dim {
		s = s.Faint(true)
	}
	styleCache[p] = s
	return s
}

// ResetStyleCache drops cached styles. Call it after changing the lipgloss
// color profile so runs are re-rendered for the new profile.
func ResetStyleCache() {
	styleMu.Lock()
	styleCache = map[Paint]lipgloss.Style{}
	styleMu.Unlock()
}

// Lines renders the grid as one styled string per row. Trailing blank cells
// are trimmed so idle rows cost nothing to transmit.
func (g *Grid) Lines() []string {
	lines := make([]string, g.H)
	var sb, run strings.Builder
	for y := 0; y < g.H; y++ {
		row := g.cells[y*g.W : (y+1)*g.W]
		end := len(row)
		for end > 0 && (row[end-1].R == 0 || row[end-1].R == ' ') {
			end--
		}
		sb.Reset()
		run.Reset()
		var cur Paint
		flush := func() {
			if run.Len() == 0 {
				return
			}
			if cur == (Paint{}) {
				sb.WriteString(run.String())
			} else {
				sb.WriteString(styleFor(cur).Render(run.String()))
			}
			run.Reset()
		}
		for x := 0; x < end; x++ {
			c := row[x]
			if c.R == cont {
				continue
			}
			r, p := c.R, c.P
			if r == 0 || r == ' ' {
				// Blank cells never need colour; folding them into the
				// surrounding run keeps escape sequences to a minimum.
				r, p = ' ', cur
			}
			if p != cur {
				flush()
				cur = p
			}
			run.WriteRune(r)
		}
		flush()
		lines[y] = sb.String()
	}
	return lines
}

// String renders the whole grid, rows joined by newlines.
func (g *Grid) String() string { return strings.Join(g.Lines(), "\n") }

// Plain renders the grid without any styling (useful for tests and logs).
func (g *Grid) Plain() []string {
	lines := make([]string, g.H)
	var sb strings.Builder
	for y := 0; y < g.H; y++ {
		sb.Reset()
		for x := 0; x < g.W; x++ {
			c := g.cells[y*g.W+x]
			switch c.R {
			case cont:
			case 0:
				sb.WriteByte(' ')
			default:
				sb.WriteRune(c.R)
			}
		}
		lines[y] = strings.TrimRight(sb.String(), " ")
	}
	return lines
}

// FromLines builds a grid from text rows. Leading and trailing spaces on each
// row stay transparent so the art keeps its silhouette, while interior spaces
// are opaque so whatever is behind the art does not show through its body.
func FromLines(lines []string, width int, paint func(x, y int) Paint) *Grid {
	for _, l := range lines {
		if w := runewidth.StringWidth(l); w > width {
			width = w
		}
	}
	g := NewGrid(width, len(lines))
	for y, l := range lines {
		trimmed := strings.TrimRight(l, " ")
		lead := len(trimmed) - len(strings.TrimLeft(trimmed, " "))
		col := 0
		for i, r := range trimmed {
			w := runewidth.RuneWidth(r)
			if w <= 0 {
				continue
			}
			if i >= lead {
				p := Paint{}
				if paint != nil {
					p = paint(col, y)
				}
				g.Text(col, y, string(r), p)
			}
			col += w
		}
	}
	return g
}
