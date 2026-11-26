package live

import (
	"fmt"
	"math/rand"
	"os"
	"strings"
	"time"

	"github.com/MagikIO/familiar-says/internal/brain"
	"github.com/MagikIO/familiar-says/internal/canvas"
	"github.com/MagikIO/familiar-says/internal/creature"
	"github.com/MagikIO/familiar-says/internal/personality"
	"github.com/MagikIO/familiar-says/internal/stage"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"
)

// gallery is the familiar picker: a list of characters beside a small stage
// where the highlighted one is alive, so you choose by meeting them.
type gallery struct {
	chars   []*canvas.Character
	theme   personality.Theme
	current string // the familiar in use now, marked in the list

	worlds map[int]*World
	idx    int
	chosen string

	width, height int
	last          time.Time
}

// RunGallery shows every character and returns the name of the one the user
// picked, or "" if they backed out.
//
// With onStderr the gallery is drawn on stderr, leaving stdout free for the
// caller to print the answer into a pipe or command substitution.
func RunGallery(chars []*canvas.Character, theme personality.Theme, current string, onStderr bool) (string, error) {
	if len(chars) == 0 {
		return "", fmt.Errorf("no characters to choose from")
	}
	g := newGallery(chars, theme, current)
	opts := []tea.ProgramOption{tea.WithAltScreen()}
	if onStderr {
		// Colour support was probed on stdout, which is a pipe here.
		lipgloss.SetColorProfile(termenv.NewOutput(os.Stderr).EnvColorProfile())
		stage.ResetStyleCache()
		opts = append(opts, tea.WithOutput(os.Stderr))
	}
	if _, err := tea.NewProgram(g, opts...).Run(); err != nil {
		return "", err
	}
	return g.chosen, nil
}

func newGallery(chars []*canvas.Character, theme personality.Theme, current string) *gallery {
	g := &gallery{chars: chars, theme: theme, current: current, worlds: map[int]*World{}, last: time.Now()}
	for i, c := range chars {
		if c.Name == current {
			g.idx = i
		}
	}
	return g
}

func (g *gallery) listWidth() int {
	w := 0
	for _, c := range g.chars {
		w = max(w, len(c.Name))
	}
	return w + 6
}

func (g *gallery) stageSize() (w, h int) {
	return max(30, g.width-g.listWidth()-3), max(8, min(g.height-6, 20))
}

// world returns the stage for the highlighted character, building it the
// first time that character is visited.
func (g *gallery) world() *World {
	if w, ok := g.worlds[g.idx]; ok {
		return w
	}
	cfg := brain.DefaultConfig()
	cfg.Talkativeness = 0
	w := NewWorld(Options{
		Character: g.chars[g.idx], Theme: g.theme, Brain: cfg, Memory: brain.LoadMemory(""),
		Scenery: true, Silent: true, BubbleWidth: 34, Seed: time.Now().UnixNano(),
	})
	w.Tasks() // the picker has no use for the brain's background jobs
	w.Resize(g.stageSize())
	g.worlds[g.idx] = w
	return w
}

// introduce has the highlighted familiar wave and say something in character.
func (g *gallery) introduce() {
	w := g.world()
	w.SkipBubble()
	w.SkipBubble()
	w.C.React(creature.ActWave)
	if sounds := w.C.Traits().Sounds; len(sounds) > 0 {
		w.Say(sounds[rand.Intn(len(sounds))])
	}
}

type galleryTick time.Time

func (g *gallery) tick() tea.Cmd {
	return tea.Tick(70*time.Millisecond, func(t time.Time) tea.Msg { return galleryTick(t) })
}

func (g *gallery) Init() tea.Cmd { return g.tick() }

func (g *gallery) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		first := g.width == 0
		g.width, g.height = msg.Width, msg.Height
		for _, w := range g.worlds {
			w.Resize(g.stageSize())
		}
		if first {
			g.introduce()
		}
	case galleryTick:
		now := time.Time(msg)
		dt := min(now.Sub(g.last), 300*time.Millisecond)
		g.last = now
		if g.width > 0 {
			g.world().Step(dt)
		}
		return g, g.tick()
	case tea.KeyMsg:
		// A held-down key can arrive as several runes in one message.
		keys := []string{msg.String()}
		if msg.Type == tea.KeyRunes && len(msg.Runes) > 1 {
			keys = strings.Split(string(msg.Runes), "")
		}
		for _, key := range keys {
			if g.key(key) {
				return g, tea.Quit
			}
		}
	}
	return g, nil
}

// key handles one keypress and reports whether the gallery should close.
func (g *gallery) key(key string) (quit bool) {
	switch key {
	case "ctrl+c", "q", "esc":
		return true
	case "enter", " ":
		g.chosen = g.chars[g.idx].Name
		return true
	case "up", "k", "left", "h", "shift+tab":
		g.move(-1)
	case "down", "j", "right", "l", "tab":
		g.move(1)
	case "home", "g":
		g.move(-g.idx)
	case "end", "G":
		g.move(len(g.chars) - 1 - g.idx)
	}
	return false
}

func (g *gallery) move(by int) {
	n := len(g.chars)
	next := ((g.idx+by)%n + n) % n
	if next == g.idx {
		return
	}
	g.idx = next
	if g.width > 0 {
		g.introduce()
	}
}

func (g *gallery) View() string {
	if g.width == 0 {
		return ""
	}
	char := g.chars[g.idx]
	w := g.world()

	var list strings.Builder
	for i, c := range g.chars {
		name := c.Name
		if name == g.current {
			name += " •"
		}
		if i == g.idx {
			list.WriteString(accent.Render("› "+name) + "\n")
		} else {
			list.WriteString(dim.Render("  "+name) + "\n")
		}
	}
	left := lipgloss.NewStyle().Width(g.listWidth()).Render(list.String())
	body := lipgloss.JoinHorizontal(lipgloss.Top, left, w.Render().String())

	about := char.Description
	if about == "" {
		about = "A familiar."
	}
	_, rows := w.C.Size()
	traits := []string{w.C.Traits().Locomotion + "s", fmt.Sprintf("%d rows tall", rows)}
	if traits[0] == "s" {
		traits[0] = "walks"
	} else if traits[0] == "flys" {
		traits[0] = "flies"
	}
	if rows <= 6 {
		traits = append(traits, "small enough to perch on a prompt")
	}

	return strings.Join([]string{
		accent.Render("Choose your familiar"),
		"",
		body,
		"",
		clipLine(accent.Render(char.Name)+"  "+about, g.width),
		clipLine(dim.Render(strings.Join(traits, " · ")), g.width),
		clipLine(dim.Render("↑/↓ browse · enter choose · q cancel"+currentNote(g.current)), g.width),
	}, "\n")
}

func currentNote(current string) string {
	if current == "" {
		return ""
	}
	return "   • current"
}
