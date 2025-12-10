package canvas

import (
	"strings"

	"github.com/MagikIO/familiar-says/internal/bubble"
	"github.com/charmbracelet/lipgloss"
)

// BubbleStyle determines how the speech bubble is rendered.
type BubbleStyle int

const (
	BubbleStyleSay BubbleStyle = iota
	BubbleStyleThink
	BubbleStyleShout
	BubbleStyleWhisper
	BubbleStyleSong
	BubbleStyleCode
)

// Layout determines how components are arranged.
type Layout int

const (
	LayoutVertical   Layout = iota // Bubble above character (default)
	LayoutHorizontal               // Bubble beside character
)

// TailDirection specifies where the bubble tail points
type TailDirection int

const (
	TailDown  TailDirection = iota // Default: tail points down toward character
	TailUp                         // Tail points up
	TailLeft                       // Tail points left
	TailRight                      // Tail points right
)

// CompositorConfig holds configuration for the compositor.
type CompositorConfig struct {
	BubbleWidth   int
	BubbleStyle   BubbleStyle
	Layout        Layout
	BubbleColor   lipgloss.Style
	CharColor     lipgloss.Style    // Fallback color for character (deprecated in favor of CharColors)
	CharColors    *CharacterColors  // Per-part colors for character (outline, eyes, mouth)
	ConnectorLen  int               // Number of connector lines (default 2)
	TailDirection TailDirection     // Direction the bubble tail points (default down)
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

	// 2. Generate the connector using template-based character
	tmpl := GetTemplateForBubbleStyle(config.BubbleStyle)
	connectorChar := tmpl.Connector
	if connectorChar == "" {
		connectorChar = "\\"
	}
	
	// Generate connector based on tail direction
	connectorCanvas := generateConnectorWithDirection(
		connectorChar, 
		config.ConnectorLen, 
		char.GetAnchorX(), 
		config.TailDirection,
		config.CharColor,
	)

	// 3. Resolve character styles
	// Merge character's default colors with config overrides
	mergedColors := MergeColors(char.Colors, config.CharColors)
	charStyles := ResolveCharacterStyles(mergedColors, config.CharColor)

	// 4. Render the character with expressions and per-part styling
	charCanvas := char.ToCanvasStyled(eyes, mouth, charStyles)

	// 5. Compose based on layout and tail direction
	return composeWithDirection(bubbleCanvas, connectorCanvas, charCanvas, config)
}

// RenderBubble creates a speech bubble canvas using the template system.
func RenderBubble(text string, width int, style BubbleStyle, color lipgloss.Style) *Canvas {
	// Get the template for this style
	tmpl := GetTemplateForBubbleStyle(style)
	
	// Use the template-based rendering
	bubbleLines := renderBubbleWithTemplate(text, width, tmpl)
	
	return FromLines(bubbleLines, color)
}

// RenderBubbleWithTemplateName renders a bubble using a template by name.
func RenderBubbleWithTemplateName(text string, width int, templateName string, color lipgloss.Style) *Canvas {
	tmpl := bubble.GetTemplate(templateName)
	bubbleLines := renderBubbleWithTemplate(text, width, tmpl)
	return FromLines(bubbleLines, color)
}

// renderBubbleWithTemplate renders bubble lines using a template.
func renderBubbleWithTemplate(text string, width int, tmpl *bubble.BubbleTemplate) []string {
	// Apply prefix/suffix decorators if present
	if tmpl.Prefix != "" || tmpl.Suffix != "" {
		text = tmpl.Prefix + text + tmpl.Suffix
	}

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

	bubbleLines := []string{}

	// Build top border
	topBorder := buildBorder(tmpl.TopBorder, maxLen+2, tmpl.TopLeftCorner, tmpl.TopRightCorner, tmpl.BorderDecorator)
	bubbleLines = append(bubbleLines, topBorder)

	// Content lines
	if len(lines) == 1 {
		bubbleLines = append(bubbleLines, tmpl.SingleLeft+" "+padRight(lines[0], maxLen)+" "+tmpl.SingleRight)
	} else {
		for i, line := range lines {
			padded := padRight(line, maxLen)
			var left, right string
			if i == 0 {
				left, right = tmpl.MultiFirst[0], tmpl.MultiFirst[1]
			} else if i == len(lines)-1 {
				left, right = tmpl.MultiLast[0], tmpl.MultiLast[1]
			} else {
				left, right = tmpl.MultiMiddle[0], tmpl.MultiMiddle[1]
			}
			bubbleLines = append(bubbleLines, left+" "+padded+" "+right)
		}
	}

	// Build bottom border
	bottomBorder := buildBorder(tmpl.BottomBorder, maxLen+2, tmpl.BottomLeftCorner, tmpl.BottomRightCorner, tmpl.BorderDecorator)
	bubbleLines = append(bubbleLines, bottomBorder)

	return bubbleLines
}

