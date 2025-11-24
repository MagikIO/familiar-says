package effects

import (
	"fmt"
	"math/rand"
	"time"

	"github.com/charmbracelet/lipgloss"
)

var rng = rand.New(rand.NewSource(time.Now().UnixNano()))

// Effect represents a visual effect type
type Effect string

const (
	EffectNone      Effect = "none"
	EffectConfetti  Effect = "confetti"
	EffectFireworks Effect = "fireworks"
	EffectSparkle   Effect = "sparkle"
	EffectRainbow   Effect = "rainbow"
)

// Apply applies an effect to the content
func Apply(content []string, effect Effect) []string {
	switch effect {
	case EffectConfetti:
		return applyConfetti(content)
	case EffectFireworks:
		return applyFireworks(content)
	case EffectSparkle:
		return applySparkle(content)
	case EffectRainbow:
		return applyRainbow(content)
	default:
		return content
	}
}

// applyConfetti adds confetti characters around the content
func applyConfetti(content []string) []string {
	confetti := []string{"*", "·", "°", "•", "◦", "∘", "○"}
	colors := []lipgloss.Color{"196", "226", "46", "51", "201", "208"}

	result := []string{}

	// Add confetti header
	header := ""
	for i := 0; i < 60; i++ {
		char := confetti[rng.Intn(len(confetti))]
		color := colors[rng.Intn(len(colors))]
		style := lipgloss.NewStyle().Foreground(color)
		header += style.Render(char) + " "
	}
	result = append(result, header)

	// Add content with occasional confetti
	for _, line := range content {
		if rng.Float64() < 0.3 {
			char := confetti[rng.Intn(len(confetti))]
			color := colors[rng.Intn(len(colors))]
			style := lipgloss.NewStyle().Foreground(color)
			line = style.Render(char) + " " + line + " " + style.Render(char)
		}
		result = append(result, line)
	}

	// Add confetti footer
	footer := ""
	for i := 0; i < 60; i++ {
		char := confetti[rng.Intn(len(confetti))]
		color := colors[rng.Intn(len(colors))]
		style := lipgloss.NewStyle().Foreground(color)
		footer += style.Render(char) + " "
	}
	result = append(result, footer)

	return result
}

// applyFireworks adds firework-like bursts
func applyFireworks(content []string) []string {
	fireworks := []string{"✦", "✧", "★", "☆", "✪", "✫", "✬", "✭", "✮", "✯"}
	colors := []lipgloss.Color{"196", "226", "201", "51"}

	result := []string{}

	// Add fireworks above
	for i := 0; i < 2; i++ {
		line := ""
		for j := 0; j < 50; j++ {
			if rng.Float64() < 0.15 {
				char := fireworks[rng.Intn(len(fireworks))]
				color := colors[rng.Intn(len(colors))]
				style := lipgloss.NewStyle().Foreground(color).Bold(true)
				line += style.Render(char)
			} else {
				line += " "
			}
		}
		result = append(result, line)
	}

	result = append(result, content...)

	// Add fireworks below
	for i := 0; i < 2; i++ {
		line := ""
		for j := 0; j < 50; j++ {
			if rng.Float64() < 0.15 {
				char := fireworks[rng.Intn(len(fireworks))]
				color := colors[rng.Intn(len(colors))]
				style := lipgloss.NewStyle().Foreground(color).Bold(true)
				line += style.Render(char)
			} else {
				line += " "
			}
		}
		result = append(result, line)
	}

	return result
}

// applySparkle adds sparkle effects
func applySparkle(content []string) []string {
	sparkles := []string{"✨", "⭐", "🌟", "💫"}

	result := []string{}
	for _, line := range content {
		if rng.Float64() < 0.4 {
			sparkle := sparkles[rng.Intn(len(sparkles))]
			line = sparkle + " " + line + " " + sparkle
		}
		result = append(result, line)
	}
	return result
}

// applyRainbow applies rainbow colors to the content
func applyRainbow(content []string) []string {
	colors := []lipgloss.Color{"196", "208", "226", "46", "51", "21", "93"}

	result := []string{}
	colorIndex := 0

	for _, line := range content {
		styledLine := ""
		for _, char := range line {
			if char != ' ' && char != '\t' {
				style := lipgloss.NewStyle().Foreground(colors[colorIndex%len(colors)])
				styledLine += style.Render(string(char))
				colorIndex++
			} else {
				styledLine += string(char)
			}
		}
		result = append(result, styledLine)
	}

	return result
}

// AnimateEffect animates an effect (for effects that support animation)
func AnimateEffect(content []string, effect Effect, frames int, delay time.Duration) {
	for i := 0; i < frames; i++ {
		// Clear screen
		fmt.Print("\033[2J\033[H")

		// Apply effect
		styled := Apply(content, effect)
		for _, line := range styled {
			fmt.Println(line)
		}

		time.Sleep(delay)
	}
}

// GetEffectDescription returns a description of the effect
func GetEffectDescription(effect Effect) string {
	descriptions := map[Effect]string{
		EffectNone:      "No visual effects",
		EffectConfetti:  "Adds colorful confetti around the output",
		EffectFireworks: "Creates firework-like bursts of stars",
		EffectSparkle:   "Adds sparkle emojis around the output",
		EffectRainbow:   "Colors each character with rainbow colors",
	}

	if desc, ok := descriptions[effect]; ok {
		return desc
	}
	return "Unknown effect"
}

// AllEffects returns all available effects
func AllEffects() []Effect {
	return []Effect{
		EffectNone,
		EffectConfetti,
		EffectFireworks,
		EffectSparkle,
		EffectRainbow,
	}
}
