package character

import (
	"os"
	"strings"
	"testing"

	"github.com/MagikIO/familiar-says/internal/bubble"
	"github.com/MagikIO/familiar-says/internal/canvas"
	"github.com/MagikIO/familiar-says/internal/personality"
)

// TestNewRenderer tests renderer creation
func TestNewRenderer(t *testing.T) {
	theme := personality.ThemeDefault
	mood := personality.MoodHappy

	t.Run("valid width", func(t *testing.T) {
		r := NewRenderer(theme, mood, 50)
		if r == nil {
			t.Fatal("NewRenderer returned nil")
		}
		if r.BubbleWidth != 50 {
			t.Errorf("BubbleWidth = %d, want 50", r.BubbleWidth)
		}
		if r.Mood != mood {
			t.Error("Mood not set correctly")
		}
	})

	t.Run("zero width", func(t *testing.T) {
		r := NewRenderer(theme, mood, 0)
		if r.BubbleWidth != 40 {
			t.Errorf("Zero width should default to 40, got %d", r.BubbleWidth)
		}
	})

	t.Run("negative width", func(t *testing.T) {
		r := NewRenderer(theme, mood, -10)
		if r.BubbleWidth != 40 {
			t.Errorf("Negative width should default to 40, got %d", r.BubbleWidth)
		}
	})
}

// TestRender tests basic rendering
func TestRender(t *testing.T) {
	r := NewRenderer(personality.ThemeDefault, personality.MoodHappy, 40)
	char, _ := canvas.GetBuiltinCharacter("cat")
	text := "Hello, world!"

	lines := r.Render(text, char, bubble.StyleSay)

	if len(lines) == 0 {
		t.Error("Render produced no output")
	}

	content := strings.Join(lines, "\n")

	// Should contain the text
	if !strings.Contains(content, "Hello") {
		t.Error("Output doesn't contain input text")
	}
}

// TestRenderDifferentStyles tests rendering with different bubble styles
func TestRenderDifferentStyles(t *testing.T) {
	r := NewRenderer(personality.ThemeDefault, personality.MoodNeutral, 40)
	char, _ := canvas.GetBuiltinCharacter("cat")
	text := "Test message"

	t.Run("say style", func(t *testing.T) {
		lines := r.Render(text, char, bubble.StyleSay)
		if len(lines) == 0 {
			t.Error("Say style produced no output")
		}
	})

	t.Run("think style", func(t *testing.T) {
		lines := r.Render(text, char, bubble.StyleThink)
		if len(lines) == 0 {
			t.Error("Think style produced no output")
		}
	})
}

// TestRenderDifferentMoods tests rendering with different moods
func TestRenderDifferentMoods(t *testing.T) {
	char, _ := canvas.GetBuiltinCharacter("cat")
	text := "Test"

	moods := []personality.Mood{
		personality.MoodNeutral,
		personality.MoodHappy,
		personality.MoodSad,
		personality.MoodAngry,
		personality.MoodExcited,
	}

	for _, mood := range moods {
		t.Run(string(mood), func(t *testing.T) {
			r := NewRenderer(personality.ThemeDefault, mood, 40)
			lines := r.Render(text, char, bubble.StyleSay)

			if len(lines) == 0 {
				t.Errorf("Mood %s produced no output", mood)
			}
		})
	}
}

// TestRenderDifferentThemes tests rendering with different themes
func TestRenderDifferentThemes(t *testing.T) {
	char, _ := canvas.GetBuiltinCharacter("cat")
	text := "Test"

	themes := []personality.Theme{
		personality.ThemeDefault,
		personality.ThemeRainbow,
		personality.ThemeCyber,
		personality.ThemeRetro,
	}

	for _, theme := range themes {
		t.Run(theme.Name, func(t *testing.T) {
			r := NewRenderer(theme, personality.MoodNeutral, 40)
			lines := r.Render(text, char, bubble.StyleSay)

			if len(lines) == 0 {
				t.Errorf("Theme %s produced no output", theme.Name)
			}
		})
	}
}

