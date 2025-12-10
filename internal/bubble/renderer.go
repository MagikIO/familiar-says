package bubble

import (
	"strings"
)

// RenderLines renders the bubble as string lines using the template system.
// This is a template-aware version that uses BubbleTemplate.
func RenderLines(text string, width int, style Style) []string {
	tmpl := GetTemplateForStyle(style)
	return RenderWithTemplate(text, width, tmpl)
}

// RenderWithTemplate renders a bubble using a specific template.
func RenderWithTemplate(text string, width int, tmpl *BubbleTemplate) []string {
	// Apply prefix/suffix decorators if present
	if tmpl.Prefix != "" || tmpl.Suffix != "" {
		text = tmpl.Prefix + text + tmpl.Suffix
	}

	lines := wrapTextForBubble(text, width)
	if len(lines) == 0 {
		lines = []string{""}
	}

	// Calculate max line length
	maxLen := 0
	for _, line := range lines {
		if l := runeCount(line); l > maxLen {
			maxLen = l
		}
	}

	bubbleLines := []string{}

	// Build top border
	topBorder := buildBorder(tmpl.TopBorder, maxLen+2, tmpl.TopLeftCorner, tmpl.TopRightCorner, tmpl.BorderDecorator)
	bubbleLines = append(bubbleLines, topBorder)

	// Content lines
	if len(lines) == 1 {
		bubbleLines = append(bubbleLines, tmpl.SingleLeft+" "+padRightRunes(lines[0], maxLen)+" "+tmpl.SingleRight)
	} else {
		for i, line := range lines {
			padded := padRightRunes(line, maxLen)
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
	if decorator != "" && len(decorator) > 0 {
		// Intersperse decorator with border characters
		middle = buildDecoratedBorder(borderChar, length, decorator)
	} else {
		middle = repeatString(borderChar, length)
	}

	return leftCorner + middle + rightCorner
}

// buildDecoratedBorder creates a border with decorators interspersed
func buildDecoratedBorder(borderChar string, length int, decorator string) string {
	// Place decorator every 4-6 characters for visual balance
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

// repeatString repeats a string n times
func repeatString(s string, n int) string {
	if n <= 0 {
		return ""
	}
	var result strings.Builder
	for i := 0; i < n; i++ {
		result.WriteString(s)
	}
	return result.String()
}

// wrapTextForBubble wraps text to fit within the given width.
func wrapTextForBubble(text string, width int) []string {
	if text == "" {
		return []string{}
	}

	words := strings.Fields(text)
	if len(words) == 0 {
		return []string{}
	}

	lines := []string{}
	current := ""

	for _, word := range words {
		if current == "" {
			current = word
		} else if runeCount(current+" "+word) <= width {
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

// runeCount returns the number of runes in a string
func runeCount(s string) int {
	return len([]rune(s))
}

// padRightRunes pads a string with spaces to reach the target width (in runes)
func padRightRunes(s string, width int) string {
	current := runeCount(s)
	if current >= width {
		return s
	}
	return s + strings.Repeat(" ", width-current)
}

// GenerateConnectorLines creates the connector/tail lines as strings.
func GenerateConnectorLines(style Style, length int, anchorX int, direction TailDirection) []string {
	tmpl := GetTemplateForStyle(style)
	return GenerateConnectorWithTemplate(tmpl, length, anchorX, direction)
}

// GenerateConnectorWithTemplate creates connector lines using a template.
func GenerateConnectorWithTemplate(tmpl *BubbleTemplate, length int, anchorX int, direction TailDirection) []string {
	char := tmpl.Connector
	if char == "" {
		char = "\\"
	}

	switch direction {
	case TailUp:
		return generateConnectorUp(char, length, anchorX)
	case TailLeft:
		return generateConnectorLeft(char, length)
	case TailRight:
		return generateConnectorRight(char, length, anchorX)
	default:
		return generateConnectorDown(char, length, anchorX)
	}
}

// generateConnectorDown creates a connector pointing down (default)
func generateConnectorDown(char string, length int, anchorX int) []string {
	lines := make([]string, length)
	for i := 0; i < length; i++ {
		indent := anchorX + i
		if indent < 0 {
			indent = 0
		}
		lines[i] = repeatString(" ", indent) + char
	}
	return lines
}

// generateConnectorUp creates a connector pointing up
func generateConnectorUp(char string, length int, anchorX int) []string {
	lines := make([]string, length)
	for i := 0; i < length; i++ {
		indent := anchorX + (length - 1 - i)
		if indent < 0 {
			indent = 0
		}
		lines[i] = repeatString(" ", indent) + char
	}
	return lines
}

// generateConnectorLeft creates a connector pointing left
func generateConnectorLeft(char string, length int) []string {
	// Horizontal connector going left
	line := repeatString(char+" ", length)
	return []string{line}
}

// generateConnectorRight creates a connector pointing right
func generateConnectorRight(char string, length int, anchorX int) []string {
	// Horizontal connector going right
	indent := repeatString(" ", anchorX)
	line := indent + repeatString(" "+char, length)
	return []string{line}
}

// GetConnectorCharForStyle returns the connector character for a bubble style
func GetConnectorCharForStyle(style Style) string {
	tmpl := GetTemplateForStyle(style)
	if tmpl.Connector != "" {
		return tmpl.Connector
	}
	return "\\"
}
