package live

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/MagikIO/familiar-says/internal/brain"
	"github.com/MagikIO/familiar-says/internal/canvas"
	"github.com/MagikIO/familiar-says/internal/creature"
	"github.com/MagikIO/familiar-says/internal/personality"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// workshop is a bench for drawing characters: it shows the character in a
// file alive on a stage and reloads it whenever the file is saved, so the
// loop is edit, save, look.
type workshop struct {
	path  string
	theme personality.Theme

	world    *World
	char     *canvas.Character
	anims    []string
	problems []string
	loadErr  error
	loadedAt time.Time
	modTime  time.Time
	checked  time.Time

	mood, act     int
	width, height int
	last          time.Time
}

var (
	workshopMoods = []personality.Mood{personality.MoodNeutral, personality.MoodHappy, personality.MoodSad,
		personality.MoodAngry, personality.MoodSurprised, personality.MoodBored, personality.MoodExcited, personality.MoodSleepy}
	workshopActs = []struct {
		name string
		act  creature.Act
	}{{"celebrate", creature.ActCelebrate}, {"cheer", creature.ActCheer}, {"wince", creature.ActWince},
		{"nervous", creature.ActNervous}, {"love", creature.ActLove}, {"alert", creature.ActAlert},
		{"wave", creature.ActWave}, {"nod", creature.ActNod}, {"sing", creature.ActSing}, {"fume", creature.ActFume}}

	good = lipgloss.NewStyle().Foreground(lipgloss.Color("#87D787"))
	bad  = lipgloss.NewStyle().Foreground(lipgloss.Color("#FF5F5F"))
	note = lipgloss.NewStyle().Foreground(lipgloss.Color("#FFD75F"))
)

// RunWorkshop opens the character file at path on the bench.
func RunWorkshop(path string, theme personality.Theme) error {
	w, err := newWorkshop(path, theme)
	if err != nil {
		return err
	}
	_, err = tea.NewProgram(w, tea.WithAltScreen()).Run()
	return err
}

func newWorkshop(path string, theme personality.Theme) (*workshop, error) {
	w := &workshop{path: path, theme: theme, last: time.Now()}
	char, err := canvas.LoadCharacter(path)
	if err != nil {
		return nil, err // nothing to show yet; a broken first load is the caller's to report
	}
	cfg := brain.DefaultConfig()
	cfg.Talkativeness = 0
	w.world = NewWorld(Options{
		Character: char, Theme: theme, Brain: cfg, Memory: brain.LoadMemory(""),
		Scenery: true, Silent: true, BubbleWidth: 36, Restless: 0.7, Seed: time.Now().UnixNano(),
	})
	w.world.Tasks()
	w.adopt(char)
	if info, err := os.Stat(path); err == nil {
		w.modTime = info.ModTime()
	}
	return w, nil
}

func (w *workshop) adopt(char *canvas.Character) {
	w.char = char
	w.problems = canvas.Lint(char)
	w.anims = w.anims[:0]
	for name := range char.Animations {
		w.anims = append(w.anims, name)
	}
	sort.Strings(w.anims)
	w.loadErr = nil
	w.loadedAt = time.Now()
}

// reload re-reads the file if it has been saved since the last look. A file
// that does not parse leaves the last good version on stage.
func (w *workshop) reload(force bool) {
	info, err := os.Stat(w.path)
	if err != nil {
		w.loadErr = err
		return
	}
	if !force && info.ModTime().Equal(w.modTime) {
		return
	}
	w.modTime = info.ModTime()
	char, err := canvas.LoadCharacter(w.path)
	if err != nil {
		// Show the cause (a JSON syntax error, usually), not the wrapping.
		for inner := errors.Unwrap(err); inner != nil; inner = errors.Unwrap(inner) {
			err = inner
		}
		w.loadErr = err
		return
	}
	w.adopt(char)
	w.world.Replace(char)
	w.resize()
	w.world.C.Feel(workshopMoods[w.mood], 3600)
}

func (w *workshop) panelRows() int { return 5 + min(len(w.problems), 6) }

func (w *workshop) resize() {
	if w.width > 0 {
		w.world.Resize(w.width, max(w.world.MinHeight(), w.height-w.panelRows()))
	}
}

type workshopTick time.Time

func (w *workshop) tick() tea.Cmd {
	return tea.Tick(70*time.Millisecond, func(t time.Time) tea.Msg { return workshopTick(t) })
}

func (w *workshop) Init() tea.Cmd { return w.tick() }

func (w *workshop) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		w.width, w.height = msg.Width, msg.Height
		w.resize()
	case workshopTick:
		now := time.Time(msg)
		dt := min(now.Sub(w.last), 300*time.Millisecond)
		w.last = now
		if now.Sub(w.checked) > 350*time.Millisecond {
			w.checked = now
			w.reload(false)
		}
		w.world.Step(dt)
		return w, w.tick()
	case tea.KeyMsg:
		key := msg.String()
		switch key {
		case "ctrl+c", "q", "esc":
			return w, tea.Quit
		case "m":
			w.mood = (w.mood + 1) % len(workshopMoods)
			w.world.C.Poke()
			w.world.C.Feel(workshopMoods[w.mood], 3600)
		case "r":
			w.world.C.React(workshopActs[w.act].act)
			w.act = (w.act + 1) % len(workshopActs)
		case "f":
			w.world.C.Face(!w.world.C.FacingRight())
		case "w":
			w.world.C.Wander()
		case "t":
			w.world.Say("Testing, testing. Does my mouth move when I talk?")
		case "z":
			w.world.ToggleNap()
		case "l":
			w.reload(true)
		default:
			if len(key) == 1 && key[0] >= '1' && key[0] <= '9' {
				if i := int(key[0] - '1'); i < len(w.anims) {
					w.world.C.Play(w.anims[i])
				}
			}
		}
	}
	return w, nil
}

func (w *workshop) View() string {
	if w.width == 0 {
		return ""
	}
	var panel []string
	head := accent.Render(filepath.Base(w.path)) + dim.Render("  loaded "+w.loadedAt.Format("15:04:05")+", reloads when you save")
	if w.loadErr != nil {
		head = accent.Render(filepath.Base(w.path)) + "  " + bad.Render("not reloaded: "+w.loadErr.Error())
	}
	panel = append(panel, head)

	if len(w.problems) == 0 {
		panel = append(panel, good.Render("no problems found"))
	}
	for i, p := range w.problems {
		if i == 6 {
			panel[len(panel)-1] = note.Render(fmt.Sprintf("! ...and %d more", len(w.problems)-5))
			break
		}
		panel = append(panel, note.Render("! "+p))
	}

	anims := "animations: none"
	if len(w.anims) > 0 {
		parts := make([]string, 0, len(w.anims))
		for i, name := range w.anims {
			if i < 9 {
				parts = append(parts, fmt.Sprintf("%d %s", i+1, name))
			} else {
				parts = append(parts, name)
			}
		}
		anims = "animations: " + strings.Join(parts, "  ")
	}
	panel = append(panel, anims,
		dim.Render(fmt.Sprintf("m mood (%s) · r react (next: %s) · f flip · w wander · t talk · z nap · l reload · q quit",
			workshopMoods[w.mood], workshopActs[w.act].name)))
	for i := range panel {
		panel[i] = clipLine(panel[i], w.width)
	}
	return w.world.Render().String() + "\n" + strings.Join(panel, "\n")
}
