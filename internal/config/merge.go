package config

import (
	"strconv"

	"github.com/spf13/cobra"
)

// GetEffectiveConfig returns the effective configuration for a given profile
// Profile values override default values (sparse merging)
func (c *Config) GetEffectiveConfig(profileName string) *FlagConfig {
	// Start with default config
	effective := &FlagConfig{}
	Merge(effective, &c.Default)

	// If profile specified, merge profile values over default
	if profileName != "" && c.Profiles != nil {
		if profile, exists := c.Profiles[profileName]; exists {
			Merge(effective, &profile)
		}
	}

	return effective
}

// Merge merges override values into base (modifies base in place)
// Only non-nil values from override are applied to base
func Merge(base, override *FlagConfig) {
	if override.Character != nil {
		base.Character = override.Character
	}
	if override.Theme != nil {
		base.Theme = override.Theme
	}
	if override.Mood != nil {
		base.Mood = override.Mood
	}
	if override.Width != nil {
		base.Width = override.Width
	}
	if override.Animate != nil {
		base.Animate = override.Animate
	}
	if override.Speed != nil {
		base.Speed = override.Speed
	}
	if override.Effect != nil {
		base.Effect = override.Effect
	}
	if override.Think != nil {
		base.Think = override.Think
	}
	if override.Multipanel != nil {
		base.Multipanel = override.Multipanel
	}
	if override.OutlineColor != nil {
		base.OutlineColor = override.OutlineColor
	}
	if override.EyeColor != nil {
		base.EyeColor = override.EyeColor
	}
	if override.MouthColor != nil {
		base.MouthColor = override.MouthColor
	}
}

// ApplyToFlags applies config values to cobra command flags
// Only applies values if the flag was not explicitly set via CLI
func ApplyToFlags(cfg *FlagConfig, cmd *cobra.Command) {
	flags := cmd.Flags()

	// Apply string flags
	if cfg.Character != nil && !flags.Changed("character") {
		flags.Set("character", *cfg.Character)
	}
	if cfg.Theme != nil && !flags.Changed("theme") {
		flags.Set("theme", *cfg.Theme)
	}
	if cfg.Mood != nil && !flags.Changed("mood") {
		flags.Set("mood", *cfg.Mood)
	}
	if cfg.Effect != nil && !flags.Changed("effect") {
		flags.Set("effect", *cfg.Effect)
	}
	if cfg.OutlineColor != nil && !flags.Changed("outline-color") {
		flags.Set("outline-color", *cfg.OutlineColor)
	}
	if cfg.EyeColor != nil && !flags.Changed("eye-color") {
		flags.Set("eye-color", *cfg.EyeColor)
	}
	if cfg.MouthColor != nil && !flags.Changed("mouth-color") {
		flags.Set("mouth-color", *cfg.MouthColor)
	}

	// Apply int flags
	if cfg.Width != nil && !flags.Changed("width") {
		flags.Set("width", intToString(*cfg.Width))
	}
	if cfg.Speed != nil && !flags.Changed("speed") {
		flags.Set("speed", intToString(*cfg.Speed))
	}

	// Apply bool flags
	if cfg.Animate != nil && !flags.Changed("animate") {
		flags.Set("animate", boolToString(*cfg.Animate))
	}
	if cfg.Think != nil && !flags.Changed("think") {
		flags.Set("think", boolToString(*cfg.Think))
	}
	if cfg.Multipanel != nil && !flags.Changed("multipanel") {
		flags.Set("multipanel", boolToString(*cfg.Multipanel))
	}
}

// Helper functions for type conversion to string (for flags.Set)
func intToString(i int) string {
	return strconv.Itoa(i)
}

func boolToString(b bool) string {
	if b {
		return "true"
	}
	return "false"
}
