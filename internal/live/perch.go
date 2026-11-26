package live

import (
	"os"
	"regexp"
	"strings"

	"github.com/MagikIO/familiar-says/internal/stage"
	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"
)

// perchWidth leaves the last column free: a prompt line that exactly fills
// the terminal makes some terminals wrap.
func perchWidth(cols int) int {
	if cols <= 0 {
		cols = 80
	}
	return max(20, cols-1)
}

// UseEnvColor picks a color profile from the environment. A headless
// familiar has no terminal to probe, so lipgloss would otherwise conclude it
// should not use color at all.
func UseEnvColor() {
	profile := termenv.ANSI
	term := os.Getenv("TERM")
	switch ct := strings.ToLower(os.Getenv("COLORTERM")); {
	case os.Getenv("NO_COLOR") != "" || term == "dumb":
		profile = termenv.Ascii
	case ct == "truecolor" || ct == "24bit":
		profile = termenv.TrueColor
	case strings.Contains(term, "256color") || strings.Contains(term, "kitty") || strings.Contains(term, "ghostty"):
		profile = termenv.ANSI256
	}
	lipgloss.SetColorProfile(profile)
	stage.ResetStyleCache()
}

var sgr = regexp.MustCompile(`\x1b\[[0-9;]*m`)

// zshPrompt rewrites a frame so zsh can use it inside PROMPT: percent signs
// are doubled so they are not read as prompt escapes, and each colour code is
// wrapped in %{ %} so zsh does not count it towards the prompt's width.
func zshPrompt(frame string) string {
	frame = strings.ReplaceAll(frame, "%", "%%")
	return sgr.ReplaceAllString(frame, "%{$0%}")
}
