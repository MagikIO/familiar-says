package canvas

import (
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
)

// TestDefaultConfig tests the default configuration
func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	if cfg.BubbleWidth != 40 {
		t.Errorf("BubbleWidth = %d, want 40", cfg.BubbleWidth)
	}

	if cfg.BubbleStyle != BubbleStyleSay {
		t.Errorf("BubbleStyle = %v, want BubbleStyleSay", cfg.BubbleStyle)
	}

	if cfg.Layout != LayoutVertical {
		t.Errorf("Layout = %v, want LayoutVertical", cfg.Layout)
	}

	if cfg.ConnectorLen != 2 {
		t.Errorf("ConnectorLen = %d, want 2", cfg.ConnectorLen)
	}
}

// TestCompose tests basic composition
func TestCompose(t *testing.T) {
	char := builtinCat()
	text := "Hello, world!"
	eyes := "^^"
	mouth := "w"
	config := DefaultConfig()

	canvas := Compose(text, char, eyes, mouth, config)

	if canvas == nil {
		t.Fatal("Compose returned nil")
	}

	if canvas.Width == 0 || canvas.Height == 0 {
		t.Error("Composed canvas has invalid dimensions")
	}

	lines := canvas.RenderPlain()
	content := strings.Join(lines, "\n")

	// Should contain the text
	if !strings.Contains(content, "Hello") {
		t.Error("Composed output doesn't contain input text")
	}

	// Should contain the eyes
	if !strings.Contains(content, eyes) {
		t.Error("Composed output doesn't contain eyes")
	}
}

// TestComposeBubbleStyles tests different bubble styles
func TestComposeBubbleStyles(t *testing.T) {
	char := builtinCat()
	text := "Test"
	config := DefaultConfig()

	t.Run("say style", func(t *testing.T) {
		config.BubbleStyle = BubbleStyleSay
		canvas := Compose(text, char, "^^", "w", config)
		lines := canvas.RenderPlain()
		content := strings.Join(lines, "")

		// Say style uses backslash connector
		if !strings.Contains(content, "\\") {
			t.Error("Say style should use backslash connector")
		}
	})

	t.Run("think style", func(t *testing.T) {
		config.BubbleStyle = BubbleStyleThink
		canvas := Compose(text, char, "^^", "w", config)
		lines := canvas.RenderPlain()
		content := strings.Join(lines, "")

		// Think style uses 'o' connector
		if !strings.Contains(content, "o") {
			t.Error("Think style should use 'o' connector")
		}
	})

	t.Run("shout style", func(t *testing.T) {
		config.BubbleStyle = BubbleStyleShout
		canvas := Compose(text, char, "^^", "w", config)
		lines := canvas.RenderPlain()
		content := strings.Join(lines, "")

		// Shout style uses '!' connector
		if !strings.Contains(content, "!") {
			t.Error("Shout style should use '!' connector")
		}
		// Shout has jagged border
		if !strings.Contains(content, "^") {
			t.Error("Shout style should have ^ in border")
		}
	})

	t.Run("whisper style", func(t *testing.T) {
		config.BubbleStyle = BubbleStyleWhisper
		canvas := Compose(text, char, "^^", "w", config)
		lines := canvas.RenderPlain()
		content := strings.Join(lines, "")

		// Whisper style uses '.' connector
		if !strings.Contains(content, ".") {
			t.Error("Whisper style should use '.' connector")
		}
	})

	t.Run("song style", func(t *testing.T) {
		config.BubbleStyle = BubbleStyleSong
		canvas := Compose(text, char, "^^", "w", config)
		lines := canvas.RenderPlain()
		content := strings.Join(lines, "")

		// Song style uses musical note connector
		if !strings.Contains(content, "♪") {
			t.Error("Song style should use '♪' connector")
		}
	})

	t.Run("code style", func(t *testing.T) {
		config.BubbleStyle = BubbleStyleCode
		canvas := Compose(text, char, "^^", "w", config)
		lines := canvas.RenderPlain()
		content := strings.Join(lines, "")

		// Code style uses box drawing characters
		if !strings.Contains(content, "│") {
			t.Error("Code style should use box drawing characters")
		}
	})
}

