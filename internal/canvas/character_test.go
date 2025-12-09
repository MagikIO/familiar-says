package canvas

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
)

// TestLoadCharacterFromJSON tests loading a character from a JSON file
func TestLoadCharacterFromJSON(t *testing.T) {
	// Create a temporary JSON file
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test_char.json")

	validJSON := `{
		"name": "testcat",
		"description": "A test cat",
		"art": [
			"  /\\_/\\  ",
			" ( @@ ) ",
			" =( Y )="
		],
		"anchor": {"x": 4, "y": 0},
		"eyes": {
			"line": 1,
			"col": 3,
			"width": 2,
			"placeholder": "@@"
		},
		"mouth": {
			"line": 2,
			"col": 4,
			"width": 1,
			"placeholder": "Y"
		}
	}`

	err := os.WriteFile(testFile, []byte(validJSON), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Test successful load
	char, err := LoadCharacter(testFile)
	if err != nil {
		t.Fatalf("LoadCharacter failed: %v", err)
	}

	if char.Name != "testcat" {
		t.Errorf("Name = %s, want testcat", char.Name)
	}
	if char.Description != "A test cat" {
		t.Errorf("Description = %s, want 'A test cat'", char.Description)
	}
	if len(char.Art) != 3 {
		t.Errorf("Art lines = %d, want 3", len(char.Art))
	}
	if char.Eyes == nil {
		t.Error("Eyes slot is nil")
	}
	if char.Mouth == nil {
		t.Error("Mouth slot is nil")
	}
}

// TestLoadCharacterErrors tests error handling when loading characters
func TestLoadCharacterErrors(t *testing.T) {
	tmpDir := t.TempDir()

	t.Run("file not found", func(t *testing.T) {
		_, err := LoadCharacter(filepath.Join(tmpDir, "nonexistent.json"))
		if err == nil {
			t.Error("Expected error for nonexistent file")
		}
	})

	t.Run("invalid JSON", func(t *testing.T) {
		invalidFile := filepath.Join(tmpDir, "invalid.json")
		err := os.WriteFile(invalidFile, []byte("not valid json{"), 0644)
		if err != nil {
			t.Fatalf("Failed to create test file: %v", err)
		}

		_, err = LoadCharacter(invalidFile)
		if err == nil {
			t.Error("Expected error for invalid JSON")
		}
	})

	t.Run("missing art", func(t *testing.T) {
		noArtFile := filepath.Join(tmpDir, "no_art.json")
		noArtJSON := `{"name": "test", "art": []}`
		err := os.WriteFile(noArtFile, []byte(noArtJSON), 0644)
		if err != nil {
			t.Fatalf("Failed to create test file: %v", err)
		}

		_, err = LoadCharacter(noArtFile)
		if err == nil {
			t.Error("Expected error for character with no art")
		}
	})

	t.Run("missing name uses filename", func(t *testing.T) {
		noNameFile := filepath.Join(tmpDir, "unnamed.json")
		noNameJSON := `{"art": ["test"]}`
		err := os.WriteFile(noNameFile, []byte(noNameJSON), 0644)
		if err != nil {
			t.Fatalf("Failed to create test file: %v", err)
		}

		char, err := LoadCharacter(noNameFile)
		if err != nil {
			t.Fatalf("LoadCharacter failed: %v", err)
		}
		if char.Name != "unnamed" {
			t.Errorf("Name = %s, want 'unnamed' (from filename)", char.Name)
		}
	})
}

// TestToCanvasStyled tests rendering with per-part styling
func TestToCanvasStyled(t *testing.T) {
	char := builtinCat()
	eyes := "^^"
	mouth := "w"

	outlineStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#FFFFFF"))
	eyeStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#00FF00"))
	mouthStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#FF0000"))

	styles := CharacterStyles{
		Outline: outlineStyle,
		Eyes:    eyeStyle,
		Mouth:   mouthStyle,
	}

	canvas := char.ToCanvasStyled(eyes, mouth, styles)

	if canvas == nil {
		t.Fatal("ToCanvasStyled returned nil")
	}

	if canvas.Width <= 0 || canvas.Height <= 0 {
		t.Error("Canvas has invalid dimensions")
	}

	// Verify the canvas contains the replaced expressions
	lines := canvas.RenderPlain()
	content := strings.Join(lines, "")

	// Should contain the eyes
	if !strings.Contains(content, eyes) {
		t.Error("Canvas doesn't contain replaced eyes")
	}

	// Should contain the mouth
	if !strings.Contains(content, mouth) {
		t.Error("Canvas doesn't contain replaced mouth")
	}

	// Should not contain placeholders
	if strings.Contains(content, "@@") {
		t.Error("Canvas still contains eye placeholder")
	}
	if strings.Contains(content, "Y") && char.Mouth.Placeholder == "Y" {
		// This might be tricky as Y could be in the mouth replacement
		// So we check for the exact placeholder pattern
		for _, line := range char.Art {
			if strings.Contains(line, "Y") && !strings.Contains(content, "w") {
				t.Error("Mouth placeholder not replaced")
			}
		}
	}
}

// TestToCanvas tests the deprecated method
func TestToCanvas(t *testing.T) {
	char := builtinCat()
	style := lipgloss.NewStyle()

	canvas := char.ToCanvas("^^", "w", style)

	if canvas == nil {
		t.Fatal("ToCanvas returned nil")
	}

	lines := canvas.RenderPlain()
	content := strings.Join(lines, "")

	if !strings.Contains(content, "^^") {
		t.Error("Eyes not replaced in ToCanvas")
	}
}

// TestReplaceSlot tests slot replacement logic
func TestReplaceSlot(t *testing.T) {
	tests := []struct {
		name        string
		line        string
		slot        *Slot
		value       string
		shouldFind  string
		shouldNotFind string
	}{
		{
			name:        "basic replacement",
			line:        "( @@ )",
			slot:        &Slot{Line: 0, Col: 2, Width: 2, Placeholder: "@@"},
			value:       "^^",
			shouldFind:  "^^",
			shouldNotFind: "@@",
		},
		{
			name:        "smaller value",
			line:        "( @@@ )",
			slot:        &Slot{Line: 0, Col: 2, Width: 3, Placeholder: "@@@"},
			value:       "o",
			shouldFind:  "o",
			shouldNotFind: "@@@",
		},
		{
			name:        "empty placeholder",
			line:        "test",
			slot:        &Slot{Line: 0, Col: 0, Width: 2, Placeholder: ""},
			value:       "XX",
			shouldFind:  "test", // Should be unchanged
			shouldNotFind: "XX",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := replaceSlot(tt.line, tt.slot, tt.value)

			if tt.shouldFind != "" && !strings.Contains(result, tt.shouldFind) {
				t.Errorf("Result %q doesn't contain expected %q", result, tt.shouldFind)
			}

			if tt.shouldNotFind != "" && strings.Contains(result, tt.shouldNotFind) {
				t.Errorf("Result %q still contains %q", result, tt.shouldNotFind)
			}
		})
	}
}

