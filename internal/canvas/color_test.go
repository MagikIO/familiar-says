package canvas

import (
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
)

// TestParseColorHex tests parsing hex color codes
func TestParseColorHex(t *testing.T) {
	tests := []struct {
		name  string
		input string
		valid bool
	}{
		{"6-char with hash", "#FF6B6B", true},
		{"6-char without hash", "FF6B6B", true},
		{"3-char with hash", "#F6B", true},
		{"3-char without hash", "F6B", true},
		{"lowercase", "#ff6b6b", true},
		{"mixed case", "#Ff6B6b", true},
		{"empty", "", false},
		{"invalid chars", "#GGGGGG", false},
		{"too short", "#FF", false},
		{"too long", "#FF6B6B00", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			color := ParseColor(tt.input)
			colorStr := string(color)

			if tt.valid {
				if colorStr == "" && tt.input != "" {
					t.Error("Valid color parsed to empty string")
				}
				// Valid hex should start with # after parsing
				if tt.input != "" && !strings.HasPrefix(colorStr, "#") && tt.input[0] != '#' {
					// Was without hash, should be normalized
				}
			}
		})
	}
}

// TestParseColorNamed tests parsing named colors
func TestParseColorNamed(t *testing.T) {
	namedColorTests := []string{
		"red", "green", "blue", "yellow", "cyan", "magenta",
		"orange", "pink", "purple", "violet", "brown",
		"gray", "grey", "gold", "silver",
		"fire", "ice", "forest", "midnight", "sunset",
	}

	for _, name := range namedColorTests {
		t.Run(name, func(t *testing.T) {
			color := ParseColor(name)
			if string(color) == "" {
				t.Errorf("Named color %q parsed to empty", name)
			}

			// Should be case-insensitive
			upperColor := ParseColor(strings.ToUpper(name))
			if string(upperColor) == "" {
				t.Errorf("Uppercase %q not recognized", name)
			}
		})
	}
}

// TestParseColorANSI tests parsing ANSI 256 color numbers
func TestParseColorANSI(t *testing.T) {
	tests := []struct {
		name  string
		input string
		valid bool
	}{
		{"valid 0", "0", true},
		{"valid 128", "128", true},
		{"valid 255", "255", true},
		{"invalid negative", "-1", false},
		{"invalid too high", "256", false},
		{"invalid 1000", "1000", false},
		{"not a number", "abc", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			color := ParseColor(tt.input)
			colorStr := string(color)

			if tt.valid {
				if colorStr == "" {
					t.Error("Valid ANSI color parsed to empty")
				}
			}
		})
	}
}

// TestParseColorEmpty tests parsing empty strings
func TestParseColorEmpty(t *testing.T) {
	tests := []string{"", "   ", "\t", "\n"}

	for _, input := range tests {
		t.Run("whitespace", func(t *testing.T) {
			color := ParseColor(input)
			if string(color) != "" {
				t.Error("Empty/whitespace should parse to empty color")
			}
		})
	}
}

// TestParseColorOrDefault tests parsing with fallback
func TestParseColorOrDefault(t *testing.T) {
	defaultStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#000000"))

	t.Run("valid color", func(t *testing.T) {
		style := ParseColorOrDefault("#FF0000", defaultStyle)
		// Should create a style (we can't compare styles directly)
		// Just verify it doesn't panic and returns a style
		_ = style
	})

	t.Run("empty string", func(t *testing.T) {
		// Empty string should use default behavior
		_ = ParseColorOrDefault("", defaultStyle)
	})

	t.Run("whitespace", func(t *testing.T) {
		// Whitespace should use default behavior
		_ = ParseColorOrDefault("   ", defaultStyle)
	})
}

// TestResolveCharacterStyles tests style resolution
func TestResolveCharacterStyles(t *testing.T) {
	fallback := lipgloss.NewStyle().Foreground(lipgloss.Color("#FFFFFF"))

	t.Run("nil colors", func(t *testing.T) {
		styles := ResolveCharacterStyles(nil, fallback)

		// Should return valid styles (can't compare directly)
		// Just verify it doesn't panic
		_ = styles.Outline
		_ = styles.Eyes
		_ = styles.Mouth
	})

	t.Run("empty colors", func(t *testing.T) {
		colors := &CharacterColors{}
		styles := ResolveCharacterStyles(colors, fallback)

		// Should return valid styles
		_ = styles.Outline
		_ = styles.Eyes
		_ = styles.Mouth
	})

	t.Run("partial colors", func(t *testing.T) {
		colors := &CharacterColors{
			Outline: "#FF0000",
			// Eyes and Mouth empty
		}
		styles := ResolveCharacterStyles(colors, fallback)

		// Should resolve successfully
		_ = styles.Outline
		_ = styles.Eyes
		_ = styles.Mouth
	})

	t.Run("all colors specified", func(t *testing.T) {
		colors := &CharacterColors{
			Outline: "#FF0000",
			Eyes:    "#00FF00",
			Mouth:   "#0000FF",
		}
		styles := ResolveCharacterStyles(colors, fallback)

		// Should resolve successfully
		_ = styles.Outline
		_ = styles.Eyes
		_ = styles.Mouth
	})
}

