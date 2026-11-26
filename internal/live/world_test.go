package live

import (
	"os"
	"strings"
	"testing"
	"time"

	"github.com/MagikIO/familiar-says/internal/brain"
	"github.com/MagikIO/familiar-says/internal/canvas"
	"github.com/MagikIO/familiar-says/internal/ipc"
	"github.com/MagikIO/familiar-says/internal/personality"
	tea "github.com/charmbracelet/bubbletea"
)

// clock is a controllable time source.
type clock struct{ t time.Time }

func (c *clock) now() time.Time { return c.t }

func newTestWorld(t *testing.T, name string, compact bool) (*World, *clock) {
	t.Helper()
	char, ok := canvas.GetBuiltinCharacter(name)
	if !ok {
		t.Fatalf("no character %q", name)
	}
	ck := &clock{t: time.Date(2026, 3, 4, 14, 0, 0, 0, time.Local)}
	cfg := brain.DefaultConfig()
	w := NewWorld(Options{
		Character: char, Theme: personality.ThemeDefault, Brain: cfg, Memory: brain.LoadMemory(""),
		Scenery: !compact, Compact: compact, Seed: 7, Now: ck.now,
	})
	w.Tasks() // discard the PATH scan; tests should not depend on the host
	return w, ck
}

// run steps the world for d of simulated time at 20 fps.
func run(w *World, ck *clock, d time.Duration) {
	const dt = 50 * time.Millisecond
	for ; d > 0; d -= dt {
		ck.t = ck.t.Add(dt)
		w.Step(dt)
	}
}

func screen(w *World) string { return strings.Join(w.Render().Plain(), "\n") }

func dump(t *testing.T, label string, w *World) {
	if os.Getenv("FAMILIAR_DUMP") != "" {
		t.Logf("%s [%s]\n%s", label, w.Status(), screen(w))
	}
}

func TestWorldGreetsAndSpeaksAboveFamiliar(t *testing.T) {
	w, ck := newTestWorld(t, "fox", false)
	w.Resize(70, 16)
	run(w, ck, 3*time.Second)
	dump(t, "greeting", w)
	if !w.Speaking() {
		t.Fatal("expected a greeting bubble shortly after start")
	}
	got := screen(w)
	if !strings.Contains(got, "fternoon") && !strings.Contains(got, "Hello again") {
		t.Errorf("greeting should suit 14:00, got:\n%s", got)
	}
}

func TestWorldSuggestsTypoFix(t *testing.T) {
	w, ck := newTestWorld(t, "owl", false)
	w.Resize(70, 14)
	run(w, ck, 30*time.Second) // let the greeting pass
	w.Handle(ipc.Event{Type: ipc.TypeCommand, Cmd: "gti status", Exit: 127, Shell: "fish"})
	run(w, ck, 3*time.Second)
	dump(t, "typo", w)
	if got := screen(w); !strings.Contains(got, "Did you mean `git status`?") {
		t.Errorf("expected a correction, got:\n%s", got)
	}
}

func TestWorldCompactPutsBubbleBeside(t *testing.T) {
	w, ck := newTestWorld(t, "bunny", true)
	w.Resize(100, w.MinHeight())
	run(w, ck, 30*time.Second)
	w.Say("hello from the side")
	run(w, ck, 2*time.Second)
	dump(t, "compact", w)
	lines := w.Render().Plain()
	if len(lines) != w.MinHeight() {
		t.Fatalf("compact stage should stay %d rows, got %d", w.MinHeight(), len(lines))
	}
	if !strings.Contains(strings.Join(lines, "\n"), "hello from the side") {
		t.Errorf("bubble text missing:\n%s", strings.Join(lines, "\n"))
	}
}

func TestWorldLongSpeechIsPaged(t *testing.T) {
	w, ck := newTestWorld(t, "bunny", true)
	w.Resize(60, w.MinHeight())
	run(w, ck, 30*time.Second)
	long := strings.Repeat("words and more words ", 12)
	w.Say(long)
	seen := map[string]bool{}
	for i := 0; i < 400; i++ {
		run(w, ck, 250*time.Millisecond)
		for _, f := range strings.Fields(screen(w)) {
			seen[f] = true
		}
		for _, l := range w.Render().Plain() {
			if len([]rune(l)) > w.W {
				t.Fatalf("row wider than the stage: %q", l)
			}
		}
		if !w.Speaking() && i > 20 {
			break
		}
	}
	if w.Speaking() {
		t.Fatal("speech never finished")
	}
	if !seen["words"] || !seen["more"] {
		t.Error("paged speech lost its text")
	}
}