// TestCharacterDimensions tests Width and Height methods
func TestCharacterDimensions(t *testing.T) {
	char := &Character{
		Art: []string{
			"12345",
			"123",
			"1234567890",
		},
	}

	// Width should be the longest line
	width := char.Width()
	if width != 10 {
		t.Errorf("Width = %d, want 10", width)
	}

	// Height should be number of lines
	height := char.Height()
	if height != 3 {
		t.Errorf("Height = %d, want 3", height)
	}
}

// TestCharacterAnchors tests anchor point getters
func TestCharacterAnchors(t *testing.T) {
	char := &Character{
		Anchor: Anchor{X: 5, Y: 2},
		Art:    []string{"test"},
	}

	if char.GetAnchorX() != 5 {
		t.Errorf("GetAnchorX = %d, want 5", char.GetAnchorX())
	}

	if char.GetAnchorY() != 2 {
		t.Errorf("GetAnchorY = %d, want 2", char.GetAnchorY())
	}
}

// TestCharacterClone tests cloning a character
func TestCharacterClone(t *testing.T) {
	original := &Character{
		Name:        "test",
		Description: "test description",
		Art:         []string{"line1", "line2"},
		Anchor:      Anchor{X: 1, Y: 2},
		Eyes:        &Slot{Line: 0, Col: 0, Width: 2, Placeholder: "@@"},
		Mouth:       &Slot{Line: 1, Col: 0, Width: 1, Placeholder: "Y"},
	}

	clone := original.Clone()

	// Check basic fields
	if clone.Name != original.Name {
		t.Error("Clone name doesn't match")
	}

	// Check art is copied
	if len(clone.Art) != len(original.Art) {
		t.Error("Clone art length doesn't match")
	}

	// Modify clone and ensure original is unchanged
	clone.Art[0] = "modified"
	if original.Art[0] == "modified" {
		t.Error("Modifying clone affected original art")
	}

	clone.Name = "modified"
	if original.Name == "modified" {
		t.Error("Modifying clone affected original name")
	}

	// Test with nil eyes/mouth
	char := &Character{
		Name: "simple",
		Art:  []string{"test"},
	}
	clone2 := char.Clone()
	if clone2.Eyes != nil {
		t.Error("Clone should have nil Eyes when original does")
	}
}