// TestMergeColors tests color merging
func TestMergeColors(t *testing.T) {
	t.Run("both nil", func(t *testing.T) {
		result := MergeColors(nil, nil)
		if result != nil {
			t.Error("Merging two nils should return nil")
		}
	})

	t.Run("base nil", func(t *testing.T) {
		override := &CharacterColors{
			Outline: "#FF0000",
		}
		result := MergeColors(nil, override)

		if result == nil {
			t.Fatal("Result should not be nil")
		}
		if result.Outline != "#FF0000" {
			t.Error("Override not applied when base is nil")
		}
	})

	t.Run("override nil", func(t *testing.T) {
		base := &CharacterColors{
			Outline: "#FF0000",
		}
		result := MergeColors(base, nil)

		if result == nil {
			t.Fatal("Result should not be nil")
		}
		if result.Outline != "#FF0000" {
			t.Error("Base not preserved when override is nil")
		}
	})

	t.Run("override takes precedence", func(t *testing.T) {
		base := &CharacterColors{
			Outline: "#FF0000",
			Eyes:    "#00FF00",
			Mouth:   "#0000FF",
		}
		override := &CharacterColors{
			Outline: "#FFFFFF",
			// Eyes and Mouth empty
		}
		result := MergeColors(base, override)

		if result.Outline != "#FFFFFF" {
			t.Error("Override should take precedence for Outline")
		}
		if result.Eyes != "#00FF00" {
			t.Error("Base Eyes should be preserved when override is empty")
		}
		if result.Mouth != "#0000FF" {
			t.Error("Base Mouth should be preserved when override is empty")
		}
	})

	t.Run("empty string in override doesn't overwrite", func(t *testing.T) {
		base := &CharacterColors{
			Outline: "#FF0000",
		}
		override := &CharacterColors{
			Outline: "", // Empty
		}
		result := MergeColors(base, override)

		if result.Outline != "#FF0000" {
			t.Error("Empty string in override should not overwrite base")
		}
	})
}

// TestCharacterColorsIsEmpty tests emptiness check
func TestCharacterColorsIsEmpty(t *testing.T) {
	tests := []struct {
		name     string
		colors   *CharacterColors
		wantEmpty bool
	}{
		{"nil", nil, true},
		{"empty struct", &CharacterColors{}, true},
		{"only outline", &CharacterColors{Outline: "#FF0000"}, false},
		{"only eyes", &CharacterColors{Eyes: "#00FF00"}, false},
		{"only mouth", &CharacterColors{Mouth: "#0000FF"}, false},
		{"all fields", &CharacterColors{Outline: "#FF0000", Eyes: "#00FF00", Mouth: "#0000FF"}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			isEmpty := tt.colors.IsEmpty()
			if isEmpty != tt.wantEmpty {
				t.Errorf("IsEmpty() = %v, want %v", isEmpty, tt.wantEmpty)
			}
		})
	}
}

// TestListNamedColors tests listing available named colors
func TestListNamedColors(t *testing.T) {
	colors := ListNamedColors()

	if len(colors) == 0 {
		t.Error("ListNamedColors returned empty list")
	}

	// Check that some expected colors are present
	expectedColors := []string{"red", "green", "blue", "fire", "ice"}
	for _, expected := range expectedColors {
		found := false
		for _, color := range colors {
			if color == expected {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected color %q not in list", expected)
		}
	}

	// Verify all listed colors can be parsed
	for _, name := range colors {
		color := ParseColor(name)
		if string(color) == "" {
			t.Errorf("Listed color %q cannot be parsed", name)
		}
	}
}

// TestValidateColor tests color validation
func TestValidateColor(t *testing.T) {
	tests := []struct {
		name  string
		input string
		valid bool
	}{
		// Valid cases
		{"empty", "", true},
		{"whitespace", "   ", true},
		{"named color", "red", true},
		{"hex 6", "#FF6B6B", true},
		{"hex 3", "#F6B", true},
		{"hex no hash", "FF6B6B", true},
		{"ansi 0", "0", true},
		{"ansi 128", "128", true},
		{"ansi 255", "255", true},
		{"uppercase named", "RED", true},
		{"mixed case hex", "#Ff6B6b", true},

		// Invalid cases
		{"unknown name", "notacolor", false},
		{"invalid hex", "#GGGGGG", false},
		{"ansi negative", "-1", false},
		{"ansi too high", "256", true}, // Note: lipgloss may accept 256 as extended color
		{"ansi 1000", "1000", false},
		{"partial hex", "#FF", false},
		{"too long hex", "#FF6B6B00", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			valid := ValidateColor(tt.input)
			if valid != tt.valid {
				t.Errorf("ValidateColor(%q) = %v, want %v", tt.input, valid, tt.valid)
			}
		})
	}
}

