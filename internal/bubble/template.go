package bubble

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
)

// TailDirection specifies where the bubble tail points
type TailDirection int

const (
	TailDown  TailDirection = iota // Default: tail points down toward character
	TailUp                         // Tail points up
	TailLeft                       // Tail points left
	TailRight                      // Tail points right
)

// String returns the string representation of a TailDirection
func (d TailDirection) String() string {
	switch d {
	case TailUp:
		return "up"
	case TailLeft:
		return "left"
	case TailRight:
		return "right"
	default:
		return "down"
	}
}

// ParseTailDirection converts a string to TailDirection
func ParseTailDirection(s string) TailDirection {
	switch strings.ToLower(s) {
	case "up":
		return TailUp
	case "left":
		return TailLeft
	case "right":
		return TailRight
	default:
		return TailDown
	}
}

// BubbleTemplate defines the characters and decorations for a bubble style
type BubbleTemplate struct {
	Name string `json:"name"`

	// Border characters
	TopBorder    string `json:"topBorder"`    // Character for top border (repeated)
	BottomBorder string `json:"bottomBorder"` // Character for bottom border (repeated)

	// Corner characters (optional, defaults to space)
	TopLeftCorner     string `json:"topLeftCorner,omitempty"`
	TopRightCorner    string `json:"topRightCorner,omitempty"`
	BottomLeftCorner  string `json:"bottomLeftCorner,omitempty"`
	BottomRightCorner string `json:"bottomRightCorner,omitempty"`

	// Single line delimiters
	SingleLeft  string `json:"singleLeft"`
	SingleRight string `json:"singleRight"`

	// Multi-line delimiters [left, right]
	MultiFirst  [2]string `json:"multiFirst"`
	MultiMiddle [2]string `json:"multiMiddle"`
	MultiLast   [2]string `json:"multiLast"`

	// Connector/tail character
	Connector       string `json:"connector"`
	ConnectorRepeat bool   `json:"connectorRepeat,omitempty"` // If true, repeat connector char; if false, use spaced pattern

	// Decorators (added to bubble)
	Prefix string `json:"prefix,omitempty"` // Added before text (e.g., "♪ " for song)
	Suffix string `json:"suffix,omitempty"` // Added after text (e.g., " ♪" for song)

	// Border decorators (interspersed in borders)
	BorderDecorator string `json:"borderDecorator,omitempty"` // e.g., "♪" for musical notes on border

	// For code blocks
	IsCodeBlock     bool   `json:"isCodeBlock,omitempty"`
	CodeLanguage    string `json:"codeLanguage,omitempty"` // Optional language hint
	SyntaxHighlight bool   `json:"syntaxHighlight,omitempty"`
}