// TestRenderDefault tests rendering with default character
func TestRenderDefault(t *testing.T) {
	r := NewRenderer(personality.ThemeDefault, personality.MoodHappy, 40)
	text := "Hello, default!"

	lines := r.RenderDefault(text, bubble.StyleSay)

	if len(lines) == 0 {
		t.Error("RenderDefault produced no output")
	}

	content := strings.Join(lines, "")
	if !strings.Contains(content, "Hello") {
		t.Error("Default render doesn't contain input text")
	}
}

// TestRenderByName tests rendering by character name
func TestRenderByName(t *testing.T) {
	r := NewRenderer(personality.ThemeDefault, personality.MoodHappy, 40)
	text := "Test"

	t.Run("builtin character", func(t *testing.T) {
		lines, err := r.RenderByName(text, "cat", bubble.StyleSay)
		if err != nil {
			t.Fatalf("RenderByName failed: %v", err)
		}

		if len(lines) == 0 {
			t.Error("RenderByName produced no output")
		}
	})

	t.Run("nonexistent character", func(t *testing.T) {
		_, err := r.RenderByName(text, "nonexistent", bubble.StyleSay)
		if err == nil {
			t.Error("Expected error for nonexistent character")
		}
	})

	t.Run("case insensitive", func(t *testing.T) {
		lines, err := r.RenderByName(text, "CAT", bubble.StyleSay)
		if err != nil {
			t.Fatalf("Case-insensitive lookup failed: %v", err)
		}

		if len(lines) == 0 {
			t.Error("Case-insensitive render produced no output")
		}
	})
}

// TestLoadCharacter tests character loading
func TestLoadCharacter(t *testing.T) {
	t.Run("builtin character", func(t *testing.T) {
		char, err := LoadCharacter("cat")
		if err != nil {
			t.Fatalf("Failed to load builtin cat: %v", err)
		}
		if char.Name != "cat" {
			t.Errorf("Character name = %s, want cat", char.Name)
		}
	})

	t.Run("case insensitive builtin", func(t *testing.T) {
		char, err := LoadCharacter("OWL")
		if err != nil {
			t.Fatalf("Failed to load builtin owl: %v", err)
		}
		if char.Name != "owl" {
			t.Error("Case-insensitive load failed")
		}
	})

	t.Run("json file", func(t *testing.T) {
		// Create temporary JSON character file
		tmpFile, err := os.CreateTemp("", "test_char_*.json")
		if err != nil {
			t.Fatalf("Failed to create temp file: %v", err)
		}
		defer os.Remove(tmpFile.Name())

		charJSON := `{
			"name": "testchar",
			"art": ["  test  ", "  char  "],
			"anchor": {"x": 3, "y": 0}
		}`

		_, err = tmpFile.Write([]byte(charJSON))
		if err != nil {
			t.Fatalf("Failed to write to temp file: %v", err)
		}
		tmpFile.Close()

		char, err := LoadCharacter(tmpFile.Name())
		if err != nil {
			t.Fatalf("Failed to load from JSON: %v", err)
		}

		if char.Name != "testchar" {
			t.Errorf("Character name = %s, want testchar", char.Name)
		}
	})

	t.Run("nonexistent character", func(t *testing.T) {
		_, err := LoadCharacter("nonexistent")
		if err == nil {
			t.Error("Expected error for nonexistent character")
		}
	})

	t.Run("whitespace handling", func(t *testing.T) {
		char, err := LoadCharacter("  cat  ")
		if err != nil {
			t.Fatalf("Failed with whitespace: %v", err)
		}
		if char.Name != "cat" {
			t.Error("Whitespace not trimmed properly")
		}
	})
}

