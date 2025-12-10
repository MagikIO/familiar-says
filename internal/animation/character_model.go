package animation

import (
	"strings"
	"time"

	"github.com/MagikIO/familiar-says/internal/canvas"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// CharacterTickMsg is sent on each character animation frame.
type CharacterTickMsg time.Time

// CharacterAnimationConfig holds configuration for character animation.
type CharacterAnimationConfig struct {
	Character    *canvas.Character
	Animation    *canvas.AnimationSequence
	BubbleText   string
	BubbleWidth  int
	BubbleStyle  canvas.BubbleStyle
	BubbleColor  lipgloss.Style
	CharColors   *canvas.CharacterColors
	CharColor    lipgloss.Style
	DefaultEyes  string
	DefaultMouth string
	TypingSpeed  time.Duration // 0 = no typing animation
	Duration     time.Duration // 0 = until keypress
	FrameRate    time.Duration // Character animation frame rate (default 50ms)
}

// CharacterModel is a Bubble Tea model for character animation with optional typing.
type CharacterModel struct {
	config       CharacterAnimationConfig
	framePlayer  *FramePlayer
	charStyles   canvas.CharacterStyles
	bubbleCanvas *canvas.Canvas
	connCanvas   *canvas.Canvas

	// Typing animation state
	typingEnabled  bool
	typingIndex    int
	typingDone     bool
	lastTypingTick time.Time

	// Duration tracking
	startTime     time.Time
	totalDuration time.Duration

	// General state
	done bool
}

// NewCharacterModel creates a new character animation model.
func NewCharacterModel(config CharacterAnimationConfig) CharacterModel {
	// Set defaults
	if config.FrameRate == 0 {
		config.FrameRate = 50 * time.Millisecond
	}
	if config.BubbleWidth <= 0 {
		config.BubbleWidth = 40
	}

	// Resolve character styles
	mergedColors := canvas.MergeColors(config.Character.Colors, config.CharColors)
	charStyles := canvas.ResolveCharacterStyles(mergedColors, config.CharColor)

	// Create frame player
	var framePlayer *FramePlayer
	if config.Animation != nil {
		framePlayer = NewFramePlayer(
			config.Character,
			config.Animation,
			charStyles,
			config.DefaultEyes,
			config.DefaultMouth,
		)
	}

	// Pre-render static bubble
	bubbleCanvas := canvas.RenderBubble(
		config.BubbleText,
		config.BubbleWidth,
		config.BubbleStyle,
		config.BubbleColor,
	)

	// Generate connector
	connectorChar := "\\"
	if config.BubbleStyle == canvas.BubbleStyleThink {
		connectorChar = "o"
	}
	connCanvas := generateConnectorCanvas(connectorChar, 2, config.Character.GetAnchorX(), config.CharColor)

	return CharacterModel{
		config:        config,
		framePlayer:   framePlayer,
		charStyles:    charStyles,
		bubbleCanvas:  bubbleCanvas,
		connCanvas:    connCanvas,
		typingEnabled: config.TypingSpeed > 0,
		typingIndex:   0,
		typingDone:    config.TypingSpeed == 0,
		done:          false,
	}
}

// Init initializes the model.
func (m CharacterModel) Init() tea.Cmd {
	m.startTime = time.Now()
	return m.tick()
}

// Update handles messages.
func (m CharacterModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case CharacterTickMsg:
		if m.done {
			return m, tea.Quit
		}

		now := time.Time(msg)

		// Check duration limit
		if m.config.Duration > 0 {
			if m.totalDuration == 0 {
				m.startTime = now
			}
			m.totalDuration = now.Sub(m.startTime)
			if m.totalDuration >= m.config.Duration {
				m.done = true
				m.typingDone = true
				return m, tea.Quit
			}
		}

		// Advance typing animation
		if m.typingEnabled && !m.typingDone {
			if m.lastTypingTick.IsZero() || now.Sub(m.lastTypingTick) >= m.config.TypingSpeed {
				m.typingIndex++
				m.lastTypingTick = now

				// Check if typing is complete
				totalChars := m.getTotalChars()
				if m.typingIndex >= totalChars {
					m.typingDone = true
				}
			}
		}

		// Advance character animation
		if m.framePlayer != nil {
			m.framePlayer.Tick(m.config.FrameRate)
		}

		// Check if we should exit (non-looping animation complete and typing done)
		if m.framePlayer != nil && m.framePlayer.IsComplete() && m.typingDone && m.config.Duration == 0 {
			// For non-looping animations, exit when complete
			if !m.framePlayer.GetAnimation().Loop {
				m.done = true
				return m, tea.Quit
			}
		}

		return m, m.tick()

	case tea.KeyMsg:
		// Handle Ctrl+C explicitly
		if msg.Type == tea.KeyCtrlC {
			return m, tea.Quit
		}
		// Exit on any other key press
		m.done = true
		m.typingDone = true
		m.typingIndex = 1000000 // Show all text
		return m, tea.Quit
	}

	return m, nil
}