// TestComposeTailDirections tests different tail directions
func TestComposeTailDirections(t *testing.T) {
	char := builtinCat()
	text := "Test"
	config := DefaultConfig()

	t.Run("tail down (default)", func(t *testing.T) {
		config.TailDirection = TailDown
		canvas := Compose(text, char, "^^", "w", config)
		if canvas == nil {
			t.Fatal("Compose returned nil")
		}
		if canvas.Height == 0 {
			t.Error("Canvas height should be > 0")
		}
	})

	t.Run("tail up", func(t *testing.T) {
		config.TailDirection = TailUp
		canvas := Compose(text, char, "^^", "w", config)
		if canvas == nil {
			t.Fatal("Compose returned nil")
		}
		// With tail up, character should be above bubble
		lines := canvas.RenderPlain()
		// The character should appear before the bubble
		charFoundFirst := false
		for _, line := range lines {
			if strings.Contains(line, "^^") {
				charFoundFirst = true
				break
			}
			if strings.Contains(line, "Test") {
				break
			}
		}
		if !charFoundFirst {
			t.Error("With TailUp, character should appear above bubble")
		}
	})

	t.Run("tail left", func(t *testing.T) {
		config.TailDirection = TailLeft
		canvas := Compose(text, char, "^^", "w", config)
		if canvas == nil {
			t.Fatal("Compose returned nil")
		}
	})

	t.Run("tail right", func(t *testing.T) {
		config.TailDirection = TailRight
		canvas := Compose(text, char, "^^", "w", config)
		if canvas == nil {
			t.Fatal("Compose returned nil")
		}
	})
}

// TestComposeLayouts tests different layout modes
func TestComposeLayouts(t *testing.T) {
	char := builtinCat()
	text := "Test message"
	config := DefaultConfig()

	t.Run("vertical layout", func(t *testing.T) {
		config.Layout = LayoutVertical
		canvas := Compose(text, char, "^^", "w", config)

		if canvas.Height <= char.Height() {
			t.Error("Vertical layout should stack bubble above character")
		}
	})

	t.Run("horizontal layout", func(t *testing.T) {
		config.Layout = LayoutHorizontal
		canvas := Compose(text, char, "^^", "w", config)

		if canvas.Width <= char.Width() {
			t.Error("Horizontal layout should place bubble beside character")
		}
	})
}

// TestComposeWithColors tests color configuration
func TestComposeWithColors(t *testing.T) {
	char := builtinCat()
	text := "Colorful"
	config := DefaultConfig()

	bubbleColor := lipgloss.NewStyle().Foreground(lipgloss.Color("#FF0000"))
	charColor := lipgloss.NewStyle().Foreground(lipgloss.Color("#00FF00"))

	config.BubbleColor = bubbleColor
	config.CharColor = charColor

	canvas := Compose(text, char, "^^", "w", config)

	if canvas == nil {
		t.Fatal("Compose with colors returned nil")
	}

	// Should not panic and should produce output
	lines := canvas.Render()
	if len(lines) == 0 {
		t.Error("Colored composition produced no output")
	}
}

// TestComposeWithCharColors tests per-part character colors
func TestComposeWithCharColors(t *testing.T) {
	char := builtinCat()
	text := "Test"
	config := DefaultConfig()

	config.CharColors = &CharacterColors{
		Outline: "#FF0000",
		Eyes:    "#00FF00",
		Mouth:   "#0000FF",
	}

	canvas := Compose(text, char, "^^", "w", config)

	if canvas == nil {
		t.Fatal("Compose with CharColors returned nil")
	}

	lines := canvas.Render()
	if len(lines) == 0 {
		t.Error("Composition with per-part colors produced no output")
	}
}

