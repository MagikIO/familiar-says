package personality

import (
	"testing"
)

func TestGetTheme(t *testing.T) {
	tests := []struct {
		name     string
		expected string
	}{
		{"default", "default"},
		{"rainbow", "rainbow"},
		{"cyber", "cyber"},
		{"retro", "retro"},
		{"unknown", "default"}, // Should return default for unknown
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			theme := GetTheme(tt.name)
			if theme.Name != tt.expected {
				t.Errorf("Expected theme name '%s', got '%s'", tt.expected, theme.Name)
			}
		})
	}
}

func TestThemeGetExpression(t *testing.T) {
	theme := ThemeDefault
	
	// Test valid mood
	expr := theme.GetExpression(MoodHappy)
	if expr.Eyes == "" {
		t.Error("Expected non-empty eyes for happy mood")
	}
	
	// Test that happy mood has different eyes than neutral
	neutralExpr := theme.GetExpression(MoodNeutral)
	if expr.Eyes == neutralExpr.Eyes {
		t.Error("Expected happy and neutral moods to have different expressions")
	}
}

func TestAllThemes(t *testing.T) {
	themes := AllThemes()
	
	if len(themes) == 0 {
		t.Error("Expected at least one theme")
	}
	
	// Check that all returned themes can be loaded
	for _, name := range themes {
		theme := GetTheme(name)
		if theme.Name != name {
			t.Errorf("Theme name mismatch: expected '%s', got '%s'", name, theme.Name)
		}
	}
}

func TestAllMoods(t *testing.T) {
	moods := AllMoods()
	
	if len(moods) == 0 {
		t.Error("Expected at least one mood")
	}
	
	// Check that all moods have expressions in default theme
	theme := ThemeDefault
	for _, mood := range moods {
		expr := theme.GetExpression(mood)
		if expr.Eyes == "" {
			t.Errorf("Expected non-empty eyes for mood %s", mood)
		}
	}
}

func TestThemeColors(t *testing.T) {
	themes := []Theme{ThemeDefault, ThemeRainbow, ThemeCyber, ThemeRetro}
	
	for _, theme := range themes {
		if theme.PrimaryColor == "" {
			t.Errorf("Theme %s has empty primary color", theme.Name)
		}
		if theme.SecondaryColor == "" {
			t.Errorf("Theme %s has empty secondary color", theme.Name)
		}
		if theme.AccentColor == "" {
			t.Errorf("Theme %s has empty accent color", theme.Name)
		}
	}
}

func TestThemeExpressions(t *testing.T) {
	themes := []Theme{ThemeDefault, ThemeRainbow, ThemeCyber, ThemeRetro}
	requiredMoods := []Mood{MoodNeutral, MoodHappy, MoodSad, MoodAngry}
	
	for _, theme := range themes {
		for _, mood := range requiredMoods {
			expr := theme.GetExpression(mood)
			if expr.Eyes == "" {
				t.Errorf("Theme %s missing eyes for mood %s", theme.Name, mood)
			}
			// Tongue can be empty, but should be defined
			if _, ok := theme.Expressions[mood]; !ok {
				t.Errorf("Theme %s missing expression for mood %s", theme.Name, mood)
			}
		}
	}
}

func TestUnknownMoodFallback(t *testing.T) {
	theme := ThemeDefault
	
	// Test with an undefined mood
	expr := theme.GetExpression(Mood("undefined"))
	neutralExpr := theme.GetExpression(MoodNeutral)
	
	if expr.Eyes != neutralExpr.Eyes {
		t.Error("Expected unknown mood to fallback to neutral expression")
	}
}
