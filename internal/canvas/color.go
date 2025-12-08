// Package canvas provides color parsing utilities for character customization.
package canvas

import (
	"regexp"
	"strconv"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// CharacterColors defines customizable color options for character parts.
// All color fields are strings to support JSON serialization and multiple formats.
type CharacterColors struct {
	Outline string `json:"outline,omitempty"` // Color for the character outline/body
	Eyes    string `json:"eyes,omitempty"`    // Color for the eyes
	Mouth   string `json:"mouth,omitempty"`   // Color for the mouth/tongue
}

// CharacterStyles holds resolved lipgloss styles for character rendering.
type CharacterStyles struct {
	Outline lipgloss.Style
	Eyes    lipgloss.Style
	Mouth   lipgloss.Style
}

// namedColors maps common color names to their hex values.
var namedColors = map[string]string{
	// Basic colors
	"black":   "#000000",
	"white":   "#FFFFFF",
	"red":     "#FF0000",
	"green":   "#00FF00",
	"blue":    "#0000FF",
	"yellow":  "#FFFF00",
	"cyan":    "#00FFFF",
	"magenta": "#FF00FF",

	// Extended colors
	"orange":    "#FFA500",
	"pink":      "#FFC0CB",
	"purple":    "#800080",
	"violet":    "#EE82EE",
	"brown":     "#A52A2A",
	"gray":      "#808080",
	"grey":      "#808080",
	"gold":      "#FFD700",
	"silver":    "#C0C0C0",
	"lime":      "#00FF00",
	"aqua":      "#00FFFF",
	"navy":      "#000080",
	"teal":      "#008080",
	"olive":     "#808000",
	"maroon":    "#800000",
	"coral":     "#FF7F50",
	"salmon":    "#FA8072",
	"turquoise": "#40E0D0",
	"indigo":    "#4B0082",
	"crimson":   "#DC143C",
	"tomato":    "#FF6347",

	// Themed colors for familiars
	"fire":     "#FF4500",
	"ice":      "#ADD8E6",
	"forest":   "#228B22",
	"midnight": "#191970",
	"sunset":   "#FF6B6B",
	"ocean":    "#0077BE",
	"lavender": "#E6E6FA",
	"mint":     "#98FF98",
	"peach":    "#FFCBA4",
	"rose":     "#FF007F",
}

// hexColorRegex matches hex color codes (#RGB, #RRGGBB, or RRGGBB without #).
var hexColorRegex = regexp.MustCompile(`^#?([0-9A-Fa-f]{3}|[0-9A-Fa-f]{6})$`)

// ParseColor converts a color string to a lipgloss.Color.
// Supports:
//   - Hex codes: "#FF6B6B", "#F6B", "FF6B6B"
//   - ANSI 256 color numbers: "196", "82"
//   - Named colors: "red", "cyan", "fire"
//
// Returns an empty color (default terminal color) if the input is empty or invalid.
func ParseColor(s string) lipgloss.Color {
	s = strings.TrimSpace(s)
	if s == "" {
		return lipgloss.Color("")
	}

	// Check for named color first
	if hex, ok := namedColors[strings.ToLower(s)]; ok {
		return lipgloss.Color(hex)
	}

	// Check for hex color
	if hexColorRegex.MatchString(s) {
		hex := s
		if !strings.HasPrefix(hex, "#") {
			hex = "#" + hex
		}
		// Expand 3-char hex to 6-char
		if len(hex) == 4 {
			r, g, b := hex[1], hex[2], hex[3]
			hex = "#" + string([]byte{r, r, g, g, b, b})
		}
		return lipgloss.Color(hex)
	}

	// Check for ANSI 256 color number (0-255)
	if num, err := strconv.Atoi(s); err == nil && num >= 0 && num <= 255 {
		return lipgloss.Color(s)
	}

	// Unknown format, return as-is (lipgloss will handle it)
	return lipgloss.Color(s)
}

// ParseColorOrDefault parses a color string, returning the default style if empty.
func ParseColorOrDefault(s string, defaultStyle lipgloss.Style) lipgloss.Style {
	if strings.TrimSpace(s) == "" {
		return defaultStyle
	}
	color := ParseColor(s)
	return lipgloss.NewStyle().Foreground(color)
}

// ResolveCharacterStyles converts CharacterColors to CharacterStyles,
// using the fallback style for any unspecified colors.
func ResolveCharacterStyles(colors *CharacterColors, fallback lipgloss.Style) CharacterStyles {
	styles := CharacterStyles{
		Outline: fallback,
		Eyes:    fallback,
		Mouth:   fallback,
	}

	if colors == nil {
		return styles
	}

	if colors.Outline != "" {
		styles.Outline = ParseColorOrDefault(colors.Outline, fallback)
	}
	if colors.Eyes != "" {
		styles.Eyes = ParseColorOrDefault(colors.Eyes, fallback)
	}
	if colors.Mouth != "" {
		styles.Mouth = ParseColorOrDefault(colors.Mouth, fallback)
	}

	return styles
}

// MergeColors merges two CharacterColors, with override taking precedence.
// Empty strings in override don't overwrite base values.
func MergeColors(base, override *CharacterColors) *CharacterColors {
	if base == nil && override == nil {
		return nil
	}

	result := &CharacterColors{}

	if base != nil {
		result.Outline = base.Outline
		result.Eyes = base.Eyes
		result.Mouth = base.Mouth
	}

	if override != nil {
		if override.Outline != "" {
			result.Outline = override.Outline
		}
		if override.Eyes != "" {
			result.Eyes = override.Eyes
		}
		if override.Mouth != "" {
			result.Mouth = override.Mouth
		}
	}

	return result
}

// IsEmpty returns true if no colors are defined.
func (c *CharacterColors) IsEmpty() bool {
	if c == nil {
		return true
	}
	return c.Outline == "" && c.Eyes == "" && c.Mouth == ""
}

// ListNamedColors returns all available named color names.
func ListNamedColors() []string {
	names := make([]string, 0, len(namedColors))
	for name := range namedColors {
		names = append(names, name)
	}
	return names
}

// ValidateColor checks if a color string is valid.
// Returns true if the color can be parsed, false otherwise.
func ValidateColor(s string) bool {
	s = strings.TrimSpace(s)
	if s == "" {
		return true // Empty is valid (means use default)
	}

	// Check named colors
	if _, ok := namedColors[strings.ToLower(s)]; ok {
		return true
	}

	// Check hex
	if hexColorRegex.MatchString(s) {
		return true
	}

	// Check ANSI number
	if num, err := strconv.Atoi(s); err == nil && num >= 0 && num <= 255 {
		return true
	}

	return false
}