// TestBuiltinCharacters tests builtin character access
func TestBuiltinCharacters(t *testing.T) {
	chars := BuiltinCharacters()

	if len(chars) == 0 {
		t.Error("No builtin characters found")
	}

	// Check that each builtin character has required fields
	for name, char := range chars {
		if char.Name == "" {
			t.Errorf("Character %s has no name", name)
		}
		if len(char.Art) == 0 {
			t.Errorf("Character %s has no art", name)
		}
	}

	// Test that default character exists
	if _, ok := chars["default"]; !ok {
		t.Error("Default character not found in builtins")
	}
}

// TestGetBuiltinCharacter tests retrieving builtins by name
func TestGetBuiltinCharacter(t *testing.T) {
	// Test valid character
	char, ok := GetBuiltinCharacter("cat")
	if !ok {
		t.Error("Failed to get builtin cat character")
	}
	if char.Name != "cat" {
		t.Errorf("Character name = %s, want cat", char.Name)
	}

	// Test case insensitivity
	char2, ok := GetBuiltinCharacter("CAT")
	if !ok {
		t.Error("Builtin lookup should be case-insensitive")
	}
	if char2.Name != "cat" {
		t.Error("Case-insensitive lookup returned wrong character")
	}

	// Test invalid character
	_, ok = GetBuiltinCharacter("nonexistent")
	if ok {
		t.Error("Should return false for nonexistent character")
	}
}

// TestListBuiltinCharacters tests listing all builtins
func TestListBuiltinCharacters(t *testing.T) {
	list := ListBuiltinCharacters()

	if len(list) == 0 {
		t.Error("List of builtin characters is empty")
	}

	// Check that default is in the list
	found := false
	for _, name := range list {
		if name == "default" {
			found = true
			break
		}
	}
	if !found {
		t.Error("Default character not in list")
	}

	// Verify all listed characters can be retrieved
	for _, name := range list {
		_, ok := GetBuiltinCharacter(name)
		if !ok {
			t.Errorf("Listed character %s cannot be retrieved", name)
		}
	}
}

// TestBuiltinCharacterRendering tests that all builtins render correctly
func TestBuiltinCharacterRendering(t *testing.T) {
	style := lipgloss.NewStyle()
	styles := CharacterStyles{
		Outline: style,
		Eyes:    style,
		Mouth:   style,
	}

	for _, name := range ListBuiltinCharacters() {
		t.Run(name, func(t *testing.T) {
			char, _ := GetBuiltinCharacter(name)

			// Test rendering with expressions
			canvas := char.ToCanvasStyled("^^", "w", styles)
			if canvas == nil {
				t.Error("Rendering returned nil canvas")
			}

			lines := canvas.RenderPlain()
			if len(lines) == 0 {
				t.Error("Rendering returned no lines")
			}

			// Test that rendering doesn't panic
			_ = canvas.Render()
		})
	}
}

// TestEmptyCharacterArt tests handling of empty art
func TestEmptyCharacterArt(t *testing.T) {
	char := &Character{
		Name: "empty",
		Art:  []string{},
	}

	canvas := char.ToCanvas("^^", "w", lipgloss.NewStyle())

	if canvas == nil {
		t.Fatal("ToCanvas returned nil for empty art")
	}

	// Should return a minimal 1x1 canvas
	if canvas.Width == 0 || canvas.Height == 0 {
		t.Error("Canvas has zero dimensions")
	}
}

// TestSlotEdgeCases tests edge cases in slot handling
func TestSlotEdgeCases(t *testing.T) {
	char := &Character{
		Name: "test",
		Art:  []string{"test", "line", "art"},
		Eyes: &Slot{Line: 10, Col: 0, Width: 2, Placeholder: "@@"}, // Out of bounds
	}

	// Should not panic with out-of-bounds slot
	canvas := char.ToCanvas("^^", "w", lipgloss.NewStyle())
	if canvas == nil {
		t.Error("ToCanvas returned nil")
	}

	// Test with very wide replacement
	char2 := builtinCat()
	canvas2 := char2.ToCanvas("^^^^^^^^^^^^", "w", lipgloss.NewStyle())
	if canvas2 == nil {
		t.Error("ToCanvas failed with wide replacement")
	}
}

// TestCharacterWithColors tests character with default colors
func TestCharacterWithColors(t *testing.T) {
	char := &Character{
		Name: "colored",
		Art:  []string{"  test  "},
		Colors: &CharacterColors{
			Outline: "#FF0000",
			Eyes:    "#00FF00",
			Mouth:   "#0000FF",
		},
	}

	if char.Colors == nil {
		t.Error("Character colors not set")
	}

	if char.Colors.Outline != "#FF0000" {
		t.Error("Outline color not set correctly")
	}
}