func TestWorldNapsAndWakes(t *testing.T) {
	w, ck := newTestWorld(t, "cat", false)
	w.Resize(60, 18)
	run(w, ck, 6*time.Minute)
	dump(t, "asleep", w)
	if !w.C.Asleep() {
		t.Fatalf("expected the familiar to be asleep after a quiet spell, status %q", w.Status())
	}
	w.Handle(ipc.Event{Type: ipc.TypeCommand, Cmd: "ls", Cwd: ""})
	if w.C.Asleep() {
		t.Fatal("a command should wake it")
	}
}

func TestWorldStaysInBounds(t *testing.T) {
	for _, name := range canvas.ListBuiltinCharacters() {
		w, ck := newTestWorld(t, name, false)
		w.Resize(50, w.MinHeight()+6)
		for i := 0; i < 600; i++ {
			run(w, ck, 500*time.Millisecond)
			if i%40 == 0 {
				w.Handle(ipc.Event{Type: ipc.TypeCommand, Cmd: "make test", Exit: i % 3})
				w.Pet()
			}
			cx, _ := w.C.Offset()
			lo, hi := w.C.Extent()
			if cx+lo < -1 || cx+hi > w.W {
				t.Fatalf("%s wandered off stage: x=%d ink=[%d,%d] width=%d", name, cx, lo, hi, w.W)
			}
		}
		dump(t, name, w)
	}
}

func TestCommentaryIsClearedByTheNextCommandButRequestsStay(t *testing.T) {
	w, ck := newTestWorld(t, "owl", false)
	w.Resize(70, 14)
	run(w, ck, 30*time.Second)

	w.Handle(ipc.Event{Type: ipc.TypeCommand, Cmd: "gti status", Exit: 127, Shell: "fish"})
	run(w, ck, time.Second)
	if !w.Speaking() || w.Suggestion() != "git status" {
		t.Fatalf("expected a suggestion on screen, speaking=%v suggestion=%q", w.Speaking(), w.Suggestion())
	}
	w.Handle(ipc.Event{Type: ipc.TypeCommand, Cmd: "true", Shell: "fish"})
	if w.Speaking() || w.Suggestion() != "" {
		t.Errorf("a new command should clear stale commentary: speaking=%v suggestion=%q", w.Speaking(), w.Suggestion())
	}

	w.Say("remember the milk")
	run(w, ck, time.Second)
	w.Handle(ipc.Event{Type: ipc.TypeCommand, Cmd: "true", Shell: "fish"})
	if !w.Speaking() || !strings.Contains(screen(w), "remember") {
		t.Error("something the user asked it to say should survive the next command")
	}
}

func TestMuteSilencesBanterButNotAdvice(t *testing.T) {
	w, ck := newTestWorld(t, "owl", false)
	w.Resize(70, 14)
	run(w, ck, 30*time.Second)
	w.ToggleMute()
	w.Chat()
	run(w, ck, time.Second)
	// Chat is an explicit request, so it is allowed through; idle chatter is not.
	run(w, ck, 40*time.Minute)
	if w.Speaking() && !w.C.Asleep() {
		t.Errorf("a muted familiar should not be chattering:\n%s", screen(w))
	}
	w.Handle(ipc.Event{Type: ipc.TypeCommand, Cmd: "gti status", Exit: 127})
	run(w, ck, time.Second)
	if !strings.Contains(screen(w), "git status") {
		t.Error("muting must not suppress corrections")
	}
}

func TestHintsFileTracksSuggestion(t *testing.T) {
	path := t.TempDir() + "/hint"
	h := NewHints(path)
	h.Update("git status")
	if data, err := os.ReadFile(path); err != nil || string(data) != "git status\n" {
		t.Fatalf("hint file should hold the suggestion: %q %v", data, err)
	}
	h.Update("")
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Error("clearing the suggestion should remove the file")
	}
	var none *Hints
	none.Update("x") // must be safe when hints are disabled
	none.Close()
}

