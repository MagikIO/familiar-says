// Package canvas provides a 2D grid-based rendering system for compositing
// ASCII art, speech bubbles, and effects together.
package canvas

import (
	"strings"
	"unicode/utf8"

	"github.com/charmbracelet/lipgloss"
	"github.com/mattn/go-runewidth"
)

// Cell represents a single character cell in the canvas with optional styling.
type Cell struct {
	Rune        rune
	Style       lipgloss.Style
	Transparent bool // If true, overlay operations skip this cell
}

// Canvas is a 2D grid of cells that can be composed and rendered.
type Canvas struct {
	Width  int
	Height int
	Cells  [][]Cell
}

// NewCanvas creates a new transparent canvas with the given dimensions.
func NewCanvas(width, height int) *Canvas {
	if width <= 0 {
		width = 1
	}
	if height <= 0 {
		height = 1
	}

	cells := make([][]Cell, height)
	for y := range cells {
		cells[y] = make([]Cell, width)
		for x := range cells[y] {
			cells[y][x] = Cell{
				Rune:        ' ',
				Style:       lipgloss.NewStyle(),
				Transparent: true,
			}
		}
	}

	return &Canvas{
		Width:  width,
		Height: height,
		Cells:  cells,
	}
}

// Set places a rune at (x, y) with the given style.
// Coordinates outside the canvas bounds are ignored.
func (c *Canvas) Set(x, y int, r rune, style lipgloss.Style) {
	if x < 0 || x >= c.Width || y < 0 || y >= c.Height {
		return
	}
	c.Cells[y][x] = Cell{
		Rune:        r,
		Style:       style,
		Transparent: false,
	}
}

// Get returns the cell at (x, y). Returns a transparent space if out of bounds.
func (c *Canvas) Get(x, y int) Cell {
	if x < 0 || x >= c.Width || y < 0 || y >= c.Height {
		return Cell{Rune: ' ', Style: lipgloss.NewStyle(), Transparent: true}
	}
	return c.Cells[y][x]
}

// DrawString writes a string horizontally starting at (x, y).
// Handles multi-byte runes correctly.
func (c *Canvas) DrawString(x, y int, s string, style lipgloss.Style) {
	col := x
	for _, r := range s {
		c.Set(col, y, r, style)
		// Handle wide characters (CJK, emoji, etc.)
		w := runewidth.RuneWidth(r)
		if w > 1 {
			// Fill extra cells with zero-width placeholder
			for i := 1; i < w; i++ {
				c.Set(col+i, y, 0, style) // 0 rune means "continuation of previous"
			}
		}
		col += w
	}
}

// DrawLines writes multiple lines starting at (x, y).
func (c *Canvas) DrawLines(x, y int, lines []string, style lipgloss.Style) {
	for i, line := range lines {
		c.DrawString(x, y+i, line, style)
	}
}

// Overlay composites another canvas on top at offset (x, y).
// Non-transparent cells from 'other' overwrite cells in 'c'.
func (c *Canvas) Overlay(other *Canvas, x, y int) {
	if other == nil {
		return
	}
	for oy := 0; oy < other.Height; oy++ {
		for ox := 0; ox < other.Width; ox++ {
			cell := other.Cells[oy][ox]
			if !cell.Transparent {
				c.Set(x+ox, y+oy, cell.Rune, cell.Style)
			}
		}
	}
}

// Clone creates a deep copy of the canvas.
func (c *Canvas) Clone() *Canvas {
	clone := NewCanvas(c.Width, c.Height)
	for y := 0; y < c.Height; y++ {
		for x := 0; x < c.Width; x++ {
			clone.Cells[y][x] = c.Cells[y][x]
		}
	}
	return clone
}

// Merge creates a new canvas containing both canvases side-by-side.
// 'left' is placed at (0, 0), 'right' is placed at (left.Width + gap, 0).
func Merge(left, right *Canvas, gap int) *Canvas {
	if left == nil && right == nil {
		return NewCanvas(1, 1)
	}
	if left == nil {
		return right.Clone()
	}
	if right == nil {
		return left.Clone()
	}

	width := left.Width + gap + right.Width
	height := left.Height
	if right.Height > height {
		height = right.Height
	}

	result := NewCanvas(width, height)
	result.Overlay(left, 0, 0)
	result.Overlay(right, left.Width+gap, 0)
	return result
}

// Stack creates a new canvas with 'top' above 'bottom'.
// 'top' is placed at (0, 0), 'bottom' is placed at (0, top.Height + gap).
func Stack(top, bottom *Canvas, gap int) *Canvas {
	if top == nil && bottom == nil {
		return NewCanvas(1, 1)
	}
	if top == nil {
		return bottom.Clone()
	}
	if bottom == nil {
		return top.Clone()
	}

	width := top.Width
	if bottom.Width > width {
		width = bottom.Width
	}
	height := top.Height + gap + bottom.Height

	result := NewCanvas(width, height)
	result.Overlay(top, 0, 0)
	result.Overlay(bottom, 0, top.Height+gap)
	return result
}

