// Package character provides the main character rendering interface using the canvas system.
package character

import (
	"errors"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/MagikIO/familiar-says/internal/bubble"
	"github.com/MagikIO/familiar-says/internal/canvas"
	customerrors "github.com/MagikIO/familiar-says/internal/errors"
	"github.com/MagikIO/familiar-says/internal/personality"
	"github.com/charmbracelet/lipgloss"
)

// Renderer handles character rendering using the canvas-based composition system.
type Renderer struct {
	Theme       personality.Theme
	Mood        personality.Mood
	BubbleWidth int
	CharColors  *canvas.CharacterColors // Optional per-part color overrides
}

// NewRenderer creates a new character renderer.
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

// Render renders a character with a speech bubble using the canvas system.
func (r *Renderer) Render(text string, char *canvas.Character, style bubble.Style) []string {
	return r.RenderWithTailDirection(text, char, style, canvas.TailDown)
}

// RenderWithTailDirection renders a character with a speech bubble and custom tail direction.
func (r *Renderer) RenderWithTailDirection(text string, char *canvas.Character, style bubble.Style, tailDir canvas.TailDirection) []string {
	// Get expression for mood
	expr := r.Theme.GetExpression(r.Mood)

	// Configure the compositor
	config := canvas.CompositorConfig{
		BubbleWidth:   r.BubbleWidth,
		BubbleStyle:   bubbleStyleToCanvasStyle(style),
		Layout:        canvas.LayoutVertical,
		BubbleColor:   r.Theme.BubbleStyle,
		CharColor:     r.Theme.CharacterStyle,
		CharColors:    r.CharColors,
		ConnectorLen:  2,
		TailDirection: tailDir,
	}

	// Compose the output
	result := canvas.Compose(text, char, expr.Eyes, expr.Tongue, config)

	return result.Render()
}

// bubbleStyleToCanvasStyle converts bubble.Style to canvas.BubbleStyle
func bubbleStyleToCanvasStyle(s bubble.Style) canvas.BubbleStyle {
	switch s {
	case bubble.StyleThink:
		return canvas.BubbleStyleThink
	case bubble.StyleShout:
		return canvas.BubbleStyleShout
	case bubble.StyleWhisper:
		return canvas.BubbleStyleWhisper
	case bubble.StyleSong:
		return canvas.BubbleStyleSong
	case bubble.StyleCode:
		return canvas.BubbleStyleCode
	default:
		return canvas.BubbleStyleSay
	}
}

// RenderDefault renders with the default character.
func (r *Renderer) RenderDefault(text string, style bubble.Style) []string {
	char, _ := canvas.GetBuiltinCharacter("default")
	return r.Render(text, char, style)
}

// RenderByName renders with a character by name (looks up builtin or loads from file).
func (r *Renderer) RenderByName(text string, name string, style bubble.Style) ([]string, error) {
	char, err := LoadCharacter(name)
	if err != nil {
		return nil, fmt.Errorf("failed to render character: %w", err)
	}
	return r.Render(text, char, style), nil
}

// LoadCharacter loads a character by name. It first tries to load from JSON files
// (which may have animations), then falls back to builtin characters.
func LoadCharacter(name string) (*canvas.Character, error) {
	// Normalize the name
	name = strings.ToLower(strings.TrimSpace(name))

	if name == "" {
		return nil, customerrors.NewValidationError("character name", name, "cannot be empty")
	}

	// Check if it's a file path
	if strings.HasSuffix(name, ".json") {
		char, err := canvas.LoadCharacter(name)
		if err != nil {
			return nil, customerrors.NewCharacterLoadError(name, err)
		}
		return char, nil
	}

	// Try to find a JSON file first (these have animations)
	searchPaths := []string{
		name + ".json",
		"characters/" + name + ".json",
		filepath.Join("characters", name+".json"),
	}

	for _, path := range searchPaths {
		char, err := canvas.LoadCharacter(path)
		if err == nil {
			return char, nil
		}
	}

	// Fall back to builtin character (no animations)
	if char, ok := canvas.GetBuiltinCharacter(name); ok {
		return char, nil
	}

	return nil, customerrors.NewCharacterLoadError(name, errors.New("character not found in any standard location"))
}

// ListCharacters returns all available character names.
func ListCharacters() []string {
	return canvas.ListBuiltinCharacters()
}

// RenderMultiPanel renders multiple characters side by side.
func (r *Renderer) RenderMultiPanel(panels []Panel) []string {
	if len(panels) == 0 {
		return []string{}
	}

	// Get expression for mood
	expr := r.Theme.GetExpression(r.Mood)

	// Convert to canvas panel configs
	canvasPanels := make([]canvas.PanelConfig, len(panels))
	for i, panel := range panels {
		var char *canvas.Character
		if panel.CharacterName != "" {
			var err error
			char, err = LoadCharacter(panel.CharacterName)
			if err != nil {
				char, _ = canvas.GetBuiltinCharacter("default")
			}
		} else {
			char, _ = canvas.GetBuiltinCharacter("default")
		}

		canvasPanels[i] = canvas.PanelConfig{
			Text:      panel.Text,
			Character: char,
			Eyes:      expr.Eyes,
			Mouth:     expr.Tongue,
		}
	}

	// Configure the compositor
	config := canvas.CompositorConfig{
		BubbleWidth:  r.BubbleWidth,
		BubbleStyle:  canvas.BubbleStyleSay,
		Layout:       canvas.LayoutVertical,
		BubbleColor:  r.Theme.BubbleStyle,
		CharColor:    r.Theme.CharacterStyle,
		CharColors:   r.CharColors,
		ConnectorLen: 2,
	}

	result := canvas.ComposeMultiPanel(canvasPanels, config)
	return result.Render()
}

// Panel represents a single panel in a multi-panel layout.
type Panel struct {
	Text          string
	CharacterName string
	Style         bubble.Style
}

// RenderInfo displays information about the current theme and mood.
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

// GetCharacterPreview returns a preview of a character without a bubble.
func GetCharacterPreview(name string, theme personality.Theme, mood personality.Mood) ([]string, error) {
	char, err := LoadCharacter(name)
	if err != nil {
		return nil, fmt.Errorf("failed to generate character preview: %w", err)
	}

	expr := theme.GetExpression(mood)
	charCanvas := char.ToCanvas(expr.Eyes, expr.Tongue, theme.CharacterStyle)
	return charCanvas.Render(), nil
}