// View renders the current state.
func (m CharacterModel) View() string {
	// Render current character frame
	var charCanvas *canvas.Canvas
	if m.framePlayer != nil {
		charCanvas = m.framePlayer.Tick(0) // Get current frame without advancing
	} else {
		charCanvas = m.config.Character.ToCanvasStyled(
			m.config.DefaultEyes,
			m.config.DefaultMouth,
			m.charStyles,
		)
	}

	// Compose: bubble + connector + character
	result := canvas.Stack(m.bubbleCanvas, m.connCanvas, 0)
	result = canvas.Stack(result, charCanvas, 0)

	// Get rendered lines
	lines := result.Render()

	// Apply typing effect if enabled and not done
	if m.typingEnabled && !m.typingDone {
		lines = m.applyTypingEffect(lines)
	}

	return strings.Join(lines, "\n")
}

// applyTypingEffect applies the typing reveal effect to the output.
func (m CharacterModel) applyTypingEffect(lines []string) []string {
	result := make([]string, 0, len(lines))
	charCount := 0

	for _, line := range lines {
		if charCount >= m.typingIndex {
			break
		}

		lineLen := len(line)
		if charCount+lineLen <= m.typingIndex {
			result = append(result, line)
			charCount += lineLen
		} else {
			// Partial line
			remaining := m.typingIndex - charCount
			if remaining > lineLen {
				remaining = lineLen
			}
			partial := line[:remaining]
			partial += "▋" // Cursor
			result = append(result, partial)
			break
		}
	}

	return result
}

// getTotalChars returns the total character count for typing animation.
func (m CharacterModel) getTotalChars() int {
	// Render a frame to get total chars
	var charCanvas *canvas.Canvas
	if m.framePlayer != nil {
		charCanvas = m.framePlayer.Tick(0)
	} else {
		charCanvas = m.config.Character.ToCanvasStyled(
			m.config.DefaultEyes,
			m.config.DefaultMouth,
			m.charStyles,
		)
	}

	result := canvas.Stack(m.bubbleCanvas, m.connCanvas, 0)
	result = canvas.Stack(result, charCanvas, 0)

	lines := result.Render()
	total := 0
	for _, line := range lines {
		total += len(line)
	}
	return total
}

// tick returns a command that sends a CharacterTickMsg.
func (m CharacterModel) tick() tea.Cmd {
	return tea.Tick(m.config.FrameRate, func(t time.Time) tea.Msg {
		return CharacterTickMsg(t)
	})
}

// generateConnectorCanvas creates connector lines between bubble and character.
func generateConnectorCanvas(char string, length int, anchorX int, style lipgloss.Style) *canvas.Canvas {
	lines := make([]string, length)
	for i := 0; i < length; i++ {
		indent := anchorX + i
		if indent < 0 {
			indent = 0
		}
		lines[i] = strings.Repeat(" ", indent) + char
	}
	return canvas.FromLines(lines, style)
}

// AnimateCharacter runs the character animation and returns when complete.
func AnimateCharacter(config CharacterAnimationConfig) error {
	p := tea.NewProgram(NewCharacterModel(config))
	if _, err := p.Run(); err != nil {
		return err
	}
	return nil
}
