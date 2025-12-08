package bubble

import (
	"strings"
	"unicode/utf8"

	"github.com/MagikIO/familiar-says/internal/canvas"
	"github.com/charmbracelet/lipgloss"
)

// Style represents the bubble style
type Style int

const (
	StyleSay Style = iota
	StyleThink
)

// Bubble represents a speech bubble
type Bubble struct {
	Text      string
	Width     int
	Style     Style
	ThinkChar string
}

// New creates a new bubble
func New(text string, width int, style Style) *Bubble {
	if width <= 0 {
		width = 40
	}
	thinkChar := "\\"
	if style == StyleThink {
		thinkChar = "o"
	}
	return &Bubble{
		Text:      text,
		Width:     width,
		Style:     style,
		ThinkChar: thinkChar,
	}
}

// Render generates the bubble as a string slice
func (b *Bubble) Render() []string {
	lines := b.wrapText(b.Text, b.Width)
	if len(lines) == 0 {
		return []string{}
	}

	maxLen := 0
	for _, line := range lines {
		if l := utf8.RuneCountInString(line); l > maxLen {
			maxLen = l
		}
	}

	result := []string{}

	// Top border
	result = append(result, " "+strings.Repeat("_", maxLen+2))

	// Content
	if len(lines) == 1 {
		result = append(result, "< "+b.padRight(lines[0], maxLen)+" >")
	} else {
		for i, line := range lines {
			padded := b.padRight(line, maxLen)
			if i == 0 {
				result = append(result, "/ "+padded+" \\")
			} else if i == len(lines)-1 {
				result = append(result, "\\ "+padded+" /")
			} else {
				result = append(result, "| "+padded+" |")
			}
		}
	}

	// Bottom border
	result = append(result, " "+strings.Repeat("-", maxLen+2))

	return result
}

// wrapText wraps text to fit within width
func (b *Bubble) wrapText(text string, width int) []string {
	words := strings.Fields(text)
	if len(words) == 0 {
		return []string{}
	}

	lines := []string{}
	current := ""

	for _, word := range words {
		if current == "" {
			current = word
		} else if utf8.RuneCountInString(current+" "+word) <= width {
			current += " " + word
		} else {
			lines = append(lines, current)
			current = word
		}
	}

	if current != "" {
		lines = append(lines, current)
	}

	return lines
}

// padRight pads a string to the right with spaces
func (b *Bubble) padRight(s string, width int) string {
	l := utf8.RuneCountInString(s)
	if l >= width {
		return s
	}
	return s + strings.Repeat(" ", width-l)
}

// RenderToCanvas renders the bubble to a Canvas for composition.
func (b *Bubble) RenderToCanvas(style lipgloss.Style) *canvas.Canvas {
	lines := b.Render()
	return canvas.FromLines(lines, style)
}

// ToCanvasStyle converts bubble.Style to canvas.BubbleStyle
func (s Style) ToCanvasStyle() canvas.BubbleStyle {
	if s == StyleThink {
		return canvas.BubbleStyleThink
	}
	return canvas.BubbleStyleSay
}
