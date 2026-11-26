package live

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/MagikIO/familiar-says/internal/ipc"
	"github.com/aymanbagabas/go-osc52/v2"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type (
	frameMsg time.Time
	eventMsg ipc.Event
	taskMsg  struct{ result any }
)

// model is the full-screen companion.
type model struct {
	world  *World
	server *ipc.Server
	hints  *Hints

	last   time.Time
	width  int
	height int

	asking bool
	input  []rune
	help   bool
	notice string
	until  time.Time
}

var (
	dim    = lipgloss.NewStyle().Faint(true)
	accent = lipgloss.NewStyle().Bold(true)
)

// RunTUI runs the companion until the user quits. Events arriving on server
// are fed to the world; server may be nil to run without a socket. hints, if
// set, receives the familiar's latest suggestion for shell hooks to pick up.
func RunTUI(world *World, server *ipc.Server, hints *Hints) error {
	m := &model{world: world, server: server, hints: hints, last: time.Now()}
	_, err := tea.NewProgram(m, tea.WithAltScreen()).Run()
	return err
}

func (m *model) Init() tea.Cmd {
	return tea.Batch(m.frame(), m.listen(), m.dispatch())
}

// frame schedules the next animation frame. The rate follows the action:
// smooth while something moves, a slow heartbeat while nothing does.
func (m *model) frame() tea.Cmd {
	d := 250 * time.Millisecond
	switch {
	case m.world.C.Asleep():
		d = 160 * time.Millisecond
	case m.world.Active():
		d = 60 * time.Millisecond
	}
	return tea.Tick(d, func(t time.Time) tea.Msg { return frameMsg(t) })
}

func (m *model) listen() tea.Cmd {
	if m.server == nil {
		return nil
	}
	return func() tea.Msg { return eventMsg(<-m.server.Events) }
}

// dispatch turns the world's pending slow jobs into background commands.
func (m *model) dispatch() tea.Cmd {
	var cmds []tea.Cmd
	for _, task := range m.world.Tasks() {
		task := task
		cmds = append(cmds, func() tea.Msg { return taskMsg{task()} })
	}
	m.hints.Update(m.world.Suggestion())
	return tea.Batch(cmds...)
}

func (m *model) flash(s string) {
	m.notice, m.until = s, time.Now().Add(3*time.Second)
}

func (m *model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		m.world.Resize(msg.Width, max(msg.Height-1, m.world.MinHeight()))
		return m, nil

	case frameMsg:
		now := time.Time(msg)
		dt := now.Sub(m.last)
		m.last = now
		if dt > 300*time.Millisecond {
			dt = 300 * time.Millisecond // don't lurch after a suspend
		}
		m.world.Step(dt)
		return m, tea.Batch(m.frame(), m.dispatch())

	case eventMsg:
		ev := ipc.Event(msg)
		if ev.Type == ipc.TypeQuit {
			return m, tea.Quit
		}
		m.world.Handle(ev)
		return m, tea.Batch(m.listen(), m.dispatch())

	case taskMsg:
		m.world.Deliver(msg.result)
		return m, m.dispatch()

	case tea.KeyMsg:
		return m.key(msg)
	}
	return m, nil
}

func (m *model) key(k tea.KeyMsg) (tea.Model, tea.Cmd) {
	if k.Type == tea.KeyCtrlC {
		return m, tea.Quit
	}
	if m.asking {
		switch k.Type {
		case tea.KeyEsc:
			m.asking, m.input = false, nil
		case tea.KeyEnter:
			q := strings.TrimSpace(string(m.input))
			m.asking, m.input = false, nil
			if q != "" {
				m.world.Ask(q)
			}
		case tea.KeyBackspace:
			if len(m.input) > 0 {
				m.input = m.input[:len(m.input)-1]
			}
		case tea.KeySpace:
			m.input = append(m.input, ' ')
		case tea.KeyRunes:
			m.input = append(m.input, k.Runes...)
		}
		return m, m.dispatch()
	}

	switch k.String() {
	case "q", "esc":
		return m, tea.Quit
	case " ", "p":
		m.world.Pet()
	case "t":
		m.world.Chat()
	case "a", "/":
		m.asking = true
	case "e":
		m.world.Explain()
	case "z":
		m.world.ToggleNap()
	case "m":
		m.world.ToggleMute()
	case "y":
		if s := m.world.Suggestion(); s != "" {
			_, _ = osc52.New(s).WriteTo(os.Stderr)
			m.flash("copied: " + s)
		} else {
			m.flash("nothing to copy yet")
		}
	case "b":
		m.world.Handle(ipc.Event{Type: ipc.TypeBond})
	case "enter":
		m.world.SkipBubble()
	case "?", "h":
		m.help = !m.help
	}
	return m, m.dispatch()
}

func (m *model) View() string {
	if m.width == 0 {
		return ""
	}
	var foot string
	switch {
	case m.asking:
		foot = accent.Render("ask › ") + string(m.input) + "▏" + dim.Render("   enter to send · esc to cancel")
	case m.help:
		foot = dim.Render("space pet · t talk · a ask · e explain · b bond · y copy suggestion · z nap · m mute · enter skip · q quit")
	case m.notice != "" && time.Now().Before(m.until):
		foot = dim.Render(m.notice)
	default:
		foot = dim.Render(m.world.Status() + "   ? for keys")
		if s := m.world.Suggestion(); s != "" {
			foot = dim.Render(m.world.Status()+"   suggests: ") + accent.Render(s) + dim.Render("   y copies")
		}
	}
	return m.world.Render().String() + "\n" + clipLine(foot, m.width)
}

// clipLine cuts a styled line to the terminal width.
func clipLine(s string, width int) string {
	if lipgloss.Width(s) <= width {
		return s
	}
	return lipgloss.NewStyle().MaxWidth(width).Render(s)
}

// Hints publishes the familiar's latest suggestion to a file, which the shell
// hooks read when the user presses the accept key.
type Hints struct {
	path string
	last string
	init bool
}

// NewHints writes suggestions to path.
func NewHints(path string) *Hints { return &Hints{path: path} }

// Update records the current suggestion if it changed. A nil receiver is a
// no-op, so callers need not care whether hints are enabled.
func (h *Hints) Update(s string) {
	if h == nil || (h.init && s == h.last) {
		return
	}
	h.init, h.last = true, s
	if s == "" {
		_ = os.Remove(h.path)
		return
	}
	_ = writeAtomic(h.path, []byte(s+"\n"))
}

// Close removes the hint file.
func (h *Hints) Close() {
	if h != nil {
		_ = os.Remove(h.path)
	}
}

func writeAtomic(path string, data []byte) error {
	tmp := fmt.Sprintf("%s.%d.tmp", path, os.Getpid())
	if err := os.WriteFile(tmp, data, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}