// buildBorder creates a border line with optional corner characters and decorators
func buildBorder(borderChar string, length int, leftCorner, rightCorner, decorator string) string {
	// Handle defaults
	if leftCorner == "" {
		leftCorner = " "
	}
	if rightCorner == "" {
		rightCorner = " "
	}

	// Build the middle part of the border
	var middle string
	if decorator != "" {
		// Intersperse decorator with border characters
		middle = buildDecoratedBorder(borderChar, length, decorator)
	} else {
		middle = repeat(borderChar, length)
	}

	return leftCorner + middle + rightCorner
}

// buildDecoratedBorder creates a border with decorators interspersed
func buildDecoratedBorder(borderChar string, length int, decorator string) string {
	// Place decorator every 5 characters for visual balance
	interval := 5
	var result strings.Builder

	for i := 0; i < length; i++ {
		if i > 0 && i < length-1 && i%interval == 0 {
			result.WriteString(decorator)
		} else {
			result.WriteString(borderChar)
		}
	}

	return result.String()
}

// GetTemplateForBubbleStyle returns the template for a given BubbleStyle
func GetTemplateForBubbleStyle(style BubbleStyle) *bubble.BubbleTemplate {
	switch style {
	case BubbleStyleThink:
		return bubble.GetTemplate("think")
	case BubbleStyleShout:
		return bubble.GetTemplate("shout")
	case BubbleStyleWhisper:
		return bubble.GetTemplate("whisper")
	case BubbleStyleSong:
		return bubble.GetTemplate("song")
	case BubbleStyleCode:
		return bubble.GetTemplate("code")
	default:
		return bubble.GetTemplate("say")
	}
}

// generateConnectorWithDirection creates connector lines based on tail direction.
func generateConnectorWithDirection(char string, length int, anchorX int, direction TailDirection, style lipgloss.Style) *Canvas {
	switch direction {
	case TailUp:
		return generateConnectorUp(char, length, anchorX, style)
	case TailLeft:
		return generateConnectorLeft(char, length, style)
	case TailRight:
		return generateConnectorRight(char, length, anchorX, style)
	default:
		return generateConnector(char, length, anchorX, style)
	}
}

// generateConnector creates the thought/speech connector lines (default: down).
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

// generateConnectorUp creates a connector pointing up.
func generateConnectorUp(char string, length int, anchorX int, style lipgloss.Style) *Canvas {
	lines := make([]string, length)
	for i := 0; i < length; i++ {
		indent := anchorX + (length - 1 - i)
		if indent < 0 {
			indent = 0
		}
		lines[i] = repeat(" ", indent) + char
	}
	return FromLines(lines, style)
}

// generateConnectorLeft creates a connector pointing left (horizontal).
func generateConnectorLeft(char string, length int, style lipgloss.Style) *Canvas {
	line := repeat(char+" ", length)
	return FromLines([]string{strings.TrimRight(line, " ")}, style)
}

// generateConnectorRight creates a connector pointing right (horizontal).
func generateConnectorRight(char string, length int, anchorX int, style lipgloss.Style) *Canvas {
	indent := repeat(" ", anchorX)
	line := indent + repeat(" "+char, length)
	return FromLines([]string{line}, style)
}

// composeWithDirection arranges bubble, connector, and character based on tail direction.
func composeWithDirection(bubbleCanvas, connectorCanvas, charCanvas *Canvas, config CompositorConfig) *Canvas {
	switch config.TailDirection {
	case TailUp:
		// Character above bubble (inverted)
		result := Stack(charCanvas, connectorCanvas, 0)
		result = Stack(result, bubbleCanvas, 0)
		return result
	case TailLeft:
		// Bubble to the right of character
		return Merge(charCanvas, Merge(connectorCanvas, bubbleCanvas, 1), 2)
	case TailRight:
		// Bubble to the left of character
		return Merge(bubbleCanvas, Merge(connectorCanvas, charCanvas, 1), 2)
	default:
		// Default: bubble above character (TailDown)
		switch config.Layout {
		case LayoutHorizontal:
			combined := Stack(bubbleCanvas, connectorCanvas, 0)
			return Merge(combined, charCanvas, 2)
		default:
			result := Stack(bubbleCanvas, connectorCanvas, 0)
			result = Stack(result, charCanvas, 0)
			return result
		}
	}
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
