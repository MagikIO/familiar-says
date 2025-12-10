package config

// Config represents the structure of the config file
type Config struct {
	Default  FlagConfig            `json:"default"`
	Profiles map[string]FlagConfig `json:"profiles,omitempty"`
}

// FlagConfig represents configuration values for CLI flags
// Pointer fields allow us to distinguish between "not set" and "set to zero value"
type FlagConfig struct {
	Character     *string `json:"character,omitempty"`
	Theme         *string `json:"theme,omitempty"`
	Mood          *string `json:"mood,omitempty"`
	Width         *int    `json:"width,omitempty"`
	Animate       *bool   `json:"animate,omitempty"`
	Speed         *int    `json:"speed,omitempty"`
	Effect        *string `json:"effect,omitempty"`
	Think         *bool   `json:"think,omitempty"`          // Deprecated: use BubbleStyle instead
	BubbleStyle   *string `json:"bubbleStyle,omitempty"`    // Bubble style: say, think, shout, whisper, song, code
	TailDirection *string `json:"tailDirection,omitempty"`  // Tail direction: down, up, left, right
	Multipanel    *bool   `json:"multipanel,omitempty"`
	OutlineColor  *string `json:"outlineColor,omitempty"`
	EyeColor      *string `json:"eyeColor,omitempty"`
	MouthColor    *string `json:"mouthColor,omitempty"`
	// Code bubble options
	CodeLanguage  *string `json:"codeLanguage,omitempty"`   // Language for syntax highlighting
	CodeStyle     *string `json:"codeStyle,omitempty"`      // Syntax highlighting theme
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