// TestComposeConfigEdgeCases tests edge cases in configuration
func TestComposeConfigEdgeCases(t *testing.T) {
	char := builtinCat()
	text := "Test"

	t.Run("zero bubble width", func(t *testing.T) {
		config := DefaultConfig()
		config.BubbleWidth = 0
		canvas := Compose(text, char, "^^", "w", config)

		if canvas == nil {
			t.Error("Compose should handle zero bubble width")
		}
	})

	t.Run("negative bubble width", func(t *testing.T) {
		config := DefaultConfig()
		config.BubbleWidth = -10
		canvas := Compose(text, char, "^^", "w", config)

		if canvas == nil {
			t.Error("Compose should handle negative bubble width")
		}
	})

	t.Run("zero connector length", func(t *testing.T) {
		config := DefaultConfig()
		config.ConnectorLen = 0
		canvas := Compose(text, char, "^^", "w", config)

		if canvas == nil {
			t.Error("Compose should handle zero connector length")
		}
	})

	t.Run("large connector length", func(t *testing.T) {
		config := DefaultConfig()
		config.ConnectorLen = 20
		canvas := Compose(text, char, "^^", "w", config)

		if canvas == nil {
			t.Error("Compose should handle large connector length")
		}
	})
}

// TestRenderBubble tests bubble rendering
func TestRenderBubble(t *testing.T) {
	style := lipgloss.NewStyle()

	t.Run("single line", func(t *testing.T) {
		canvas := RenderBubble("Hi", 40, BubbleStyleSay, style)
		lines := canvas.RenderPlain()

		if len(lines) == 0 {
			t.Error("RenderBubble returned no lines")
		}

		content := strings.Join(lines, "\n")

		// Should have borders
		if !strings.Contains(content, "_") {
			t.Error("Bubble missing top border")
		}
		if !strings.Contains(content, "-") {
			t.Error("Bubble missing bottom border")
		}

		// Single line should use < >
		if !strings.Contains(content, "<") || !strings.Contains(content, ">") {
			t.Error("Single line bubble should use < > brackets")
		}
	})

	t.Run("multi line", func(t *testing.T) {
		text := "This is a longer message that should wrap to multiple lines"
		canvas := RenderBubble(text, 15, BubbleStyleSay, style)
		lines := canvas.RenderPlain()

		if len(lines) <= 3 { // top border + content + bottom border
			t.Error("Long text should produce multiple content lines")
		}

		// Multi-line should use / \ |
		content := strings.Join(lines, "\n")
		if !strings.Contains(content, "/") && !strings.Contains(content, "\\") {
			t.Error("Multi-line bubble should use / and \\ characters")
		}
	})

	t.Run("empty text", func(t *testing.T) {
		canvas := RenderBubble("", 40, BubbleStyleSay, style)
		lines := canvas.RenderPlain()

		if len(lines) == 0 {
			t.Error("Empty text should still produce a bubble")
		}
	})

	t.Run("think style", func(t *testing.T) {
		canvas := RenderBubble("Think", 40, BubbleStyleThink, style)
		lines := canvas.RenderPlain()

		content := strings.Join(lines, "")

		// Think bubbles use parentheses
		if !strings.Contains(content, "(") || !strings.Contains(content, ")") {
			t.Error("Think bubble should use parentheses")
		}
	})
}

// TestGenerateConnector tests connector generation
func TestGenerateConnector(t *testing.T) {
	style := lipgloss.NewStyle()

	t.Run("backslash connector", func(t *testing.T) {
		canvas := generateConnector("\\", 3, 5, style)
		lines := canvas.RenderPlain()

		if len(lines) != 3 {
			t.Errorf("Connector should have 3 lines, got %d", len(lines))
		}

		// Each line should have increasing indent
		for i, line := range lines {
			if !strings.Contains(line, "\\") {
				t.Errorf("Line %d missing connector character", i)
			}
		}
	})

	t.Run("o connector", func(t *testing.T) {
		canvas := generateConnector("o", 2, 3, style)
		lines := canvas.RenderPlain()

		if len(lines) != 2 {
			t.Errorf("Connector should have 2 lines, got %d", len(lines))
		}

		for _, line := range lines {
			if !strings.Contains(line, "o") {
				t.Error("Line missing 'o' connector")
			}
		}
	})

	t.Run("zero length", func(t *testing.T) {
		canvas := generateConnector("\\", 0, 5, style)
		// FromLines with empty slice returns minimum 1x1 canvas
		if canvas.Height < 0 || canvas.Height > 1 {
			t.Errorf("Zero length connector should have height 0 or 1, got %d", canvas.Height)
		}
	})
}

