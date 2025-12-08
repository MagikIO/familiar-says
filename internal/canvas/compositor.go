package canvas

import (
	"github.com/charmbracelet/lipgloss"
)

// BubbleStyle determines how the speech bubble is rendered.
type BubbleStyle int

const (
	BubbleStyleSay BubbleStyle = iota
	BubbleStyleThink
)

// Layout determines how components are arranged.
type Layout int

const (
	LayoutVertical   Layout = iota // Bubble above character (default)
	LayoutHorizontal               // Bubble beside character
)

// CompositorConfig holds configuration for the compositor.
type CompositorConfig struct {
	BubbleWidth  int
	BubbleStyle  BubbleStyle
	Layout       Layout
	BubbleColor  lipgloss.Style
	CharColor    lipgloss.Style
	ConnectorLen int // Number of connector lines (default 2)
}

// DefaultConfig returns a default compositor configuration.
func DefaultConfig() CompositorConfig {
	return CompositorConfig{
		BubbleWidth:  40,
		BubbleStyle:  BubbleStyleSay,
		Layout:       LayoutVertical,
		BubbleColor:  lipgloss.NewStyle(),
		CharColor:    lipgloss.NewStyle(),
		ConnectorLen: 2,
	}
}

// Compose combines a speech bubble and character into a single canvas.
func Compose(text string, char *Character, eyes, mouth string, config CompositorConfig) *Canvas {
	if config.BubbleWidth <= 0 {
		config.BubbleWidth = 40
	}
	if config.ConnectorLen <= 0 {
		config.ConnectorLen = 2
	}

	// 1. Render the speech bubble
	bubbleCanvas := RenderBubble(text, config.BubbleWidth, config.BubbleStyle, config.BubbleColor)

	// 2. Generate the connector
	connectorChar := "\\"
	if config.BubbleStyle == BubbleStyleThink {
		connectorChar = "o"
	}
	connectorCanvas := generateConnector(connectorChar, config.ConnectorLen, char.GetAnchorX(), config.CharColor)

	// 3. Render the character with expressions
	charCanvas := char.ToCanvas(eyes, mouth, config.CharColor)

	// 4. Compose based on layout
	switch config.Layout {
	case LayoutHorizontal:
		// Character beside bubble
		combined := Stack(bubbleCanvas, connectorCanvas, 0)
		return Merge(combined, charCanvas, 2)
	default:
		// Vertical: bubble above connector above character
		result := Stack(bubbleCanvas, connectorCanvas, 0)
		result = Stack(result, charCanvas, 0)
		return result
	}
}

// RenderBubble creates a speech bubble canvas.
func RenderBubble(text string, width int, style BubbleStyle, color lipgloss.Style) *Canvas {
	lines := wrapText(text, width)
	if len(lines) == 0 {
		lines = []string{""}
	}

	// Calculate max line length
	maxLen := 0
	for _, line := range lines {
		w := StringWidth(line)
		if w > maxLen {
			maxLen = w
		}
	}

	// Build bubble lines
	bubbleLines := []string{}

	// Top border
	topBorder := " " + repeat("_", maxLen+2)
	bubbleLines = append(bubbleLines, topBorder)

	// Content
	if len(lines) == 1 {
		bubbleLines = append(bubbleLines, "< "+padRight(lines[0], maxLen)+" >")
	} else {
		for i, line := range lines {
			padded := padRight(line, maxLen)
			if i == 0 {
				bubbleLines = append(bubbleLines, "/ "+padded+" \\")
			} else if i == len(lines)-1 {
				bubbleLines = append(bubbleLines, "\\ "+padded+" /")
			} else {
				bubbleLines = append(bubbleLines, "| "+padded+" |")
			}
		}
	}

	// Bottom border
	bottomBorder := " " + repeat("-", maxLen+2)
	bubbleLines = append(bubbleLines, bottomBorder)

	// Apply thought bubble style if needed
	if style == BubbleStyleThink {
		// Replace < > with ( )
		for i := range bubbleLines {
			if len(bubbleLines[i]) > 0 {
				bubbleLines[i] = replaceThinkChars(bubbleLines[i])
			}
		}
	}

	return FromLines(bubbleLines, color)
}

// generateConnector creates the thought/speech connector lines.
func generateConnector(char string, length int, anchorX int, style lipgloss.Style) *Canvas {
	lines := make([]string, length)
	for i := 0; i < length; i++ {
		// Indent increases as we go down toward the character
		indent := anchorX + i
		if indent < 0 {
			indent = 0
		}
		lines[i] = repeat(" ", indent) + char
	}
	return FromLines(lines, style)
}

// replaceThinkChars converts speech bubble chars to thought bubble chars.
func replaceThinkChars(s string) string {
	result := []rune(s)
	for i, r := range result {
		switch r {
		case '<':
			result[i] = '('
		case '>':
			result[i] = ')'
		case '/':
			if i > 0 {
				result[i] = '('
			}
		case '\\':
			result[i] = ')'
		}
	}
	return string(result)
}

// wrapText wraps text to fit within the given width.
func wrapText(text string, width int) []string {
	if text == "" {
		return []string{}
	}

	words := splitWords(text)
	if len(words) == 0 {
		return []string{}
	}

	lines := []string{}
	current := ""

	for _, word := range words {
		if current == "" {
			current = word
		} else if StringWidth(current+" "+word) <= width {
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

// splitWords splits text into words, preserving whitespace behavior.
func splitWords(text string) []string {
	words := []string{}
	current := ""

	for _, r := range text {
		if r == ' ' || r == '\t' || r == '\n' {
			if current != "" {
				words = append(words, current)
				current = ""
			}
		} else {
			current += string(r)
		}
	}

	if current != "" {
		words = append(words, current)
	}

	return words
}

// padRight pads a string with spaces to reach the target width.
func padRight(s string, width int) string {
	current := StringWidth(s)
	if current >= width {
		return s
	}
	return s + repeat(" ", width-current)
}

// repeat returns a string of n copies of s.
func repeat(s string, n int) string {
	if n <= 0 {
		return ""
	}
	result := ""
	for i := 0; i < n; i++ {
		result += s
	}
	return result
}

// ComposeMultiPanel renders multiple characters with their messages side by side.
func ComposeMultiPanel(panels []PanelConfig, config CompositorConfig) *Canvas {
	if len(panels) == 0 {
		return NewCanvas(1, 1)
	}

	var result *Canvas
	for i, panel := range panels {
		panelCanvas := Compose(panel.Text, panel.Character, panel.Eyes, panel.Mouth, config)
		if i == 0 {
			result = panelCanvas
		} else {
			result = Merge(result, panelCanvas, 3)
		}
	}

	return result
}

// PanelConfig holds configuration for a single panel in multi-panel mode.
type PanelConfig struct {
	Text      string
	Character *Character
	Eyes      string
	Mouth     string
}
