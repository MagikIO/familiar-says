package canvas

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/MagikIO/familiar-says/internal/errors"
	"github.com/charmbracelet/lipgloss"
)

// Slot defines a replaceable region in the character art.
type Slot struct {
	Line        int    `json:"line"`        // 0-indexed line in art
	Col         int    `json:"col"`         // 0-indexed column (start position)
	Width       int    `json:"width"`       // Width of the placeholder
	Placeholder string `json:"placeholder"` // The placeholder string to replace (e.g., "@@")
}

// Anchor defines where the speech bubble connector attaches to the character.
type Anchor struct {
	X int `json:"x"` // Column where connector attaches
	Y int `json:"y"` // Row where connector attaches (usually 0 = top)
}

// AnimationFrame represents a single frame in a character animation.
type AnimationFrame struct {
	DurationMs int      `json:"duration"`          // Duration in milliseconds
	Art        []string `json:"art,omitempty"`     // Full art override (optional)
	Eyes       string   `json:"eyes,omitempty"`    // Eyes expression override (optional)
	Mouth      string   `json:"mouth,omitempty"`   // Mouth expression override (optional)
	OffsetX    int      `json:"offsetX,omitempty"` // Horizontal offset from base position
	OffsetY    int      `json:"offsetY,omitempty"` // Vertical offset from base position
}

// AnimationSequence defines a named animation sequence for a character.
type AnimationSequence struct {
	Frames []AnimationFrame `json:"frames"`
	Loop   bool             `json:"loop"`
}

// Character represents a familiar/animal character with ASCII art and expression slots.
type Character struct {
	Name             string                        `json:"name"`
	Description      string                        `json:"description,omitempty"`
	Art              []string                      `json:"art"`                       // Raw ASCII art lines
	Anchor           Anchor                        `json:"anchor"`                    // Where the thought/speech connector joins
	Eyes             *Slot                         `json:"eyes,omitempty"`            // Where to insert eyes (nil if no eyes)
	Mouth            *Slot                         `json:"mouth,omitempty"`           // Where to insert mouth/tongue (nil if none)
	Colors           *CharacterColors              `json:"colors,omitempty"`          // Default colors for character parts
	Animations       map[string]*AnimationSequence `json:"animations,omitempty"`      // Named animation sequences
	DefaultAnimation string                        `json:"defaultAnimation,omitempty"` // Default animation to play (e.g., "idle")
}

// LoadCharacter loads a character from a JSON file.
func LoadCharacter(filename string) (*Character, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to read character file %q: %w", filename, err)
	}

	var char Character
	if err := json.Unmarshal(data, &char); err != nil {
		return nil, fmt.Errorf("failed to parse character JSON from %q: %w", filename, err)
	}

	// Validate required fields
	if char.Name == "" {
		char.Name = strings.TrimSuffix(filepath.Base(filename), filepath.Ext(filename))
	}
	if len(char.Art) == 0 {
		return nil, errors.ErrEmptyArt
	}

	return &char, nil
}

// ToCanvas renders the character with expressions filled in.
// Deprecated: Use ToCanvasStyled for per-part coloring support.
func (ch *Character) ToCanvas(eyes, mouth string, style lipgloss.Style) *Canvas {
	styles := CharacterStyles{
		Outline: style,
		Eyes:    style,
		Mouth:   style,
	}
	return ch.ToCanvasStyled(eyes, mouth, styles)
}

