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
	EffectNone        Effect = "none"
	EffectConfetti    Effect = "confetti"
	EffectFireworks   Effect = "fireworks"
	EffectSparkle     Effect = "sparkle"
	EffectRainbow     Effect = "rainbow"
	EffectRainbowText Effect = "rainbow-text"
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
	case EffectRainbowText:
		return applyRainbowTextOnly(content)
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

// applySparkle adds sparkle effects - adds sparkles to both sides of each line consistently
func applySparkle(content []string) []string {
	sparkles := []string{"✨", "⭐", "🌟", "💫"}

	result := []string{}
	for i, line := range content {
		// Use consistent sparkle pattern based on line index for visual consistency
		sparkle := sparkles[i%len(sparkles)]
		// Add sparkle to both sides of every line to maintain alignment
		line = sparkle + " " + line + " " + sparkle
		result = append(result, line)
	}
	return result
}

// stripAnsi removes ANSI escape codes from a string
func stripAnsi(s string) string {
	result := ""
	inEscape := false
	for _, r := range s {
		if r == '\033' {
			inEscape = true
			continue
		}
		if inEscape {
			// ANSI escape sequences end with a letter
			if (r >= 'A' && r <= 'Z') || (r >= 'a' && r <= 'z') {
				inEscape = false
			}
			continue
		}
		result += string(r)
	}
	return result
}

// applyRainbow applies rainbow colors to the content
func applyRainbow(content []string) []string {
	colors := []lipgloss.Color{"196", "208", "226", "46", "51", "21", "93"}

	result := []string{}
	colorIndex := 0

	for _, line := range content {
		// Strip any existing ANSI codes before applying rainbow
		cleanLine := stripAnsi(line)
		styledLine := ""
		for _, char := range cleanLine {
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
		EffectNone:        "No visual effects",
		EffectConfetti:    "Adds colorful confetti around the output",
		EffectFireworks:   "Creates firework-like bursts of stars",
		EffectSparkle:     "Adds sparkle emojis around the output",
		EffectRainbow:     "Colors each character with rainbow colors",
		EffectRainbowText: "Colors only the message text with rainbow (bubble/character plain)",
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
		EffectRainbowText,
	}
}

// applyRainbowTextOnly colors only the text inside bubble lines, leaving borders and character plain
func applyRainbowTextOnly(content []string) []string {
	colors := []lipgloss.Color{"196", "208", "226", "46", "51", "21", "93"}
	result := []string{}
	colorIndex := 0
	inBubble := false
	bubbleEnded := false

	for _, line := range content {
		// Strip any existing ANSI codes
		cleanLine := stripAnsi(line)
		trimmed := trimSpaces(cleanLine) // trim both leading and trailing

		// Detect bubble boundaries
		// Top border: line of underscores (starts the bubble)
		if !inBubble && !bubbleEnded && len(trimmed) > 0 && isAllChar(trimmed, '_') {
			inBubble = true
			result = append(result, cleanLine)
			continue
		}

		// Bottom border: line of dashes (ends the bubble)
		if inBubble && len(trimmed) > 0 && isAllChar(trimmed, '-') {
			inBubble = false
			bubbleEnded = true
			result = append(result, cleanLine)
			continue
		}

		// If we're inside the bubble and this is a content line
		if inBubble && len(trimmed) > 0 {
			firstChar := trimmed[0]
			// Bubble content lines start with < / | or ( for think bubbles
			if firstChar == '<' || firstChar == '/' || firstChar == '|' || firstChar == '(' {
				styledLine := ""
				inContent := false

				for i, char := range cleanLine {
					// Detect opening border characters
					if char == '<' || char == '(' {
						inContent = true
						styledLine += string(char)
					} else if char == '/' || char == '|' {
						// / at start opens, \ at end closes; | can be either
						if !inContent {
							inContent = true
						}
						styledLine += string(char)
					} else if char == '>' || char == ')' {
						// Closing - check if it's the end border
						remainder := cleanLine[i:]
						if isClosingBorder(remainder) {
							inContent = false
						}
						styledLine += string(char)
					} else if char == '\\' {
						// \ at end of line is closing border
						remainder := cleanLine[i:]
						if isClosingBorder(remainder) {
							inContent = false
						}
						styledLine += string(char)
					} else if inContent && char != ' ' && char != '\t' {
						// Color the content character
						style := lipgloss.NewStyle().Foreground(colors[colorIndex%len(colors)])
						styledLine += style.Render(string(char))
						colorIndex++
					} else {
						styledLine += string(char)
					}
				}
				result = append(result, styledLine)
				continue
			}
		}

		// Not a bubble content line - keep as-is
		result = append(result, cleanLine)
	}

	return result
}

// isAllChar checks if a string consists only of a specific character
func isAllChar(s string, c rune) bool {
	if len(s) == 0 {
		return false
	}
	for _, r := range s {
		if r != c {
			return false
		}
	}
	return true
}

// trimLeftSpaces removes leading spaces from a string
func trimLeftSpaces(s string) string {
	for i, r := range s {
		if r != ' ' && r != '\t' {
			return s[i:]
		}
	}
	return ""
}

// trimSpaces removes leading and trailing spaces from a string
func trimSpaces(s string) string {
	// Trim leading
	start := 0
	for i, r := range s {
		if r != ' ' && r != '\t' {
			start = i
			break
		}
	}
	// Trim trailing
	end := len(s)
	for i := len(s) - 1; i >= start; i-- {
		if s[i] != ' ' && s[i] != '\t' {
			end = i + 1
			break
		}
	}
	if start >= end {
		return ""
	}
	return s[start:end]
}

// isClosingBorder checks if the remainder of the line is just the closing border
func isClosingBorder(s string) bool {
	// After a closing char like > \ ), there should only be spaces left
	if len(s) == 0 {
		return false
	}
	for _, r := range s[1:] {
		if r != ' ' && r != '\t' {
			return false
		}
	}
	return true
}