// TestListCharacters tests character listing
func TestListCharacters(t *testing.T) {
	list := ListCharacters()

	if len(list) == 0 {
		t.Error("ListCharacters returned empty list")
	}

	// Should include standard characters
	expectedChars := []string{"cat", "owl", "fox", "default"}
	for _, expected := range expectedChars {
		found := false
		for _, name := range list {
			if name == expected {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected character %q not in list", expected)
		}
	}

	// All listed characters should be loadable
	for _, name := range list {
		_, err := LoadCharacter(name)
		if err != nil {
			t.Errorf("Listed character %q cannot be loaded: %v", name, err)
		}
	}
}

// TestRenderMultiPanel tests multi-panel rendering
func TestRenderMultiPanel(t *testing.T) {
	r := NewRenderer(personality.ThemeDefault, personality.MoodHappy, 40)

	t.Run("empty panels", func(t *testing.T) {
		lines := r.RenderMultiPanel([]Panel{})
		if len(lines) != 0 {
			t.Error("Empty panels should produce empty output")
		}
	})

	t.Run("single panel", func(t *testing.T) {
		panels := []Panel{
			{
				Text:          "Hello",
				CharacterName: "cat",
				Style:         bubble.StyleSay,
			},
		}

		lines := r.RenderMultiPanel(panels)
		if len(lines) == 0 {
			t.Error("Single panel produced no output")
		}

		content := strings.Join(lines, "")
		if !strings.Contains(content, "Hello") {
			t.Error("Panel text not found in output")
		}
	})

	t.Run("multiple panels", func(t *testing.T) {
		panels := []Panel{
			{
				Text:          "First",
				CharacterName: "cat",
				Style:         bubble.StyleSay,
			},
			{
				Text:          "Second",
				CharacterName: "owl",
				Style:         bubble.StyleSay,
			},
		}

		lines := r.RenderMultiPanel(panels)
		if len(lines) == 0 {
			t.Error("Multi panel produced no output")
		}

		content := strings.Join(lines, "")
		if !strings.Contains(content, "First") {
			t.Error("First panel text not found")
		}
		if !strings.Contains(content, "Second") {
			t.Error("Second panel text not found")
		}
	})

	t.Run("panel with invalid character", func(t *testing.T) {
		panels := []Panel{
			{
				Text:          "Test",
				CharacterName: "nonexistent",
				Style:         bubble.StyleSay,
			},
		}

		// Should fall back to default character
		lines := r.RenderMultiPanel(panels)
		if len(lines) == 0 {
			t.Error("Should fallback to default character")
		}
	})

	t.Run("panel with empty character name", func(t *testing.T) {
		panels := []Panel{
			{
				Text:          "Test",
				CharacterName: "",
				Style:         bubble.StyleSay,
			},
		}

		// Should use default character
		lines := r.RenderMultiPanel(panels)
		if len(lines) == 0 {
			t.Error("Should use default character")
		}
	})
}

// TestRenderInfo tests info rendering
func TestRenderInfo(t *testing.T) {
	r := NewRenderer(personality.ThemeDefault, personality.MoodHappy, 40)

	info := r.RenderInfo()

	if info == "" {
		t.Error("RenderInfo produced empty string")
	}

	// Should contain theme and mood info
	if !strings.Contains(info, "Theme") {
		t.Error("Info doesn't contain theme information")
	}
	if !strings.Contains(info, "Mood") {
		t.Error("Info doesn't contain mood information")
	}
}

// TestGetCharacterPreview tests character preview
func TestGetCharacterPreview(t *testing.T) {
	theme := personality.ThemeDefault
	mood := personality.MoodHappy

	t.Run("valid character", func(t *testing.T) {
		lines, err := GetCharacterPreview("cat", theme, mood)
		if err != nil {
			t.Fatalf("GetCharacterPreview failed: %v", err)
		}

		if len(lines) == 0 {
			t.Error("Preview produced no output")
		}
	})

	t.Run("invalid character", func(t *testing.T) {
		_, err := GetCharacterPreview("nonexistent", theme, mood)
		if err == nil {
			t.Error("Expected error for nonexistent character")
		}
	})

	t.Run("all builtin characters", func(t *testing.T) {
		for _, name := range ListCharacters() {
			_, err := GetCharacterPreview(name, theme, mood)
			if err != nil {
				t.Errorf("Preview failed for %s: %v", name, err)
			}
		}
	})
}

// TestRendererWithCharColors tests rendering with per-part character colors
func TestRendererWithCharColors(t *testing.T) {
	r := NewRenderer(personality.ThemeDefault, personality.MoodHappy, 40)
	r.CharColors = &canvas.CharacterColors{
		Outline: "#FF0000",
		Eyes:    "#00FF00",
		Mouth:   "#0000FF",
	}

	char, _ := canvas.GetBuiltinCharacter("cat")
	lines := r.Render("Test", char, bubble.StyleSay)

	if len(lines) == 0 {
		t.Error("Render with CharColors produced no output")
	}
}

// TestRenderLongText tests rendering with long text
func TestRenderLongText(t *testing.T) {
	r := NewRenderer(personality.ThemeDefault, personality.MoodHappy, 30)
	longText := strings.Repeat("This is a very long message that will definitely need to wrap across multiple lines. ", 5)

	char, _ := canvas.GetBuiltinCharacter("cat")
	lines := r.Render(longText, char, bubble.StyleSay)

	if len(lines) == 0 {
		t.Error("Long text render produced no output")
	}

	// Should have multiple lines
	if len(lines) < 5 {
		t.Error("Long text should produce multiple lines")
	}
}

// TestRenderEmptyText tests rendering with empty text
func TestRenderEmptyText(t *testing.T) {
	r := NewRenderer(personality.ThemeDefault, personality.MoodHappy, 40)
	char, _ := canvas.GetBuiltinCharacter("cat")

	lines := r.Render("", char, bubble.StyleSay)

	// Should still produce output (character with empty bubble)
	if len(lines) == 0 {
		t.Error("Empty text should still produce character output")
	}
}

// TestRenderSpecialCharacters tests rendering with special characters
func TestRenderSpecialCharacters(t *testing.T) {
	r := NewRenderer(personality.ThemeDefault, personality.MoodHappy, 40)
	char, _ := canvas.GetBuiltinCharacter("cat")

	specialTexts := []string{
		"Unicode: 你好",
		"Emoji: 😀🎉",
		"Symbols: !@#$%",
		"Newline\nTest",
	}

	for _, text := range specialTexts {
		t.Run(text[:min(len(text), 20)], func(t *testing.T) {
			lines := r.Render(text, char, bubble.StyleSay)
			if len(lines) == 0 {
				t.Error("Special character render produced no output")
			}
		})
	}
}

// TestRenderAllBuiltinCharacters tests that all builtin characters render
func TestRenderAllBuiltinCharacters(t *testing.T) {
	r := NewRenderer(personality.ThemeDefault, personality.MoodHappy, 40)
	text := "Test"

	for _, name := range ListCharacters() {
		t.Run(name, func(t *testing.T) {
			lines, err := r.RenderByName(text, name, bubble.StyleSay)
			if err != nil {
				t.Fatalf("Failed to render %s: %v", name, err)
			}

			if len(lines) == 0 {
				t.Errorf("Character %s produced no output", name)
			}
		})
	}
}

// TestRendererThreadSafety tests that renderers can be used concurrently
func TestRendererThreadSafety(t *testing.T) {
	r := NewRenderer(personality.ThemeDefault, personality.MoodHappy, 40)
	char, _ := canvas.GetBuiltinCharacter("cat")

	// Render in multiple goroutines
	done := make(chan bool, 10)
	for i := 0; i < 10; i++ {
		go func(id int) {
			lines := r.Render("Concurrent test", char, bubble.StyleSay)
			if len(lines) == 0 {
				t.Error("Concurrent render produced no output")
			}
			done <- true
		}(i)
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}
}

// Helper function for min
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