// TestWrapText tests text wrapping
func TestWrapText(t *testing.T) {
	tests := []struct {
		name      string
		text      string
		width     int
		minLines  int
	}{
		{"short text", "Hello", 40, 1},
		{"exact fit", "12345", 5, 1},
		{"needs wrap", "Hello World Test", 8, 2},
		{"empty", "", 40, 0},
		{"single word long", "Supercalifragilisticexpialidocious", 10, 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lines := wrapText(tt.text, tt.width)

			if len(lines) < tt.minLines {
				t.Errorf("Got %d lines, want at least %d", len(lines), tt.minLines)
			}

			// Check that no line exceeds width (unless it's a single long word)
			for _, line := range lines {
				lineWidth := StringWidth(line)
				words := splitWords(line)
				if lineWidth > tt.width && len(words) > 1 {
					t.Errorf("Line exceeds width: %d > %d: %q", lineWidth, tt.width, line)
				}
			}
		})
	}
}

// TestSplitWords tests word splitting
func TestSplitWords(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  int
	}{
		{"simple", "hello world", 2},
		{"extra spaces", "hello  world", 2},
		{"tabs", "hello\tworld", 2},
		{"newlines", "hello\nworld", 2},
		{"single word", "hello", 1},
		{"empty", "", 0},
		{"only spaces", "   ", 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			words := splitWords(tt.input)
			if len(words) != tt.want {
				t.Errorf("Got %d words, want %d from %q", len(words), tt.want, tt.input)
			}
		})
	}
}

// TestPadRight tests right padding
func TestPadRight(t *testing.T) {
	tests := []struct {
		name  string
		input string
		width int
		want  int
	}{
		{"needs padding", "Hi", 10, 10},
		{"exact fit", "Hello", 5, 5},
		{"too long", "TooLong", 3, 7}, // Should not truncate
		{"empty", "", 5, 5},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := padRight(tt.input, tt.width)
			resultWidth := StringWidth(result)

			if resultWidth != tt.want {
				t.Errorf("Padded width = %d, want %d", resultWidth, tt.want)
			}

			if !strings.HasPrefix(result, tt.input) {
				t.Error("Padding should not change the original content")
			}
		})
	}
}

// TestRepeat tests string repetition
func TestRepeat(t *testing.T) {
	tests := []struct {
		name  string
		str   string
		count int
		want  string
	}{
		{"basic", "x", 3, "xxx"},
		{"zero", "x", 0, ""},
		{"negative", "x", -1, ""},
		{"multi char", "ab", 2, "abab"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := repeat(tt.str, tt.count)
			if result != tt.want {
				t.Errorf("repeat(%q, %d) = %q, want %q", tt.str, tt.count, result, tt.want)
			}
		})
	}
}

// TestReplaceThinkChars tests character replacement for think bubbles
func TestReplaceThinkChars(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"< text >", "( text )"},
		{"a/ text \\", "a( text )"}, // Forward slash only replaced if not at position 0
		{"| text |", "| text |"},   // Pipes unchanged
		{"normal", "normal"},       // No special chars
		{"/test", "/test"},         // Forward slash at position 0 stays
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := replaceThinkChars(tt.input)
			if result != tt.want {
				t.Errorf("replaceThinkChars(%q) = %q, want %q", tt.input, result, tt.want)
			}
		})
	}
}

