package stage

import (
	"strings"
	"time"

	"github.com/MagikIO/familiar-says/internal/bubble"
	"github.com/MagikIO/familiar-says/internal/canvas"
	"github.com/mattn/go-runewidth"
)

// Bubble is a speech bubble that types its text out and then lingers long
// enough to be read.
type Bubble struct {
	Template *bubble.BubbleTemplate
	Border   Paint
	Ink      Paint

	lines []string // wrapped text, full
	total int      // runes across all lines
	shown float64  // runes revealed so far
	cps   float64  // typing speed, runes per second
	hold  time.Duration
	held  time.Duration
}

// BubbleOptions tunes a new bubble.
type BubbleOptions struct {
	Style string        // template name: say, think, shout, whisper, song, ...
	Width int           // wrap width of the text
	CPS   float64       // typing speed; <= 0 shows the text immediately
	Hold  time.Duration // how long to stay once typed; 0 picks a reading time
}

// NewBubble lays out text in the named template.
func NewBubble(text string, opt BubbleOptions, border, ink Paint) *Bubble {
	tmpl := bubble.GetTemplate(opt.Style)
	if opt.Width < 8 {
		opt.Width = 8
	}
	text = tmpl.Prefix + strings.TrimSpace(text) + tmpl.Suffix
	lines := canvas.WrapText(text, opt.Width)
	if len(lines) == 0 {
		lines = []string{""}
	}
	b := &Bubble{Template: tmpl, Border: border, Ink: ink, lines: lines, cps: opt.CPS, hold: opt.Hold}
	for _, l := range lines {
		b.total += len([]rune(l))
	}
	if b.cps <= 0 {
		b.shown = float64(b.total)
	}
	if b.hold <= 0 {
		b.hold = ReadingTime(b.total)
	}
	return b
}

// ReadingTime estimates how long a line of n runes needs to stay on screen.
func ReadingTime(n int) time.Duration {
	d := 2500*time.Millisecond + time.Duration(n)*55*time.Millisecond
	if d > 14*time.Second {
		d = 14 * time.Second
	}
	return d
}

// Lines reports how many rows of text the bubble holds.
func (b *Bubble) Lines() int { return len(b.lines) }

// Size returns the bubble's outer dimensions in cells.
func (b *Bubble) Size() (w, h int) {
	framed, _ := canvas.FrameBubble(b.lines, b.Template)
	for _, l := range framed {
		if lw := runewidth.StringWidth(l); lw > w {
			w = lw
		}
	}
	return w, len(framed)
}

// Typing reports whether text is still being revealed.
func (b *Bubble) Typing() bool { return int(b.shown) < b.total }

// Done reports whether the bubble has been fully shown for its hold time.
func (b *Bubble) Done() bool { return !b.Typing() && b.held >= b.hold }

// Held reports how long the fully typed bubble has been on screen.
func (b *Bubble) Held() time.Duration { return b.held }

// Skip reveals all remaining text at once.
func (b *Bubble) Skip() { b.shown = float64(b.total) }

// Dismiss ends the bubble immediately.
func (b *Bubble) Dismiss() {
	b.Skip()
	b.held = b.hold
}

// Step advances typing and the hold timer.
func (b *Bubble) Step(dt time.Duration) {
	if b.Typing() {
		b.shown += b.cps * dt.Seconds()
		if b.shown > float64(b.total) {
			b.shown = float64(b.total)
		}
		return
	}
	b.held += dt
}

// Grid renders the bubble at its current point in the reveal. The frame is
// always laid out for the full text, so the bubble never resizes mid-sentence.
func (b *Bubble) Grid() *Grid {
	framed, contentX := canvas.FrameBubble(b.lines, b.Template)
	g := FromLines(framed, 0, func(int, int) Paint { return b.Border })

	left := int(b.shown)
	for i, line := range b.lines {
		y := i + 1
		col := contentX
		for _, r := range line {
			w := runewidth.RuneWidth(r)
			if w <= 0 {
				continue
			}
			if left > 0 {
				g.Text(col, y, string(r), b.Ink)
			} else {
				for k := 0; k < w; k++ {
					g.Set(col+k, y, ' ', b.Ink)
				}
			}
			left--
			col += w
		}
	}
	return g
}
