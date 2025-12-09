package animation

import (
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

// TickMsg is sent on each animation frame
type TickMsg time.Time

// Model represents the animation state
type Model struct {
	Content       []string
	CurrentIndex  int
	AnimationType AnimationType
	Speed         time.Duration
	Done          bool
	ShowCursor    bool
	CursorBlink   bool
}

// AnimationType defines the type of animation
type AnimationType int

const (
	AnimationNone AnimationType = iota
	AnimationTyping
	AnimationFadeIn
	AnimationSlide
)

// New creates a new animation model
func New(content []string, animType AnimationType, speed time.Duration) Model {
	if speed == 0 {
		speed = 50 * time.Millisecond
	}
	return Model{
		Content:       content,
		CurrentIndex:  0,
		AnimationType: animType,
		Speed:         speed,
		Done:          animType == AnimationNone,
		ShowCursor:    animType == AnimationTyping,
		CursorBlink:   true,
	}
}

// Init initializes the animation
func (m Model) Init() tea.Cmd {
	if m.AnimationType == AnimationNone {
		return nil
	}
	return tick(m.Speed)
}

// Update handles animation updates
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg.(type) {
	case TickMsg:
		if m.Done {
			return m, nil
		}

		m.CurrentIndex++

		// Calculate total characters
		totalChars := 0
		for _, line := range m.Content {
			totalChars += len(line)
		}

		if m.CurrentIndex >= totalChars {
			m.Done = true
			m.ShowCursor = false
			return m, nil
		}

		return m, tick(m.Speed)

	case tea.KeyMsg:
		// Skip animation on any key press
		if !m.Done {
			m.Done = true
			m.CurrentIndex = 1000000 // Set to very high to show all
			m.ShowCursor = false
			return m, nil
		}
	}

	return m, nil
}

// View renders the current animation frame
func (m Model) View() string {
	if m.AnimationType == AnimationNone || m.Done {
		return strings.Join(m.Content, "\n")
	}

	result := []string{}
	charCount := 0

	for _, line := range m.Content {
		if charCount >= m.CurrentIndex {
			break
		}

		if charCount+len(line) <= m.CurrentIndex {
			result = append(result, line)
			charCount += len(line)
		} else {
			// Partial line
			remaining := m.CurrentIndex - charCount
			partial := line[:remaining]
			if m.ShowCursor && m.CursorBlink {
				partial += "▋"
			}
			result = append(result, partial)
			break
		}
	}

	return strings.Join(result, "\n")
}

// tick returns a command that sends a TickMsg after the given duration
func tick(d time.Duration) tea.Cmd {
	return tea.Tick(d, func(t time.Time) tea.Msg {
		return TickMsg(t)
	})
}

// Animate runs the animation and returns the final output
func Animate(content []string, animType AnimationType, speed time.Duration) error {
	if animType == AnimationNone {
		fmt.Println(strings.Join(content, "\n"))
		return nil
	}

	p := tea.NewProgram(New(content, animType, speed))
	if _, err := p.Run(); err != nil {
		return fmt.Errorf("animation rendering failed: %w", err)
	}
	return nil
}