// Built-in templates
var builtinTemplates = map[string]*BubbleTemplate{
	"say": {
		Name:         "say",
		TopBorder:    "_",
		BottomBorder: "-",
		SingleLeft:   "<",
		SingleRight:  ">",
		MultiFirst:   [2]string{"/", "\\"},
		MultiMiddle:  [2]string{"|", "|"},
		MultiLast:    [2]string{"\\", "/"},
		Connector:    "\\",
	},
	"think": {
		Name:         "think",
		TopBorder:    "_",
		BottomBorder: "-",
		SingleLeft:   "(",
		SingleRight:  ")",
		MultiFirst:   [2]string{"(", ")"},
		MultiMiddle:  [2]string{"(", ")"},
		MultiLast:    [2]string{"(", ")"},
		Connector:    "o",
	},
	"shout": {
		Name:              "shout",
		TopBorder:         "^",
		BottomBorder:      "v",
		TopLeftCorner:     "/",
		TopRightCorner:    "\\",
		BottomLeftCorner:  "\\",
		BottomRightCorner: "/",
		SingleLeft:        "<",
		SingleRight:       ">",
		MultiFirst:        [2]string{"<", ">"},
		MultiMiddle:       [2]string{"<", ">"},
		MultiLast:         [2]string{"<", ">"},
		Connector:         "!",
		Suffix:            "!!!",
	},
	"whisper": {
		Name:         "whisper",
		TopBorder:    ".",
		BottomBorder: ".",
		SingleLeft:   ":",
		SingleRight:  ":",
		MultiFirst:   [2]string{":", ":"},
		MultiMiddle:  [2]string{":", ":"},
		MultiLast:    [2]string{":", ":"},
		Connector:    ".",
		Prefix:       "(",
		Suffix:       ")",
	},
	"song": {
		Name:            "song",
		TopBorder:       "~",
		BottomBorder:    "~",
		BorderDecorator: "♪",
		SingleLeft:      "♪",
		SingleRight:     "♫",
		MultiFirst:      [2]string{"♪", "♫"},
		MultiMiddle:     [2]string{"♫", "♪"},
		MultiLast:       [2]string{"♪", "♫"},
		Connector:       "♪",
		Prefix:          "♪ ",
		Suffix:          " ♫",
	},
	"code": {
		Name:            "code",
		TopBorder:       "─",
		BottomBorder:    "─",
		TopLeftCorner:   "┌",
		TopRightCorner:  "┐",
		BottomLeftCorner: "└",
		BottomRightCorner: "┘",
		SingleLeft:      "│",
		SingleRight:     "│",
		MultiFirst:      [2]string{"│", "│"},
		MultiMiddle:     [2]string{"│", "│"},
		MultiLast:       [2]string{"│", "│"},
		Connector:       "│",
		IsCodeBlock:     true,
		SyntaxHighlight: true,
	},
	"dream": {
		Name:         "dream",
		TopBorder:    "☆",
		BottomBorder: "☆",
		SingleLeft:   "✧",
		SingleRight:  "✧",
		MultiFirst:   [2]string{"✧", "✧"},
		MultiMiddle:  [2]string{"✧", "✧"},
		MultiLast:    [2]string{"✧", "✧"},
		Connector:    "✧",
		Prefix:       "✨ ",
		Suffix:       " ✨",
	},
	"angry": {
		Name:              "angry",
		TopBorder:         "#",
		BottomBorder:      "#",
		TopLeftCorner:     "╔",
		TopRightCorner:    "╗",
		BottomLeftCorner:  "╚",
		BottomRightCorner: "╝",
		SingleLeft:        "║",
		SingleRight:       "║",
		MultiFirst:        [2]string{"║", "║"},
		MultiMiddle:       [2]string{"║", "║"},
		MultiLast:         [2]string{"║", "║"},
		Connector:         "!",
		Suffix:            " >:(",
	},
	"box": {
		Name:              "box",
		TopBorder:         "═",
		BottomBorder:      "═",
		TopLeftCorner:     "╔",
		TopRightCorner:    "╗",
		BottomLeftCorner:  "╚",
		BottomRightCorner: "╝",
		SingleLeft:        "║",
		SingleRight:       "║",
		MultiFirst:        [2]string{"║", "║"},
		MultiMiddle:       [2]string{"║", "║"},
		MultiLast:         [2]string{"║", "║"},
		Connector:         "║",
	},
}

// templateCache stores loaded custom templates
var templateCache = make(map[string]*BubbleTemplate)

// GetTemplate returns a bubble template by name.
// It first checks built-in templates, then custom templates.
func GetTemplate(name string) *BubbleTemplate {
	// Normalize name
	name = strings.ToLower(name)

	// Check built-in templates first
	if tmpl, ok := builtinTemplates[name]; ok {
		return tmpl
	}

	// Check cache
	if tmpl, ok := templateCache[name]; ok {
		return tmpl
	}

	// Try to load from custom templates directory
	tmpl, err := loadCustomTemplate(name)
	if err == nil && tmpl != nil {
		templateCache[name] = tmpl
		return tmpl
	}

	// Default to "say" template
	return builtinTemplates["say"]
}

