package config

// Config represents the structure of the config file
type Config struct {
	Default  FlagConfig            `json:"default"`
	Profiles map[string]FlagConfig `json:"profiles,omitempty"`
}

// FlagConfig represents configuration values for CLI flags
// Pointer fields allow us to distinguish between "not set" and "set to zero value"
type FlagConfig struct {
	Character    *string `json:"character,omitempty"`
	Theme        *string `json:"theme,omitempty"`
	Mood         *string `json:"mood,omitempty"`
	Width        *int    `json:"width,omitempty"`
	Animate      *bool   `json:"animate,omitempty"`
	Speed        *int    `json:"speed,omitempty"`
	Effect       *string `json:"effect,omitempty"`
	Think        *bool   `json:"think,omitempty"`
	Multipanel   *bool   `json:"multipanel,omitempty"`
	OutlineColor *string `json:"outlineColor,omitempty"`
	EyeColor     *string `json:"eyeColor,omitempty"`
	MouthColor   *string `json:"mouthColor,omitempty"`
}

// Helper functions to create pointer values
func stringPtr(s string) *string {
	return &s
}

func intPtr(i int) *int {
	return &i
}

func boolPtr(b bool) *bool {
	return &b
}
