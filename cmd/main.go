package cmd

import (
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/MagikIO/familiar-says/internal/animation"
	"github.com/MagikIO/familiar-says/internal/bubble"
	"github.com/MagikIO/familiar-says/internal/canvas"
	"github.com/MagikIO/familiar-says/internal/character"
	"github.com/MagikIO/familiar-says/internal/config"
	"github.com/MagikIO/familiar-says/internal/effects"
	customerrors "github.com/MagikIO/familiar-says/internal/errors"
	"github.com/MagikIO/familiar-says/internal/personality"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

var (
	// Flags
	themeName       string
	moodName        string
	characterName   string
	bubbleWidth     int
	animate         bool
	animSpeed       int
	effect          string
	thinkMode       bool
	bubbleStyleName string
	tailDirection   string
	listThemes      bool
	listMoods       bool
	listEffects     bool
	listCharacters  bool
	listBubbles     bool
	multipanel      bool

	// Character color flags
	outlineColor string
	eyeColor     string
	mouthColor   string
	listColors   bool

	// Profile flag
	profileName string

	// Character animation flags
	actionName   string
	idleAnim     bool
	animDuration int
	listActions  bool

	// Code bubble flags
	codeLanguage string
	codeStyle    string
	
	// Custom template
	customTemplate string
)

var rootCmd = &cobra.Command{
	Use:   "familiar-says [message]",
	Short: "A personality-themed speech bubble tool with rich styling and animations",
	Long: `familiar-says is a modern replacement for cowsay with:
- Rich styling via Charm/Lipgloss
- Mood-based expressions
- Typing animations
- Dynamic colors and visual effects
- Built-in character familiars (cat, owl, dragon, etc.)
- Multi-panel layouts`,
	RunE: runSay,
}

func init() {
	// Add flags
	rootCmd.Flags().StringVarP(&themeName, "theme", "t", "default", "Theme to use (default, rainbow, cyber, retro)")
	rootCmd.Flags().StringVarP(&moodName, "mood", "m", "neutral", "Mood expression (happy, sad, angry, surprised, bored, excited, neutral, sleepy)")
	rootCmd.Flags().StringVarP(&characterName, "character", "c", "", "Character to use (cat, owl, fox, bunny, penguin, dragon, robot, bat, turtle, default)")
	rootCmd.Flags().IntVarP(&bubbleWidth, "width", "w", 40, "Width of speech bubble")
	rootCmd.Flags().BoolVarP(&animate, "animate", "a", false, "Enable typing animation")
	rootCmd.Flags().IntVarP(&animSpeed, "speed", "s", 50, "Animation speed in milliseconds")
	rootCmd.Flags().StringVarP(&effect, "effect", "e", "none", "Visual effect (none, confetti, fireworks, sparkle, rainbow, rainbow-text)")
	rootCmd.Flags().BoolVar(&thinkMode, "think", false, "Use thought bubble (deprecated: use --bubble-style think)")
	rootCmd.Flags().StringVar(&bubbleStyleName, "bubble-style", "say", "Bubble style (say, think, shout, whisper, song, code)")
	rootCmd.Flags().StringVar(&tailDirection, "tail-direction", "down", "Tail direction (down, up, left, right)")
	rootCmd.Flags().BoolVarP(&listThemes, "list-themes", "T", false, "List available themes")
	rootCmd.Flags().BoolVarP(&listMoods, "list-moods", "M", false, "List available moods")
	rootCmd.Flags().BoolVarP(&listEffects, "list-effects", "E", false, "List available effects")
	rootCmd.Flags().BoolVarP(&listCharacters, "list-characters", "C", false, "List available characters")
	rootCmd.Flags().BoolVar(&listBubbles, "list-bubbles", false, "List available bubble styles")
	rootCmd.Flags().BoolVarP(&multipanel, "multipanel", "p", false, "Enable multi-panel mode (experimental)")

	// Character color flags
	rootCmd.Flags().StringVar(&outlineColor, "outline-color", "", "Color for character outline/body (hex, ANSI, or name)")
	rootCmd.Flags().StringVar(&eyeColor, "eye-color", "", "Color for character eyes (hex, ANSI, or name)")
	rootCmd.Flags().StringVar(&mouthColor, "mouth-color", "", "Color for character mouth (hex, ANSI, or name)")
	rootCmd.Flags().BoolVar(&listColors, "list-colors", false, "List available named colors")

	// Profile flag
	rootCmd.Flags().StringVar(&profileName, "profile", "", "Configuration profile to use")

	// Character animation flags
	rootCmd.Flags().StringVar(&actionName, "action", "none", "Character action animation (wave, jump, blink, etc.)")
	rootCmd.Flags().BoolVar(&idleAnim, "idle", false, "Enable idle animation (blink, breathe)")
	rootCmd.Flags().IntVar(&animDuration, "duration", 0, "Animation duration in ms (0 = until keypress)")
	rootCmd.Flags().BoolVar(&listActions, "list-actions", false, "List available character actions")

	// Code bubble flags
	rootCmd.Flags().StringVar(&codeLanguage, "code-language", "", "Language for syntax highlighting (go, python, javascript, etc.)")
	rootCmd.Flags().StringVar(&codeStyle, "code-style", "monokai", "Syntax highlighting theme (monokai, dracula, github, etc.)")
	
	// Custom template
	rootCmd.Flags().StringVar(&customTemplate, "custom-bubble", "", "Path to custom bubble template JSON file or template name in ~/.config/familiar-says/bubbles/")
}

// Execute runs the root command
func Execute() error {
	return rootCmd.Execute()
}

func runSay(cmd *cobra.Command, args []string) error {
	// Load config file if it exists
	cfg, loadErr := config.Load()
	if loadErr != nil {
		return fmt.Errorf("config file error: %w", loadErr)
	}

	// Get effective config from file (default + profile)
	var fileConfig *config.FlagConfig
	if cfg != nil {
		fileConfig = cfg.GetEffectiveConfig(profileName)
	} else {
		fileConfig = &config.FlagConfig{}
	}

	// Load environment variables
	envConfig := config.LoadFromEnv()

	// Merge: config file < env vars (env vars override config file)
	mergedConfig := &config.FlagConfig{}
	config.Merge(mergedConfig, fileConfig)
	config.Merge(mergedConfig, envConfig)

	// Apply merged config (will be overridden by explicit CLI flags)
	config.ApplyToFlags(mergedConfig, cmd)

	// CLI flags already have highest precedence (handled by cobra)

	// Handle list commands
	if listThemes {
		fmt.Println("Available themes:")
		for _, t := range personality.AllThemes() {
			fmt.Printf("  - %s\n", t)
		}
		return nil
	}

	if listMoods {
		fmt.Println("Available moods:")
		for _, m := range personality.AllMoods() {
			fmt.Printf("  - %s\n", m)
		}
		return nil
	}

	if listEffects {
		fmt.Println("Available effects:")
		for _, e := range effects.AllEffects() {
			fmt.Printf("  - %s: %s\n", e, effects.GetEffectDescription(e))
		}
		return nil
	}

	if listCharacters {
		fmt.Println("Available characters:")
		for _, c := range character.ListCharacters() {
			fmt.Printf("  - %s\n", c)
		}
		return nil
	}

	if listColors {
		fmt.Println("Available named colors:")
		fmt.Println("  Basic: black, white, red, green, blue, yellow, cyan, magenta")
		fmt.Println("  Extended: orange, pink, purple, violet, brown, gray, gold, silver")
		fmt.Println("  More: lime, aqua, navy, teal, olive, maroon, coral, salmon")
		fmt.Println("  Themed: fire, ice, forest, midnight, sunset, ocean, lavender, mint")
		fmt.Println("")
		fmt.Println("You can also use:")
		fmt.Println("  Hex codes: #FF6B6B, #F6B, FF6B6B")
		fmt.Println("  ANSI 256:  196, 82, 46")
		return nil
	}

	if listActions {
		fmt.Println("Available character actions:")
		for _, a := range animation.AllActions() {
			fmt.Printf("  - %s: %s\n", a, animation.GetActionDescription(a))
		}
		return nil
	}

	if listBubbles {
		fmt.Println("Available bubble styles:")
		fmt.Println("  - say: Standard speech bubble with < > delimiters")
		fmt.Println("  - think: Thought bubble with ( ) delimiters and 'o' connector")
		fmt.Println("  - shout: Jagged bubble with ^v^ border and !!! suffix")
		fmt.Println("  - whisper: Dotted border with : delimiters and (text)")
		fmt.Println("  - song: Musical bubble with ♪♫ notes and ~ border")
		fmt.Println("  - code: Code block with box drawing chars and syntax highlighting")
		fmt.Println("")
		fmt.Println("Tail directions: down (default), up, left, right")
		return nil
	}

	// Validate flags
	if err := validateFlags(); err != nil {
		return fmt.Errorf("invalid flags: %w", err)
	}

	// Get message
	var message string
	if len(args) > 0 {
		message = strings.Join(args, " ")
	} else {
		// Read from stdin if available
		stat, err := os.Stdin.Stat()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to check stdin: %v\n", err)
			message = "Hello from familiar-says!"
		} else if (stat.Mode() & os.ModeCharDevice) == 0 {
			// Data is being piped to stdin
			data, err := io.ReadAll(os.Stdin)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Warning: failed to read stdin: %v\n", err)
				message = "Hello from familiar-says!"
			} else {
				message = strings.TrimSpace(string(data))
				if message == "" {
					message = "Hello from familiar-says!"
				}
			}
		} else {
			message = "Hello from familiar-says!"
		}
	}

	// Get theme and mood
	theme := personality.GetTheme(themeName)
	mood := personality.Mood(moodName)

	// Create renderer
	renderer := character.NewRenderer(theme, mood, bubbleWidth)

	// Apply character color overrides from CLI flags
	if outlineColor != "" || eyeColor != "" || mouthColor != "" {
		renderer.CharColors = &canvas.CharacterColors{
			Outline: outlineColor,
			Eyes:    eyeColor,
			Mouth:   mouthColor,
		}
	}
	
	// Set custom template if provided
	if customTemplate != "" {
		renderer.CustomTemplate = customTemplate
	}

	// Determine bubble style (--think is deprecated, --bubble-style takes precedence)
	bubbleStyleVal := bubble.ParseStyle(bubbleStyleName)
	if thinkMode && bubbleStyleName == "say" {
		// Backward compatibility: if --think is set and --bubble-style wasn't explicitly changed
		bubbleStyleVal = bubble.StyleThink
	}

	// Determine tail direction
	tailDir := canvas.TailDown
	switch strings.ToLower(tailDirection) {
	case "up":
		tailDir = canvas.TailUp
	case "left":
		tailDir = canvas.TailLeft
	case "right":
		tailDir = canvas.TailRight
	}

	// Convert bubble style to canvas style
	canvasBubbleStyle := canvas.BubbleStyleSay
	switch bubbleStyleVal {
	case bubble.StyleThink:
		canvasBubbleStyle = canvas.BubbleStyleThink
	case bubble.StyleShout:
		canvasBubbleStyle = canvas.BubbleStyleShout
	case bubble.StyleWhisper:
		canvasBubbleStyle = canvas.BubbleStyleWhisper
	case bubble.StyleSong:
		canvasBubbleStyle = canvas.BubbleStyleSong
	case bubble.StyleCode:
		canvasBubbleStyle = canvas.BubbleStyleCode
	}

	// Get expression for mood
	expr := theme.GetExpression(mood)

	// Check if character animation is requested
	wantCharAnim := (actionName != "" && actionName != "none") || idleAnim

	// Load character if specified
	var char *canvas.Character
	if characterName != "" {
		var loadErr error
		char, loadErr = character.LoadCharacter(characterName)
		if loadErr != nil {
			return fmt.Errorf("failed to load character: %w", loadErr)
		}
	} else {
		char, _ = canvas.GetBuiltinCharacter("default")
	}

	// Handle character animation mode
	if wantCharAnim {
		// Determine which animation to use
		var anim *canvas.AnimationSequence

		// Priority: explicit action > idle > character default
		if actionName != "" && actionName != "none" {
			anim = char.GetAnimation(actionName)
			if anim == nil {
				// Action not found for this character, warn and fall back to static
				fmt.Fprintf(os.Stderr, "Warning: character '%s' does not have animation '%s', using static render\n", char.Name, actionName)
			}
		} else if idleAnim {
			// Try idle animation, then blink, then character's default
			anim = char.GetAnimation("idle")
			if anim == nil {
				anim = char.GetAnimation("blink")
			}
			if anim == nil && char.DefaultAnimation != "" {
				anim = char.GetAnimation(char.DefaultAnimation)
			}
		}

		if anim != nil {
			// Configure character animation
			config := animation.CharacterAnimationConfig{
				Character:    char,
				Animation:    anim,
				BubbleText:   message,
				BubbleWidth:  bubbleWidth,
				BubbleStyle:  canvasBubbleStyle,
				BubbleColor:  theme.BubbleStyle,
				CharColor:    theme.CharacterStyle,
				DefaultEyes:  expr.Eyes,
				DefaultMouth: expr.Tongue,
				Duration:     time.Duration(animDuration) * time.Millisecond,
				Effect:       effects.Effect(effect),
			}

			// Apply character color overrides
			if outlineColor != "" || eyeColor != "" || mouthColor != "" {
				config.CharColors = &canvas.CharacterColors{
					Outline: outlineColor,
					Eyes:    eyeColor,
					Mouth:   mouthColor,
				}
			}

			// Enable typing animation if --animate is set
			if animate {
				config.TypingSpeed = time.Duration(animSpeed) * time.Millisecond
			}

			// Run the character animation
			if err := animation.AnimateCharacter(config); err != nil {
				return fmt.Errorf("character animation failed: %w", err)
			}
			return nil
		}
		// Fall through to static rendering if no animation found
	}

	// Static rendering path (original behavior)
	var output []string
	output = renderer.RenderWithTailDirection(message, char, bubbleStyleVal, tailDir)

	// Apply visual effects (for effects that apply to full output)
	effectType := effects.Effect(effect)
	output = effects.Apply(output, effectType)

	// Handle typing animation
	if animate {
		speed := time.Duration(animSpeed) * time.Millisecond
		if err := animation.Animate(output, animation.AnimationTyping, speed); err != nil {
			return fmt.Errorf("animation failed: %w", err)
		}
	} else {
		for _, line := range output {
			fmt.Println(line)
		}
	}

	return nil
}

