package config

import "encoding/json"

// Config represents the structure of the config file
type Config struct {
	Default  FlagConfig            `json:"default"`
	Profiles map[string]FlagConfig `json:"profiles,omitempty"`
	Live     *LiveConfig           `json:"live,omitempty"`
}

// LiveConfig configures the live familiar (`familiar-says live` and the
// prompt perch). Every field is optional.
type LiveConfig struct {
	Character          *string       `json:"character,omitempty"`          // Which familiar to bring to life
	Theme              *string       `json:"theme,omitempty"`              // Color theme and expressions
	Talkativeness      *float64      `json:"talkativeness,omitempty"`      // 0 (essentials only) to 1 (chatterbox); default 0.5
	Restlessness       *float64      `json:"restlessness,omitempty"`       // How much it wanders; 0 keeps it still, 1 is the default
	LongCommandSeconds *int          `json:"longCommandSeconds,omitempty"` // Announce commands that ran at least this long; default 20
	BreakMinutes       *int          `json:"breakMinutes,omitempty"`       // Suggest a break after this much steady work; 0 disables; default 50
	NapMinutes         *int          `json:"napMinutes,omitempty"`         // Quiet minutes before it dozes off; default 5
	Tips               *bool         `json:"tips,omitempty"`               // Share shell tips when idle; default true
	Scenery            *bool         `json:"scenery,omitempty"`            // Draw the ground and night sky; default true
	BubbleWidth        *int          `json:"bubbleWidth,omitempty"`        // Widest line of speech
	Oracle             *OracleConfig `json:"oracle,omitempty"`             // External command that answers questions

	// Rules teaches the familiar your own reactions: when a command matching
	// a pattern succeeds or fails, say or suggest something. See the README.
	Rules json.RawMessage `json:"rules,omitempty"`
	// Notifications controls desktop notifications when long commands
	// finish: "unfocused" (the default; needs `familiar-says wrap` to know),
	// "always" or "never".
	Notifications *string `json:"notifications,omitempty"`
	// GitHub lets the familiar watch your pull requests and CI with the gh
	// CLI. Off by default: it makes network requests in the background.
	GitHub *bool `json:"github,omitempty"`
}

// OracleConfig names a program the familiar may consult when asked a
// question: anything that reads a prompt on stdin and prints an answer.
type OracleConfig struct {
	Command        []string `json:"command"`                  // e.g. ["claude", "-p"]
	TimeoutSeconds int      `json:"timeoutSeconds,omitempty"` // default 60
	ShareContext   *bool    `json:"shareContext,omitempty"`   // include cwd and last failed command; default true
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
