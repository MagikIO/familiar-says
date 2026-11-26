package stage

import (
	"math/rand"
	"strings"
	"testing"
	"time"
)

func TestGridTextAndPlain(t *testing.T) {
	g := NewGrid(10, 2)
	if n := g.Text(1, 0, "hi", Paint{}); n != 2 {
		t.Fatalf("Text returned width %d, want 2", n)
	}
	g.Text(8, 1, "wxyz", Paint{}) // runs off the edge; must be clipped, not panic
	got := g.Plain()
	if got[0] != " hi" || got[1] != "        wx" {
		t.Errorf("unexpected rows: %q", got)
	}
	if c := g.At(99, 99); c.R != 0 {
		t.Errorf("out-of-range read should be transparent, got %q", c.R)
	}
}

func TestGridWideRunes(t *testing.T) {
	g := NewGrid(6, 1)
	if n := g.Text(0, 0, "日本", Paint{}); n != 4 {
		t.Fatalf("two wide runes should take 4 columns, took %d", n)
	}
	if got := g.Plain()[0]; got != "日本" {
		t.Errorf("wide runes should round-trip, got %q", got)
	}
	g.Set(1, 0, 'x', Paint{}) // overwrite the second half of 日
	if got := g.Plain()[0]; got != " x本" {
		t.Errorf("overwriting half a wide rune should blank the other half, got %q", got)
	}
}

func TestGridDropsControlCharacters(t *testing.T) {
	g := NewGrid(20, 1)
	g.Text(0, 0, "a\x1b[31mb\x07c", Paint{})
	if got := g.Plain()[0]; strings.ContainsAny(got, "\x1b\x07") {
		t.Errorf("control characters must never reach the terminal, got %q", got)
	}
}

func TestBlitKeepsTransparency(t *testing.T) {
	dst := NewGrid(5, 1)
	dst.Text(0, 0, "abcde", Paint{})
	src := NewGrid(3, 1)
	src.Set(0, 0, 'X', Paint{})
	src.Set(2, 0, 'Z', Paint{})
	dst.Blit(src, 1, 0)
	if got := dst.Plain()[0]; got != "aXcZe" {
		t.Errorf("transparent cells should let the destination show, got %q", got)
	}
	dst.Blit(src, 4, 0)  // partly off the right edge
	dst.Blit(src, -2, 0) // partly off the left edge
	if got := dst.Plain()[0]; got != "ZXcZX" {
		t.Errorf("clipped blit drew the wrong cells: %q", got)
	}
}

func TestFlipMirrorsGlyphs(t *testing.T) {
	g := FromLines([]string{`(o>  \`, `/--b`}, 0, nil)
	got := g.Flip().Plain()
	want := []string{`/  <o)`, `  d--\`}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("row %d: got %q, want %q", i, got[i], want[i])
		}
	}
	if twice := g.Flip().Flip().Plain(); twice[0] != g.Plain()[0] {
		t.Errorf("flipping twice should be the identity, got %q", twice[0])
	}
}

func TestFromLinesSilhouette(t *testing.T) {
	g := FromLines([]string{"  ( )  "}, 0, nil)
	if g.At(0, 0).R != 0 || g.At(6, 0).R != 0 {
		t.Error("padding around the art should be transparent")
	}
	if g.At(3, 0).R != ' ' {
		t.Error("space inside the art should be opaque so the background does not show through")
	}
}

func TestBubbleTypesThenLingersThenEnds(t *testing.T) {
	b := NewBubble("hello there", BubbleOptions{Style: "say", Width: 30, CPS: 10, Hold: time.Second}, Paint{}, Paint{})
	w0, h0 := b.Size()
	if got := strings.Join(b.Grid().Plain(), "\n"); strings.Contains(got, "h") {
		t.Errorf("nothing should be typed yet:\n%s", got)
	}

	b.Step(500 * time.Millisecond)
	mid := strings.Join(b.Grid().Plain(), "\n")
	if !strings.Contains(mid, "hello") || strings.Contains(mid, "there") {
		t.Errorf("half a second at 10 cps should reveal the first word only:\n%s", mid)
	}
	if !b.Typing() {
		t.Error("should still be typing")
	}
	if w, h := b.Grid().W, b.Grid().H; w != w0 || h != h0 {
		t.Errorf("bubble must not resize while typing: %dx%d vs %dx%d", w, h, w0, h0)
	}

	b.Step(time.Second)
	if b.Typing() || b.Done() {
		t.Fatalf("after typing it should linger: typing=%v done=%v", b.Typing(), b.Done())
	}
	b.Step(1100 * time.Millisecond)
	if !b.Done() {
		t.Error("should be done once the hold time has passed")
	}
}

func TestBubbleSkipAndDismiss(t *testing.T) {
	b := NewBubble("a long enough sentence", BubbleOptions{Width: 40, CPS: 1}, Paint{}, Paint{})
	b.Skip()
	if b.Typing() {
		t.Error("Skip should reveal everything")
	}
	if b.Done() {
		t.Error("Skip should not end the bubble")
	}
	b.Dismiss()
	if !b.Done() {
		t.Error("Dismiss should end the bubble")
	}
}

func TestBubbleStylesApplyDecoration(t *testing.T) {
	b := NewBubble("fire", BubbleOptions{Style: "shout", Width: 20}, Paint{}, Paint{})
	if got := strings.Join(b.Grid().Plain(), "\n"); !strings.Contains(got, "fire!!!") {
		t.Errorf("shout template should add its suffix:\n%s", got)
	}
}

func TestParticlesLiveAndDie(t *testing.T) {
	var ps Particles
	rng := rand.New(rand.NewSource(1))
	ps.Emit(Confetti, 10, 5, rng)
	if ps.Len() == 0 {
		t.Fatal("confetti should spawn particles")
	}
	g := NewGrid(20, 10)
	ps.Draw(g) // must not panic for particles that are off-grid
	for i := 0; i < 100; i++ {
		ps.Step(0.1)
	}
	if ps.Len() != 0 {
		t.Errorf("%d particles outlived their lifetime", ps.Len())
	}
	for i := 0; i < 100; i++ {
		ps.Emit(Confetti, 0, 0, rng)
	}
	if ps.Len() > 400 {
		t.Errorf("particle pool should be capped, has %d", ps.Len())
	}
}

func TestSceneryIsStableAndNocturnal(t *testing.T) {
	s := NewScenery(3)
	day, night := NewGrid(60, 12), NewGrid(60, 12)
	s.Draw(day, 11, 13)
	s.Draw(night, 11, 23)
	if strings.TrimSpace(strings.Join(day.Plain()[:11], "")) != "" {
		t.Error("the daytime sky should be empty")
	}
	if strings.TrimSpace(strings.Join(night.Plain()[:11], "")) == "" {
		t.Error("the night sky should have stars")
	}
	again := NewGrid(60, 12)
	s.Draw(again, 11, 13)
	if day.Plain()[11] != again.Plain()[11] {
		t.Error("the ground should not change from frame to frame")
	}
}