// TestComposeMultiPanel tests multi-panel composition
func TestComposeMultiPanel(t *testing.T) {
	config := DefaultConfig()

	t.Run("empty panels", func(t *testing.T) {
		canvas := ComposeMultiPanel([]PanelConfig{}, config)

		if canvas.Width != 1 || canvas.Height != 1 {
			t.Error("Empty panels should return 1x1 canvas")
		}
	})

	t.Run("single panel", func(t *testing.T) {
		cat, _ := GetBuiltinCharacter("cat")
		panels := []PanelConfig{
			{
				Text:      "Hello",
				Character: cat,
				Eyes:      "^^",
				Mouth:     "w",
			},
		}

		canvas := ComposeMultiPanel(panels, config)
		if canvas == nil {
			t.Error("Single panel composition returned nil")
		}

		lines := canvas.RenderPlain()
		if len(lines) == 0 {
			t.Error("Single panel produced no output")
		}
	})

	t.Run("multiple panels", func(t *testing.T) {
		cat, _ := GetBuiltinCharacter("cat")
		owl, _ := GetBuiltinCharacter("owl")

		panels := []PanelConfig{
			{
				Text:      "Hello",
				Character: cat,
				Eyes:      "^^",
				Mouth:     "w",
			},
			{
				Text:      "World",
				Character: owl,
				Eyes:      "oo",
				Mouth:     "",
			},
		}

		canvas := ComposeMultiPanel(panels, config)
		if canvas == nil {
			t.Error("Multi panel composition returned nil")
		}

		// Width should be sum of both panels plus gap
		if canvas.Width <= cat.Width() {
			t.Error("Multi panel should be wider than single character")
		}

		lines := canvas.RenderPlain()
		content := strings.Join(lines, "")

		// Should contain text from both panels
		if !strings.Contains(content, "Hello") {
			t.Error("Missing text from first panel")
		}
		if !strings.Contains(content, "World") {
			t.Error("Missing text from second panel")
		}
	})

	t.Run("three panels", func(t *testing.T) {
		cat, _ := GetBuiltinCharacter("cat")

		panels := []PanelConfig{
			{Text: "One", Character: cat, Eyes: "^^", Mouth: "w"},
			{Text: "Two", Character: cat, Eyes: "..", Mouth: "o"},
			{Text: "Three", Character: cat, Eyes: "@@", Mouth: "D"},
		}

		canvas := ComposeMultiPanel(panels, config)
		if canvas == nil {
			t.Error("Three panel composition returned nil")
		}

		lines := canvas.RenderPlain()
		content := strings.Join(lines, "")

		if !strings.Contains(content, "One") || !strings.Contains(content, "Two") || !strings.Contains(content, "Three") {
			t.Error("Missing text from one of the three panels")
		}
	})
}

// TestComposeLongText tests composition with very long text
func TestComposeLongText(t *testing.T) {
	char := builtinCat()
	longText := strings.Repeat("This is a very long message that should wrap multiple times. ", 10)
	config := DefaultConfig()
	config.BubbleWidth = 30

	canvas := Compose(longText, char, "^^", "w", config)

	if canvas == nil {
		t.Fatal("Compose with long text returned nil")
	}

	lines := canvas.RenderPlain()
	if len(lines) == 0 {
		t.Error("Long text composition produced no output")
	}
}

// TestComposeSpecialCharacters tests composition with special characters
func TestComposeSpecialCharacters(t *testing.T) {
	char := builtinCat()
	config := DefaultConfig()

	specialTexts := []string{
		"Hello\nWorld",
		"Tab\tCharacter",
		"Symbols: !@#$%^&*()",
		"Unicode: 你好世界",
		"Emoji: 😀🎉",
	}

	for _, text := range specialTexts {
		t.Run(text[:min(len(text), 20)], func(t *testing.T) {
			canvas := Compose(text, char, "^^", "w", config)
			if canvas == nil {
				t.Error("Compose failed with special characters")
			}

			lines := canvas.Render()
			if len(lines) == 0 {
				t.Error("Special character composition produced no output")
			}
		})
	}
}

// Helper function for min
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