// validateFlags validates command-line flags
func validateFlags() error {
	// Validate width
	if bubbleWidth <= 0 {
		return customerrors.NewValidationError("width", bubbleWidth, "must be greater than 0")
	}
	if bubbleWidth > 1000 {
		return customerrors.NewValidationError("width", bubbleWidth, "must be 1000 or less")
	}

	// Validate animation speed
	if animSpeed < 0 {
		return customerrors.NewValidationError("speed", animSpeed, "must be non-negative")
	}
	if animSpeed > 10000 {
		return customerrors.NewValidationError("speed", animSpeed, "must be 10000ms or less")
	}

	// Validate colors if provided
	if outlineColor != "" && !canvas.ValidateColor(outlineColor) {
		return customerrors.NewColorParseError(outlineColor, nil)
	}
	if eyeColor != "" && !canvas.ValidateColor(eyeColor) {
		return customerrors.NewColorParseError(eyeColor, nil)
	}
	if mouthColor != "" && !canvas.ValidateColor(mouthColor) {
		return customerrors.NewColorParseError(mouthColor, nil)
	}

	// Validate action if provided
	if actionName != "" && actionName != "none" && !animation.ValidateAction(actionName) {
		return customerrors.NewValidationError("action", actionName, "unknown action. Use --list-actions to see available actions")
	}

	// Validate duration
	if animDuration < 0 {
		return customerrors.NewValidationError("duration", animDuration, "must be non-negative")
	}

	return nil
}

// getTerminalWidth attempts to detect the terminal width, falling back to a default
func getTerminalWidth() int {
	const defaultWidth = 40
	const maxWidth = 1000

	// Try to get terminal dimensions
	width, _, err := term.GetSize(int(os.Stdout.Fd()))
	if err != nil {
		// Terminal size detection failed, use default
		return defaultWidth
	}

	// Sanity check the width
	if width <= 0 || width > maxWidth {
		return defaultWidth
	}

	// Leave some margin for the speech bubble
	width = width - 10
	if width < 20 {
		width = 20
	}

	return width
}