func TestRenderNeverExceedsStage(t *testing.T) {
	for _, compact := range []bool{false, true} {
		w, ck := newTestWorld(t, "dragon", compact)
		for _, width := range []int{20, 37, 80, 200} {
			h := w.MinHeight()
			if !compact {
				h += 8
			}
			w.Resize(width, h)
			w.Say("a reasonably long sentence that will need wrapping at narrow widths")
			run(w, ck, 2*time.Second)
			lines := w.Render().Plain()
			if len(lines) != h {
				t.Fatalf("compact=%v width=%d: %d rows, want %d", compact, width, len(lines), h)
			}
			for _, l := range lines {
				if n := len([]rune(l)); n > width {
					t.Fatalf("compact=%v width=%d: row is %d wide: %q", compact, width, n, l)
				}
			}
			run(w, ck, 30*time.Second)
		}
	}
}

func TestTalkingToTheFamiliarIsNotACommandToReactTo(t *testing.T) {
	w, ck := newTestWorld(t, "owl", false)
	w.Resize(70, 14)
	run(w, ck, 30*time.Second)
	w.Pet()
	run(w, ck, 500*time.Millisecond)
	before := screen(w)
	// The shell hook reports `familiar pet` itself as a finished command.
	w.Handle(ipc.Event{Type: ipc.TypeCommand, Cmd: "familiar pet", Shell: "fish"})
	if screen(w) != before {
		t.Errorf("reporting the familiar's own command should change nothing:\n%s\n---\n%s", before, screen(w))
	}
	if n := w.Brain.Memory().Commands; n != 0 {
		t.Errorf("it should not be counted as work, counted %d", n)
	}
}

func TestBecomeChangesShapeInPlace(t *testing.T) {
	w, ck := newTestWorld(t, "cat", true)
	w.opt.LoadCharacter = func(name string) (*canvas.Character, error) {
		c, ok := canvas.GetBuiltinCharacter(name)
		if !ok {
			return nil, os.ErrNotExist
		}
		return c, nil
	}
	w.Resize(90, w.MinHeight())
	run(w, ck, 30*time.Second)
	tall := w.MinHeight()

	w.Handle(ipc.Event{Type: ipc.TypeBecome, Text: "owl"})
	run(w, ck, time.Second)
	if got := screen(w); !strings.Contains(got, ",_,") {
		t.Errorf("it should now be an owl:\n%s", got)
	}
	if w.MinHeight() >= tall {
		t.Errorf("an owl needs a shorter perch than a cat: %d vs %d", w.MinHeight(), tall)
	}
	if !strings.HasPrefix(w.Status(), "owl") {
		t.Errorf("status should name the new shape: %q", w.Status())
	}

	w.Handle(ipc.Event{Type: ipc.TypeBecome, Text: "unicorn"})
	run(w, ck, 3*time.Second)
	if !strings.Contains(screen(w), "unicorn") || !strings.Contains(screen(w), ",_,") {
		t.Errorf("an unknown shape should be declined and the owl kept:\n%s", screen(w))
	}
}

func TestAccessoryIsDrawnOnTheHead(t *testing.T) {
	w, ck := newTestWorld(t, "owl", true)
	w.Resize(60, w.MinHeight())
	mem := w.Brain.Memory()
	mem.Days, mem.Commands = 3, 80
	run(w, ck, 30*time.Second)
	w.Handle(ipc.Event{Type: ipc.TypeWear, Text: "flower"})
	run(w, ck, 8*time.Second)
	lines := w.Render().Plain()
	if !strings.Contains(lines[0], "✿") {
		t.Errorf("the flower should sit in the row above its head:\n%s", strings.Join(lines, "\n"))
	}
}

func TestZshPromptEscaping(t *testing.T) {
	got := zshPrompt("\x1b[38;5;213m100% `done`\x1b[0m\nplain")
	want := "%{\x1b[38;5;213m%}100%% `done`%{\x1b[0m%}\nplain"
	if got != want {
		t.Errorf("got %q\nwant %q", got, want)
	}
}