// ToCanvasStyled renders the character with expressions and per-part styling.
// This allows different colors for the outline, eyes, and mouth.
func (ch *Character) ToCanvasStyled(eyes, mouth string, styles CharacterStyles) *Canvas {
	if len(ch.Art) == 0 {
		return NewCanvas(1, 1)
	}

	// Calculate dimensions
	maxWidth := 0
	for _, line := range ch.Art {
		w := StringWidth(line)
		if w > maxWidth {
			maxWidth = w
		}
	}
	if maxWidth == 0 {
		maxWidth = 1
	}

	canvas := NewCanvas(maxWidth, len(ch.Art))

	// Track which cells are eyes or mouth for special styling
	eyeCells := make(map[int]map[int]bool)  // line -> col -> isEye
	mouthCells := make(map[int]map[int]bool) // line -> col -> isMouth

	// Make a copy of art lines to modify
	lines := make([]string, len(ch.Art))
	copy(lines, ch.Art)

	// Replace eye placeholder and track positions
	if ch.Eyes != nil && ch.Eyes.Line < len(lines) {
		origLine := lines[ch.Eyes.Line]
		newLine := replaceSlot(origLine, ch.Eyes, eyes)
		lines[ch.Eyes.Line] = newLine

		// Mark eye cells for special styling
		eyeCells[ch.Eyes.Line] = make(map[int]bool)
		// Find where the placeholder was and mark new content positions
		placeholderIdx := strings.Index(origLine, ch.Eyes.Placeholder)
		if placeholderIdx >= 0 {
			// Calculate centered position
			valueWidth := StringWidth(eyes)
			slotWidth := ch.Eyes.Width
			if slotWidth < valueWidth {
				slotWidth = valueWidth
			}
			padding := slotWidth - valueWidth
			leftPad := padding / 2

			for i := 0; i < slotWidth; i++ {
				eyeCells[ch.Eyes.Line][placeholderIdx+i] = true
			}
			_ = leftPad // Used in calculation above
		}
	}

	// Replace mouth placeholder and track positions
	if ch.Mouth != nil && ch.Mouth.Line < len(lines) {
		origLine := lines[ch.Mouth.Line]
		newLine := replaceSlot(origLine, ch.Mouth, mouth)
		lines[ch.Mouth.Line] = newLine

		// Mark mouth cells for special styling
		mouthCells[ch.Mouth.Line] = make(map[int]bool)
		placeholderIdx := strings.Index(origLine, ch.Mouth.Placeholder)
		if placeholderIdx >= 0 {
			valueWidth := StringWidth(mouth)
			slotWidth := ch.Mouth.Width
			if slotWidth < valueWidth {
				slotWidth = valueWidth
			}

			for i := 0; i < slotWidth; i++ {
				mouthCells[ch.Mouth.Line][placeholderIdx+i] = true
			}
		}
	}

	// Draw each line with appropriate styling
	for y, line := range lines {
		col := 0
		for _, r := range line {
			// Determine which style to use for this cell
			var cellStyle lipgloss.Style
			if eyeMap, ok := eyeCells[y]; ok && eyeMap[col] {
				cellStyle = styles.Eyes
			} else if mouthMap, ok := mouthCells[y]; ok && mouthMap[col] {
				cellStyle = styles.Mouth
			} else {
				cellStyle = styles.Outline
			}

			canvas.Set(col, y, r, cellStyle)
			col++
		}
	}

	return canvas
}

// replaceSlot replaces a placeholder in a line with the given value, centered within the slot width.
func replaceSlot(line string, slot *Slot, value string) string {
	if slot.Placeholder == "" {
		return line
	}

	// Center the value within the slot width
	valueWidth := StringWidth(value)
	slotWidth := slot.Width
	if slotWidth < valueWidth {
		slotWidth = valueWidth
	}

	// Pad to center
	padding := slotWidth - valueWidth
	leftPad := padding / 2
	rightPad := padding - leftPad

	centered := strings.Repeat(" ", leftPad) + value + strings.Repeat(" ", rightPad)

	// Replace the placeholder
	return strings.Replace(line, slot.Placeholder, centered, 1)
}

// GetAnchorX returns the X coordinate of the anchor point.
func (ch *Character) GetAnchorX() int {
	return ch.Anchor.X
}

// GetAnchorY returns the Y coordinate of the anchor point.
func (ch *Character) GetAnchorY() int {
	return ch.Anchor.Y
}

// Width returns the maximum width of the character art.
func (ch *Character) Width() int {
	maxWidth := 0
	for _, line := range ch.Art {
		w := StringWidth(line)
		if w > maxWidth {
			maxWidth = w
		}
	}
	return maxWidth
}

// Height returns the number of lines in the character art.
func (ch *Character) Height() int {
	return len(ch.Art)
}