// GetTemplateForStyle converts a Style to its corresponding template
func GetTemplateForStyle(s Style) *BubbleTemplate {
	switch s {
	case StyleThink:
		return GetTemplate("think")
	case StyleShout:
		return GetTemplate("shout")
	case StyleWhisper:
		return GetTemplate("whisper")
	case StyleSong:
		return GetTemplate("song")
	case StyleCode:
		return GetTemplate("code")
	default:
		return GetTemplate("say")
	}
}

// AllTemplates returns a list of all available template names
func AllTemplates() []string {
	names := make([]string, 0, len(builtinTemplates))
	for name := range builtinTemplates {
		names = append(names, name)
	}
	// Add custom templates from cache
	for name := range templateCache {
		if _, exists := builtinTemplates[name]; !exists {
			names = append(names, name)
		}
	}
	return names
}

// RegisterTemplate adds a custom template to the registry
func RegisterTemplate(name string, tmpl *BubbleTemplate) {
	templateCache[strings.ToLower(name)] = tmpl
}

// loadCustomTemplate attempts to load a template from the user's config directory
func loadCustomTemplate(name string) (*BubbleTemplate, error) {
	// Try ~/.config/familiar-says/bubbles/<name>.json
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}

	configPath := filepath.Join(homeDir, ".config", "familiar-says", "bubbles", name+".json")
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, err
	}

	var tmpl BubbleTemplate
	if err := json.Unmarshal(data, &tmpl); err != nil {
		return nil, err
	}

	tmpl.Name = name
	return &tmpl, nil
}

// LoadCustomTemplatesFromDir loads all custom templates from a directory
func LoadCustomTemplatesFromDir(dir string) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // Directory doesn't exist, that's fine
		}
		return err
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		if !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}

		name := strings.TrimSuffix(entry.Name(), ".json")
		path := filepath.Join(dir, entry.Name())

		data, err := os.ReadFile(path)
		if err != nil {
			continue // Skip files we can't read
		}

		var tmpl BubbleTemplate
		if err := json.Unmarshal(data, &tmpl); err != nil {
			continue // Skip invalid JSON
		}

		tmpl.Name = name
		templateCache[name] = &tmpl
	}

	return nil
}

// ValidateTemplate checks if a template name is valid (built-in or loadable)
func ValidateTemplate(name string) bool {
	name = strings.ToLower(name)
	if _, ok := builtinTemplates[name]; ok {
		return true
	}
	if _, ok := templateCache[name]; ok {
		return true
	}
	// Try to load custom template
	tmpl, err := loadCustomTemplate(name)
	if err == nil && tmpl != nil {
		templateCache[name] = tmpl
		return true
	}
	return false
}

// GetDefaultTailDirection returns the default tail direction for a template
func (t *BubbleTemplate) GetDefaultTailDirection() TailDirection {
	return TailDown
}

// LoadTemplateFromFile loads a template from a specific file path.
func LoadTemplateFromFile(path string) (*BubbleTemplate, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var tmpl BubbleTemplate
	if err := json.Unmarshal(data, &tmpl); err != nil {
		return nil, err
	}

	// Use filename as name if not specified
	if tmpl.Name == "" {
		base := filepath.Base(path)
		tmpl.Name = strings.TrimSuffix(base, ".json")
	}

	return &tmpl, nil
}

// GetOrLoadTemplate tries to get a template by name or load it from a file path.
// If the name ends with .json or contains a path separator, it's treated as a file path.
func GetOrLoadTemplate(nameOrPath string) (*BubbleTemplate, error) {
	// Check if it's a file path
	if strings.HasSuffix(nameOrPath, ".json") || strings.Contains(nameOrPath, string(filepath.Separator)) {
		return LoadTemplateFromFile(nameOrPath)
	}
	
	// Try as a template name
	tmpl := GetTemplate(nameOrPath)
	if tmpl != nil {
		return tmpl, nil
	}
	
	return nil, os.ErrNotExist
}