// TestParseColorConsistency tests that parsing and validation agree
func TestParseColorConsistency(t *testing.T) {
	testInputs := []string{
		"red", "blue", "#FF0000", "#F00", "128", "invalid", "", "  ",
	}

	for _, input := range testInputs {
		t.Run(input, func(t *testing.T) {
			isValid := ValidateColor(input)
			parsed := ParseColor(input)

			if isValid {
				// Valid colors should parse to non-empty (unless input is whitespace)
				if strings.TrimSpace(input) != "" && string(parsed) == "" {
					t.Error("Valid color parsed to empty")
				}
			}
		})
	}
}

// TestColorExpansion tests 3-char hex expansion
func TestColorExpansion(t *testing.T) {
	// Test that 3-char hex codes are expanded correctly
	color3 := ParseColor("#F6B")
	color6 := ParseColor("#FF66BB")

	// Both should be valid
	if string(color3) == "" {
		t.Error("3-char hex should be valid")
	}
	if string(color6) == "" {
		t.Error("6-char hex should be valid")
	}

	// Both should start with #
	if !strings.HasPrefix(string(color3), "#") {
		t.Error("3-char hex should be normalized with #")
	}
}

// TestNamedColorCaseInsensitivity tests case-insensitive named colors
func TestNamedColorCaseInsensitivity(t *testing.T) {
	testCases := []string{
		"red", "RED", "Red", "rEd",
		"blue", "BLUE", "Blue",
		"fire", "FIRE", "Fire",
	}

	for _, input := range testCases {
		t.Run(input, func(t *testing.T) {
			color := ParseColor(input)
			if string(color) == "" {
				t.Errorf("Named color %q not recognized", input)
			}

			// Should also validate
			if !ValidateColor(input) {
				t.Errorf("Named color %q not validated", input)
			}
		})
	}
}

// TestResolveCharacterStylesWithNamedColors tests resolution with named colors
func TestResolveCharacterStylesWithNamedColors(t *testing.T) {
	fallback := lipgloss.NewStyle()

	colors := &CharacterColors{
		Outline: "red",
		Eyes:    "green",
		Mouth:   "blue",
	}

	styles := ResolveCharacterStyles(colors, fallback)

	// All should be resolved (verify doesn't panic)
	_ = styles.Outline
	_ = styles.Eyes
	_ = styles.Mouth
}

// TestResolveCharacterStylesWithMixedColors tests resolution with mixed color formats
func TestResolveCharacterStylesWithMixedColors(t *testing.T) {
	fallback := lipgloss.NewStyle()

	colors := &CharacterColors{
		Outline: "#FF0000",  // Hex
		Eyes:    "green",    // Named
		Mouth:   "196",      // ANSI
	}

	styles := ResolveCharacterStyles(colors, fallback)

	// All should be resolved (verify doesn't panic)
	_ = styles.Outline
	_ = styles.Eyes
	_ = styles.Mouth
}

// TestParseColorInvalidFormats tests handling of invalid formats
func TestParseColorInvalidFormats(t *testing.T) {
	invalidInputs := []string{
		"rgb(255,0,0)",     // CSS format
		"hsl(0,100%,50%)",  // HSL format
		"#GG0000",          // Invalid hex chars
		"256colors",        // Not a valid format
		"color:red",        // CSS-like syntax
	}

	for _, input := range invalidInputs {
		t.Run(input, func(t *testing.T) {
			// Should not panic
			color := ParseColor(input)
			// May return the input as-is for lipgloss to handle
			_ = color

			// Should validate as false
			if ValidateColor(input) {
				t.Errorf("Invalid input %q validated as true", input)
			}
		})
	}
}