// Clone creates a deep copy of the character.
func (ch *Character) Clone() *Character {
	clone := &Character{
		Name:             ch.Name,
		Description:      ch.Description,
		Art:              make([]string, len(ch.Art)),
		Anchor:           ch.Anchor,
		DefaultAnimation: ch.DefaultAnimation,
	}
	copy(clone.Art, ch.Art)

	if ch.Eyes != nil {
		eyes := *ch.Eyes
		clone.Eyes = &eyes
	}
	if ch.Mouth != nil {
		mouth := *ch.Mouth
		clone.Mouth = &mouth
	}
	if ch.Colors != nil {
		colors := *ch.Colors
		clone.Colors = &colors
	}
	if ch.Animations != nil {
		clone.Animations = make(map[string]*AnimationSequence, len(ch.Animations))
		for name, anim := range ch.Animations {
			clonedAnim := &AnimationSequence{
				Frames: make([]AnimationFrame, len(anim.Frames)),
				Loop:   anim.Loop,
			}
			copy(clonedAnim.Frames, anim.Frames)
			clone.Animations[name] = clonedAnim
		}
	}

	return clone
}

// GetAnimation returns the animation sequence by name, or nil if not found.
func (ch *Character) GetAnimation(name string) *AnimationSequence {
	if ch.Animations == nil {
		return nil
	}
	return ch.Animations[name]
}

// HasAnimation returns true if the character has the named animation.
func (ch *Character) HasAnimation(name string) bool {
	if ch.Animations == nil {
		return false
	}
	_, ok := ch.Animations[name]
	return ok
}

// ListAnimations returns the names of all animations defined for this character.
func (ch *Character) ListAnimations() []string {
	if ch.Animations == nil {
		return nil
	}
	names := make([]string, 0, len(ch.Animations))
	for name := range ch.Animations {
		names = append(names, name)
	}
	return names
}

// BuiltinCharacters returns a map of built-in character definitions.
func BuiltinCharacters() map[string]*Character {
	return map[string]*Character{
		"cat":     builtinCat(),
		"owl":     builtinOwl(),
		"fox":     builtinFox(),
		"bunny":   builtinBunny(),
		"penguin": builtinPenguin(),
		"dragon":  builtinDragon(),
		"robot":   builtinRobot(),
		"bat":     builtinBat(),
		"turtle":  builtinTurtle(),
		"default": builtinDefault(),
	}
}

// GetBuiltinCharacter returns a built-in character by name.
func GetBuiltinCharacter(name string) (*Character, bool) {
	chars := BuiltinCharacters()
	char, ok := chars[strings.ToLower(name)]
	return char, ok
}

// ListBuiltinCharacters returns the names of all built-in characters.
func ListBuiltinCharacters() []string {
	return []string{
		"apt",
		"bat",
		"bear",
		"bunny",
		"cat",
		"default",
		"dragon",
		"fox",
		"owl",
		"penguin",
		"robot",
		"suse",
		"turtle",
	}
}

// Built-in character definitions

func builtinCat() *Character {
	return &Character{
		Name:        "cat",
		Description: "A cute cat familiar",
		Art: []string{
			`  /\_/\  `,
			` ( @@ ) `,
			` =( Y )=`,
			`   ^ ^  `,
		},
		Anchor: Anchor{X: 4, Y: 0},
		Eyes: &Slot{
			Line:        1,
			Col:         3,
			Width:       2,
			Placeholder: "@@",
		},
		Mouth: &Slot{
			Line:        2,
			Col:         4,
			Width:       1,
			Placeholder: "Y",
		},
	}
}

func builtinOwl() *Character {
	return &Character{
		Name:        "owl",
		Description: "A wise owl familiar",
		Art: []string{
			`  ,_,  `,
			` (@@)  `,
			` /)_)  `,
			`  ""   `,
		},
		Anchor: Anchor{X: 3, Y: 0},
		Eyes: &Slot{
			Line:        1,
			Col:         2,
			Width:       2,
			Placeholder: "@@",
		},
	}
}

func builtinFox() *Character {
	return &Character{
		Name:        "fox",
		Description: "A clever fox familiar",
		Art: []string{
			`  /\   /\ `,
			` ( @@ )  `,
			`  (Y)    `,
			` /   \   `,
			`(_) (_)  `,
		},
		Anchor: Anchor{X: 4, Y: 0},
		Eyes: &Slot{
			Line:        1,
			Col:         3,
			Width:       2,
			Placeholder: "@@",
		},
		Mouth: &Slot{
			Line:        2,
			Col:         3,
			Width:       1,
			Placeholder: "Y",
		},
	}
}

