package character

import (
	"fmt"
	"strings"

	"github.com/MagikIO/familiar-says/internal/bubble"
	"github.com/MagikIO/familiar-says/internal/personality"
	"github.com/MagikIO/familiar-says/pkg/cowparser"
	"github.com/charmbracelet/lipgloss"
)

// Renderer handles character rendering
type Renderer struct {
	Theme       personality.Theme
	Mood        personality.Mood
	BubbleWidth int
}

// NewRenderer creates a new character renderer
func NewRenderer(theme personality.Theme, mood personality.Mood, width int) *Renderer {
	if width <= 0 {
		width = 40
	}
	return &Renderer{
		Theme:       theme,
		Mood:        mood,
		BubbleWidth: width,
	}
}

// Render renders a character with a speech bubble
func (r *Renderer) Render(text string, cow *cowparser.CowFile, style bubble.Style) []string {
	// Get expression for mood
	expr := r.Theme.GetExpression(r.Mood)

	// Create speech bubble
	bubbleObj := bubble.New(text, r.BubbleWidth, style)
	bubbleLines := bubbleObj.Render()

	// Apply styling to bubble
	styledBubble := []string{}
	for _, line := range bubbleLines {
		styledBubble = append(styledBubble, r.Theme.BubbleStyle.Render(line))
	}

	// Get character body with replaced variables
	characterLines := cow.ReplaceVariables(expr.Eyes, expr.Tongue)

	// Apply styling to character
	styledCharacter := []string{}
	for _, line := range characterLines {
		if line != "" {
			styledCharacter = append(styledCharacter, r.Theme.CharacterStyle.Render(line))
		}
	}

	// Combine bubble and character
	result := append(styledBubble, styledCharacter...)
	return result
}

// RenderDefault renders with a default built-in character
func (r *Renderer) RenderDefault(text string, style bubble.Style) []string {
	// Get expression for mood
	expr := r.Theme.GetExpression(r.Mood)

	// Create speech bubble
	bubbleObj := bubble.New(text, r.BubbleWidth, style)
	bubbleLines := bubbleObj.Render()

	// Apply styling to bubble
	styledBubble := []string{}
	for _, line := range bubbleLines {
		styledBubble = append(styledBubble, r.Theme.BubbleStyle.Render(line))
	}

	// Create default character
	thinkChar := bubbleObj.ThinkChar
	characterLines := []string{
		"        " + thinkChar,
		"         " + thinkChar,
		"          (",
		"           ) " + expr.Eyes,
		"          (  ----",
		"           " + expr.Tongue,
	}

	// Apply styling to character
	styledCharacter := []string{}
	for _, line := range characterLines {
		styledCharacter = append(styledCharacter, r.Theme.CharacterStyle.Render(line))
	}

	// Combine bubble and character
	result := append(styledBubble, styledCharacter...)
	return result
}

// RenderMultiPanel renders multiple characters in panels
func (r *Renderer) RenderMultiPanel(panels []Panel) []string {
	if len(panels) == 0 {
		return []string{}
	}

	// Render each panel
	renderedPanels := [][]string{}
	maxHeight := 0

	for _, panel := range panels {
		var lines []string
		if panel.Cow != nil {
			lines = r.Render(panel.Text, panel.Cow, panel.Style)
		} else {
			lines = r.RenderDefault(panel.Text, panel.Style)
		}
		renderedPanels = append(renderedPanels, lines)
		if len(lines) > maxHeight {
			maxHeight = len(lines)
		}
	}

	// Combine panels side by side
	result := []string{}
	for i := 0; i < maxHeight; i++ {
		line := ""
		for j, panel := range renderedPanels {
			if i < len(panel) {
				line += panel[i]
			} else {
				line += strings.Repeat(" ", r.BubbleWidth+10)
			}
			if j < len(renderedPanels)-1 {
				line += "  " // Spacing between panels
			}
		}
		result = append(result, line)
	}

	return result
}

// Panel represents a single panel in a multi-panel layout
type Panel struct {
	Text  string
	Cow   *cowparser.CowFile
	Style bubble.Style
}

// GetDefaultCharacter returns a default character definition
func GetDefaultCharacter() *cowparser.CowFile {
	return &cowparser.CowFile{
		Eyes:     "oo",
		Tongue:   "  ",
		Thoughts: "\\",
		Body: []string{
			"        $thoughts",
			"         $thoughts",
			"          (",
			"           ) $eyes",
			"          (  ----",
			"           $tongue",
		},
		Variables: make(map[string]string),
	}
}

// RenderInfo displays information about the current theme and mood
func (r *Renderer) RenderInfo() string {
	expr := r.Theme.GetExpression(r.Mood)

	info := fmt.Sprintf("Theme: %s | Mood: %s | Eyes: %s | Tongue: %s",
		r.Theme.Name,
		r.Mood,
		expr.Eyes,
		expr.Tongue,
	)

	style := lipgloss.NewStyle().
		Foreground(r.Theme.AccentColor).
		Bold(true).
		Padding(1, 2)

	return style.Render(info)
}
