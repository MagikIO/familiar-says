package cmd

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/MagikIO/familiar-says/internal/animation"
	"github.com/MagikIO/familiar-says/internal/bubble"
	"github.com/MagikIO/familiar-says/internal/character"
	"github.com/MagikIO/familiar-says/internal/effects"
	"github.com/MagikIO/familiar-says/internal/personality"
	"github.com/MagikIO/familiar-says/pkg/cowparser"
	"github.com/spf13/cobra"
)

var (
	// Flags
	themeName     string
	moodName      string
	characterFile string
	bubbleWidth   int
	animate       bool
	animSpeed     int
	effect        string
	thinkMode     bool
	listThemes    bool
	listMoods     bool
	listEffects   bool
	multipanel    bool
)

var rootCmd = &cobra.Command{
	Use:   "familiar-says [message]",
	Short: "A personality-themed speech bubble tool with rich styling and animations",
	Long: `familiar-says is a modern replacement for cowsay with:
- Rich styling via Charm/Lipgloss
- Mood-based expressions
- Typing animations
- Dynamic colors and visual effects
- Support for .cow character files
- Multi-panel layouts`,
	RunE: runSay,
}

func init() {
	// Add flags
	rootCmd.Flags().StringVarP(&themeName, "theme", "t", "default", "Theme to use (default, rainbow, cyber, retro)")
	rootCmd.Flags().StringVarP(&moodName, "mood", "m", "neutral", "Mood expression (happy, sad, angry, surprised, bored, excited, neutral, sleepy)")
	rootCmd.Flags().StringVarP(&characterFile, "character", "c", "", "Path to .cow character file")
	rootCmd.Flags().IntVarP(&bubbleWidth, "width", "w", 40, "Width of speech bubble")
	rootCmd.Flags().BoolVarP(&animate, "animate", "a", false, "Enable typing animation")
	rootCmd.Flags().IntVarP(&animSpeed, "speed", "s", 50, "Animation speed in milliseconds")
	rootCmd.Flags().StringVarP(&effect, "effect", "e", "none", "Visual effect (none, confetti, fireworks, sparkle, rainbow)")
	rootCmd.Flags().BoolVar(&thinkMode, "think", false, "Use thought bubble instead of speech bubble")
	rootCmd.Flags().BoolVarP(&listThemes, "list-themes", "T", false, "List available themes")
	rootCmd.Flags().BoolVarP(&listMoods, "list-moods", "M", false, "List available moods")
	rootCmd.Flags().BoolVarP(&listEffects, "list-effects", "E", false, "List available effects")
	rootCmd.Flags().BoolVarP(&multipanel, "multipanel", "p", false, "Enable multi-panel mode (experimental)")
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

	// Get message
	var message string
	if len(args) > 0 {
		message = strings.Join(args, " ")
	} else {
		// Read from stdin
		data, err := os.ReadFile("/dev/stdin")
		if err != nil {
			message = "Hello from familiar-says!"
		} else {
			message = strings.TrimSpace(string(data))
			if message == "" {
				message = "Hello from familiar-says!"
			}
		}
	}

	// Get theme and mood
	theme := personality.GetTheme(themeName)
	mood := personality.Mood(moodName)

	// Create renderer
	renderer := character.NewRenderer(theme, mood, bubbleWidth)

	// Determine bubble style
	bubbleStyle := bubble.StyleSay
	if thinkMode {
		bubbleStyle = bubble.StyleThink
	}

	// Load character if specified
	var output []string
	if characterFile != "" {
		cow, err := cowparser.Parse(characterFile)
		if err != nil {
			return fmt.Errorf("failed to parse character file: %w", err)
		}
		output = renderer.Render(message, cow, bubbleStyle)
	} else {
		output = renderer.RenderDefault(message, bubbleStyle)
	}

	// Apply visual effects
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