// Render converts the canvas to styled strings for terminal output.
func (c *Canvas) Render() []string {
	lines := make([]string, c.Height)

	for y := 0; y < c.Height; y++ {
		var sb strings.Builder
		x := 0
		for x < c.Width {
			cell := c.Cells[y][x]

			// Skip zero-width continuation markers
			if cell.Rune == 0 {
				x++
				continue
			}

			// Render the rune with its style
			styled := cell.Style.Render(string(cell.Rune))
			sb.WriteString(styled)

			// Advance by the rune's display width
			w := runewidth.RuneWidth(cell.Rune)
			if w < 1 {
				w = 1
			}
			x += w
		}
		lines[y] = sb.String()
	}

	// Trim trailing empty lines
	for len(lines) > 0 && strings.TrimSpace(stripAnsi(lines[len(lines)-1])) == "" {
		lines = lines[:len(lines)-1]
	}

	return lines
}

// RenderPlain converts the canvas to plain strings without styling.
func (c *Canvas) RenderPlain() []string {
	lines := make([]string, c.Height)

	for y := 0; y < c.Height; y++ {
		var sb strings.Builder
		for x := 0; x < c.Width; x++ {
			cell := c.Cells[y][x]
			if cell.Rune == 0 {
				continue // Skip continuation markers
			}
			sb.WriteRune(cell.Rune)
		}
		lines[y] = sb.String()
	}

	return lines
}

// Width calculates the display width of a string (handling multi-width runes).
func StringWidth(s string) int {
	return runewidth.StringWidth(s)
}

// FromLines creates a canvas from a slice of strings.
func FromLines(lines []string, style lipgloss.Style) *Canvas {
	if len(lines) == 0 {
		return NewCanvas(1, 1)
	}

	// Calculate dimensions
	maxWidth := 0
	for _, line := range lines {
		w := StringWidth(line)
		if w > maxWidth {
			maxWidth = w
		}
	}

	if maxWidth == 0 {
		maxWidth = 1
	}

	canvas := NewCanvas(maxWidth, len(lines))
	canvas.DrawLines(0, 0, lines, style)
	return canvas
}

// stripAnsi removes ANSI escape sequences from a string.
// This is a simple implementation for trimming purposes.
func stripAnsi(s string) string {
	var result strings.Builder
	inEscape := false

	for _, r := range s {
		if r == '\x1b' {
			inEscape = true
			continue
		}
		if inEscape {
			if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') {
				inEscape = false
			}
			continue
		}
		result.WriteRune(r)
	}

	return result.String()
}

// Height returns the visual height of the canvas content (excluding trailing empty rows).
func (c *Canvas) ContentHeight() int {
	for y := c.Height - 1; y >= 0; y-- {
		for x := 0; x < c.Width; x++ {
			if !c.Cells[y][x].Transparent && c.Cells[y][x].Rune != ' ' {
				return y + 1
			}
		}
	}
	return 0
}

// ContentWidth returns the visual width of the canvas content (excluding trailing empty columns).
func (c *Canvas) ContentWidth() int {
	maxX := 0
	for y := 0; y < c.Height; y++ {
		for x := c.Width - 1; x >= 0; x-- {
			if !c.Cells[y][x].Transparent && c.Cells[y][x].Rune != ' ' {
				if x+1 > maxX {
					maxX = x + 1
				}
				break
			}
		}
	}
	return maxX
}

// Resize creates a new canvas with different dimensions, preserving content.
func (c *Canvas) Resize(newWidth, newHeight int) *Canvas {
	result := NewCanvas(newWidth, newHeight)
	result.Overlay(c, 0, 0)
	return result
}

// Clear resets all cells to transparent spaces.
func (c *Canvas) Clear() {
	for y := 0; y < c.Height; y++ {
		for x := 0; x < c.Width; x++ {
			c.Cells[y][x] = Cell{
				Rune:        ' ',
				Style:       lipgloss.NewStyle(),
				Transparent: true,
			}
		}
	}
}

// Fill sets all cells to the given rune and style (non-transparent).
func (c *Canvas) Fill(r rune, style lipgloss.Style) {
	for y := 0; y < c.Height; y++ {
		for x := 0; x < c.Width; x++ {
			c.Cells[y][x] = Cell{
				Rune:        r,
				Style:       style,
				Transparent: false,
			}
		}
	}
}

// ApplyStyle applies a style to all non-transparent cells.
func (c *Canvas) ApplyStyle(style lipgloss.Style) {
	for y := 0; y < c.Height; y++ {
		for x := 0; x < c.Width; x++ {
			if !c.Cells[y][x].Transparent {
				c.Cells[y][x].Style = style
			}
		}
	}
}

// unused but needed for compilation
var _ = utf8.RuneCountInString
