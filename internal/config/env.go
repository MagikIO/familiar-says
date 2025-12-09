package config

import (
	"os"
	"strconv"
	"strings"
)

// LoadFromEnv parses environment variables and returns a FlagConfig
// All environment variables use the FAMILIAR_SAYS_ prefix
func LoadFromEnv() *FlagConfig {
	cfg := &FlagConfig{}

	if val := os.Getenv("FAMILIAR_SAYS_CHARACTER"); val != "" {
		cfg.Character = stringPtr(val)
	}

	if val := os.Getenv("FAMILIAR_SAYS_THEME"); val != "" {
		cfg.Theme = stringPtr(val)
	}

	if val := os.Getenv("FAMILIAR_SAYS_MOOD"); val != "" {
		cfg.Mood = stringPtr(val)
	}

	if val := os.Getenv("FAMILIAR_SAYS_WIDTH"); val != "" {
		if i, err := strconv.Atoi(val); err == nil {
			cfg.Width = intPtr(i)
		}
	}

	if val := os.Getenv("FAMILIAR_SAYS_ANIMATE"); val != "" {
		if b, ok := parseBool(val); ok {
			cfg.Animate = boolPtr(b)
		}
	}

	if val := os.Getenv("FAMILIAR_SAYS_SPEED"); val != "" {
		if i, err := strconv.Atoi(val); err == nil {
			cfg.Speed = intPtr(i)
		}
	}

	if val := os.Getenv("FAMILIAR_SAYS_EFFECT"); val != "" {
		cfg.Effect = stringPtr(val)
	}

	if val := os.Getenv("FAMILIAR_SAYS_THINK"); val != "" {
		if b, ok := parseBool(val); ok {
			cfg.Think = boolPtr(b)
		}
	}

	if val := os.Getenv("FAMILIAR_SAYS_MULTIPANEL"); val != "" {
		if b, ok := parseBool(val); ok {
			cfg.Multipanel = boolPtr(b)
		}
	}

	if val := os.Getenv("FAMILIAR_SAYS_OUTLINE_COLOR"); val != "" {
		cfg.OutlineColor = stringPtr(val)
	}

	if val := os.Getenv("FAMILIAR_SAYS_EYE_COLOR"); val != "" {
		cfg.EyeColor = stringPtr(val)
	}

	if val := os.Getenv("FAMILIAR_SAYS_MOUTH_COLOR"); val != "" {
		cfg.MouthColor = stringPtr(val)
	}

	return cfg
}

// parseBool parses a boolean value from a string
// Accepts: "true", "false", "1", "0", "yes", "no" (case-insensitive)
// Returns (value, ok) where ok indicates if parsing was successful
func parseBool(s string) (bool, bool) {
	s = strings.ToLower(strings.TrimSpace(s))
	switch s {
	case "true", "1", "yes":
		return true, true
	case "false", "0", "no":
		return false, true
	default:
		return false, false
	}
}
