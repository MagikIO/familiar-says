package live

import (
	"strings"
	"testing"
	"time"

	"github.com/MagikIO/familiar-says/internal/canvas"
	"github.com/MagikIO/familiar-says/internal/personality"
	tea "github.com/charmbracelet/bubbletea"
)

func testGallery(t *testing.T, current string) *gallery {
	t.Helper()
	var chars []*canvas.Character
	for _, n := range canvas.ListBuiltinCharacters() {
		c, ok := canvas.GetBuiltinCharacter(n)
		if !ok {
			t.Fatalf("no character %q", n)
		}
		chars = append(chars, c)
	}
	g := newGallery(chars, personality.ThemeDefault, current)
	g.Update(tea.WindowSizeMsg{Width: 90, Height: 26})
	return g
}

func press(g *gallery, key string) tea.Cmd {
	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(key)}
	switch key {
	case "enter":
		msg = tea.KeyMsg{Type: tea.KeyEnter}
	case "down":
		msg = tea.KeyMsg{Type: tea.KeyDown}
	case "up":
		msg = tea.KeyMsg{Type: tea.KeyUp}
	case "esc":
		msg = tea.KeyMsg{Type: tea.KeyEsc}
	}
	_, cmd := g.Update(msg)
	return cmd
}

func TestGalleryListsEveryFamiliarAndStartsOnTheCurrentOne(t *testing.T) {
	g := testGallery(t, "owl")
	view := g.View()
	for _, n := range canvas.ListBuiltinCharacters() {
		if !strings.Contains(view, n) {
			t.Errorf("gallery should list %q", n)
		}
	}
	if !strings.Contains(view, "› owl •") {
		t.Errorf("should open on the current familiar and mark it:\n%s", view)
	}
	if !strings.Contains(view, "small enough to perch on a prompt") {
		t.Error("should say that the owl suits a prompt")
	}
}

func TestGalleryBrowsesWrapsAndChooses(t *testing.T) {
	g := testGallery(t, "")
	names := canvas.ListBuiltinCharacters()

	press(g, "up") // wraps from the first to the last
	if got := g.chars[g.idx].Name; got != names[len(names)-1] {
		t.Fatalf("up from the top should wrap to %q, got %q", names[len(names)-1], got)
	}
	press(g, "down")
	press(g, "down")
	if got := g.chars[g.idx].Name; got != names[1] {
		t.Fatalf("expected %q, got %q", names[1], got)
	}

	// Each one visited is alive on the stage and introduces itself.
	for i := 0; i < 40; i++ {
		g.Update(galleryTick(g.last.Add(70 * time.Millisecond)))
	}
	if !g.world().Speaking() {
		t.Errorf("the highlighted familiar should be saying hello:\n%s", g.View())
	}

	if cmd := press(g, "enter"); cmd == nil {
		t.Fatal("enter should quit")
	}
	if g.chosen != names[1] {
		t.Errorf("chose %q, want %q", g.chosen, names[1])
	}
}

func TestGalleryCancelChoosesNothing(t *testing.T) {
	g := testGallery(t, "cat")
	press(g, "down")
	if cmd := press(g, "esc"); cmd == nil {
		t.Fatal("esc should quit")
	}
	if g.chosen != "" {
		t.Errorf("backing out must not choose anything, got %q", g.chosen)
	}
}

func TestGalleryFitsSmallTerminals(t *testing.T) {
	g := testGallery(t, "dragon")
	g.Update(tea.WindowSizeMsg{Width: 50, Height: 16})
	for i := 0; i < len(g.chars); i++ {
		press(g, "down")
		for _, l := range strings.Split(g.View(), "\n") {
			if n := len([]rune(l)); n > 50 {
				t.Fatalf("%s: line is %d wide in a 50-column terminal: %q", g.chars[g.idx].Name, n, l)
			}
		}
	}
}

func TestGalleryHandlesKeyRepeatArrivingInOneMessage(t *testing.T) {
	g := testGallery(t, "")
	press(g, "jjj")
	if want := canvas.ListBuiltinCharacters()[3]; g.chars[g.idx].Name != want {
		t.Errorf("three batched presses should move three places to %q, at %q", want, g.chars[g.idx].Name)
	}
}