func builtinBunny() *Character {
	return &Character{
		Name:        "bunny",
		Description: "An adorable bunny familiar",
		Art: []string{
			` (\(\  `,
			` ( @@) `,
			` c(")(")`,
		},
		Anchor: Anchor{X: 3, Y: 0},
		Eyes: &Slot{
			Line:        1,
			Col:         3,
			Width:       2,
			Placeholder: "@@",
		},
	}
}

func builtinPenguin() *Character {
	return &Character{
		Name:        "penguin",
		Description: "A cute Linux penguin familiar",
		Art: []string{
			`   .--.   `,
			`  |@@ |  `,
			`  |:_/|  `,
			`  //  \\  `,
			` (|    |) `,
			` /'\_/'\  `,
			` \_)=(_/  `,
		},
		Anchor: Anchor{X: 5, Y: 0},
		Eyes: &Slot{
			Line:        1,
			Col:         3,
			Width:       2,
			Placeholder: "@@",
		},
	}
}

func builtinDragon() *Character {
	return &Character{
		Name:        "dragon",
		Description: "A mighty dragon familiar",
		Art: []string{
			`      ____ `,
			`     / @@ \`,
			`    /|    |\`,
			`   (_|    |_)`,
			`     \    / `,
			`      \  /  `,
			`       \/   `,
			`      /  \  `,
			`     / /\ \ `,
			`    /_/  \_\`,
		},
		Anchor: Anchor{X: 6, Y: 0},
		Eyes: &Slot{
			Line:        1,
			Col:         7,
			Width:       2,
			Placeholder: "@@",
		},
	}
}

func builtinRobot() *Character {
	return &Character{
		Name:        "robot",
		Description: "A mechanical robot familiar",
		Art: []string{
			`  _____  `,
			` |     | `,
			` | @@  | `,
			` |__Y__| `,
			`   | |   `,
			`  /| |\  `,
			` /_| |_\ `,
		},
		Anchor: Anchor{X: 4, Y: 0},
		Eyes: &Slot{
			Line:        2,
			Col:         3,
			Width:       2,
			Placeholder: "@@",
		},
		Mouth: &Slot{
			Line:        3,
			Col:         4,
			Width:       1,
			Placeholder: "Y",
		},
	}
}

func builtinBat() *Character {
	return &Character{
		Name:        "bat",
		Description: "A spooky bat familiar",
		Art: []string{
			` /\   /\ `,
			`{  @@@  }`,
			` \  Y  / `,
			`  \   /  `,
			`   \_/   `,
		},
		Anchor: Anchor{X: 4, Y: 0},
		Eyes: &Slot{
			Line:        1,
			Col:         4,
			Width:       3,
			Placeholder: "@@@",
		},
		Mouth: &Slot{
			Line:        2,
			Col:         4,
			Width:       1,
			Placeholder: "Y",
		},
	}
}

func builtinTurtle() *Character {
	return &Character{
		Name:        "turtle",
		Description: "A slow and steady turtle familiar",
		Art: []string{
			`    _____    `,
			`  .'     '.  `,
			` / @@      \ `,
			`|   ____    |`,
			`|  /    \   |`,
			` \ \____/  / `,
			`  '._____.'  `,
			`    |   |    `,
			`   _|   |_   `,
		},
		Anchor: Anchor{X: 6, Y: 0},
		Eyes: &Slot{
			Line:        2,
			Col:         3,
			Width:       2,
			Placeholder: "@@",
		},
	}
}

func builtinDefault() *Character {
	return &Character{
		Name:        "default",
		Description: "The classic cow familiar",
		Art: []string{
			`   ^__^    `,
			`   (@@)\_______`,
			`   (__)\       )\/\`,
			`    Y  ||----w |`,
			`       ||     ||`,
		},
		Anchor: Anchor{X: 4, Y: 0},
		Eyes: &Slot{
			Line:        1,
			Col:         4,
			Width:       2,
			Placeholder: "@@",
		},
		Mouth: &Slot{
			Line:        3,
			Col:         4,
			Width:       1,
			Placeholder: "Y",
		},
	}
}
