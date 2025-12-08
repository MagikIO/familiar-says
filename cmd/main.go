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
	"github.com/MagikIO/familiar-says/internal/effects"
	"github.com/MagikIO/familiar-says/internal/personality"
	"github.com/spf13/cobra"
)

var (
	// Flags
	themeName      string
	moodName       string
	characterName  string
	bubbleWidth    int
	animate        bool
	animSpeed      int
	effect         string
	thinkMode      bool
	listThemes     bool
	listMoods      bool
	listEffects    bool
	listCharacters bool
	multipanel     bool

	// Character color flags
	outlineColor string
	eyeColor     string
	mouthColor   string
	listColors   bool
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
	rootCmd.Flags().BoolVar(&thinkMode, "think", false, "Use thought bubble instead of speech bubble")
	rootCmd.Flags().BoolVarP(&listThemes, "list-themes", "T", false, "List available themes")
	rootCmd.Flags().BoolVarP(&listMoods, "list-moods", "M", false, "List available moods")
	rootCmd.Flags().BoolVarP(&listEffects, "list-effects", "E", false, "List available effects")
	rootCmd.Flags().BoolVarP(&listCharacters, "list-characters", "C", false, "List available characters")
	rootCmd.Flags().BoolVarP(&multipanel, "multipanel", "p", false, "Enable multi-panel mode (experimental)")

	// Character color flags
	rootCmd.Flags().StringVar(&outlineColor, "outline-color", "", "Color for character outline/body (hex, ANSI, or name)")
	rootCmd.Flags().StringVar(&eyeColor, "eye-color", "", "Color for character eyes (hex, ANSI, or name)")
	rootCmd.Flags().StringVar(&mouthColor, "mouth-color", "", "Color for character mouth (hex, ANSI, or name)")
	rootCmd.Flags().BoolVar(&listColors, "list-colors", false, "List available named colors")
}

// Execute runs the root command
func Execute() error {
	return rootCmd.Execute()
}

func runSay(cmd *cobra.Command, args []string) error {
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

	// Get message
	var message string
	if len(args) > 0 {
		message = strings.Join(args, " ")
	} else {
		// Read from stdin if available
		stat, err := os.Stdin.Stat()
		if err == nil && (stat.Mode()&os.ModeCharDevice) == 0 {
			// Data is being piped to stdin
			data, err := io.ReadAll(os.Stdin)
			if err != nil {
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

	// Determine bubble style
	bubbleStyle := bubble.StyleSay
	if thinkMode {
		bubbleStyle = bubble.StyleThink
	}

	// Render with character
	var output []string
	var err error
	if characterName != "" {
		output, err = renderer.RenderByName(message, characterName, bubbleStyle)
		if err != nil {
			return fmt.Errorf("failed to load character: %w", err)
		}
	} else {
		output = renderer.RenderDefault(message, bubbleStyle)
	}

	// Apply visual effects (for effects that apply to full output)
	effectType := effects.Effect(effect)
	output = effects.Apply(output, effectType)

	// Handle animation
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