func TestWorkshopReloadsOnSaveAndSurvivesBrokenFiles(t *testing.T) {
	path := t.TempDir() + "/blob.json"
	write := func(body string, at time.Time) {
		if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
		os.Chtimes(path, at, at)
	}
	t0 := time.Now().Add(-time.Hour)
	write(`{"name":"blob","art":[" (@@) "," /  \\ "],"anchor":{"x":2},"eyes":{"line":0,"col":2,"width":2,"placeholder":"@@"}}`, t0)

	w, err := newWorkshop(path, personality.ThemeDefault)
	if err != nil {
		t.Fatal(err)
	}
	w.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	if len(w.problems) != 0 || !strings.Contains(w.View(), "no problems found") {
		t.Fatalf("a sound character should pass: %v", w.problems)
	}

	// Saving an edit is picked up, and a mistake in it is explained.
	write(`{"name":"blob","art":[" (@@)~"," /  \\ "],"anchor":{"x":2},"eyes":{"line":0,"col":4,"width":2,"placeholder":"@@"}}`, t0.Add(time.Minute))
	w.reload(false)
	view := w.View()
	if !strings.Contains(view, "(oo)~") {
		t.Errorf("the edited art should be on stage:\n%s", view)
	}
	if !strings.Contains(view, `placeholder "@@" is at column 2, but col says 4`) {
		t.Errorf("the mistake should be explained:\n%s", view)
	}

	// A save that does not parse leaves the last good version up.
	write(`{ "name": `, t0.Add(2*time.Minute))
	w.reload(false)
	view = w.View()
	if !strings.Contains(view, "not reloaded") || !strings.Contains(view, "(oo)~") {
		t.Errorf("a broken file should be reported without losing the preview:\n%s", view)
	}
	if strings.Contains(view, path) {
		t.Errorf("the error should name the cause, not repeat the path:\n%s", view)
	}

	// Untouched files are not re-read.
	before := w.loadedAt
	w.reload(false)
	if !w.loadedAt.Equal(before) {
		t.Error("an unchanged file should not be reloaded")
	}
}

func TestEachPageOfALongAnswerGetsItsReadingTime(t *testing.T) {
	w, ck := newTestWorld(t, "bunny", true)
	w.Resize(70, w.MinHeight())
	run(w, ck, 30*time.Second)
	w.Say(strings.Repeat("alpha ", 20) + strings.Repeat("omega ", 20))
	run(w, ck, 4*time.Second) // typed out, and well past the 1.5s that would let a different remark cut in
	if got := screen(w); !strings.Contains(got, "alpha") {
		t.Errorf("the first page should still be up while it is being read:\n%s", got)
	}
	seenLast := false
	for i := 0; i < 200 && w.Speaking(); i++ {
		run(w, ck, 500*time.Millisecond)
		seenLast = seenLast || strings.Contains(screen(w), "omega")
	}
	if !seenLast {
		t.Error("the later pages should still follow")
	}
}

func TestPerchGrowsForALongAnswerAndShrinksBack(t *testing.T) {
	w, ck := newTestWorld(t, "owl", true)
	w.Resize(110, w.MinHeight())
	run(w, ck, 30*time.Second)
	short := w.MinHeight()

	answer := "familiar-says is a Go replacement for cowsay that draws personality speech bubbles with moods, themes, animations, and effects. " +
		"It also has a live familiar that wanders your terminal, reacts to your commands, and offers tips. " +
		"Many files are uncommitted right now, so git status shows what is in progress."
	w.Say(answer)
	run(w, ck, 9*time.Second)
	if w.H <= short {
		t.Fatalf("the perch should grow to fit a long answer: still %d rows", w.H)
	}
	if w.H > tallest {
		t.Errorf("but not without limit: %d rows", w.H)
	}
	got := screen(w)
	for _, want := range []string{"familiar-says is a Go replacement", "tips.", "uncommitted", "progress."} {
		if !strings.Contains(got, want) {
			t.Errorf("the whole answer should be on screen at once, missing %q:\n%s", want, screen(w))
		}
	}
	for _, l := range w.Render().Plain() {
		if len([]rune(l)) > w.W {
			t.Fatalf("row wider than the stage: %q", l)
		}
	}

	run(w, ck, 40*time.Second)
	if w.Speaking() || w.H != short {
		t.Errorf("once it has finished speaking the perch should be %d rows again, is %d", short, w.H)
	}
}
