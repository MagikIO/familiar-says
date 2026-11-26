package creature

import (
	"math/rand"
	"strings"
	"testing"

	"github.com/MagikIO/familiar-says/internal/canvas"
	"github.com/MagikIO/familiar-says/internal/personality"
)

func newCreature(t *testing.T, name string) *Creature {
	t.Helper()
	char, ok := canvas.GetBuiltinCharacter(name)
	if !ok {
		t.Fatalf("no character %q", name)
	}
	c := New(char, personality.ThemeDefault, nil, rand.New(rand.NewSource(1)))
	c.SetBounds(0, 60, 4)
	return c
}

func look(c *Creature) string { return strings.Join(c.Sprite().Plain(), "\n") }

func step(c *Creature, seconds float64) {
	for ; seconds > 0; seconds -= 0.05 {
		c.Step(0.05)
	}
}

func TestEveryBuiltinCharacterComesToLife(t *testing.T) {
	for _, name := range canvas.ListBuiltinCharacters() {
		c := newCreature(t, name)
		w, h := c.Size()
		lo, hi := c.Extent()
		if lo < 0 || hi >= w || lo > hi {
			t.Errorf("%s: ink extent [%d,%d] outside sprite width %d", name, lo, hi, w)
		}
		for i := 0; i < 2000; i++ { // 100 simulated seconds of whatever it feels like doing
			c.Step(0.05)
			if i%400 == 0 {
				c.React(Act(1 + i/400%10))
			}
			g := c.Sprite()
			if g.H > h || g.W < w {
				t.Fatalf("%s: sprite %dx%d does not fit declared size %dx%d", name, g.W, g.H, w, h)
			}
			if c.X < 0 || c.X > 60 {
				t.Fatalf("%s: left its bounds, x=%v", name, c.X)
			}
			if c.Lift < 0 || c.Lift > 4 {
				t.Fatalf("%s: lift %v outside headroom", name, c.Lift)
			}
		}
		c.Puffs()
	}
}

func TestMoodShowsInTheEyes(t *testing.T) {
	c := newCreature(t, "default")
	c.Feel(personality.MoodHappy, 5)
	if got := look(c); !strings.Contains(got, "(^^)") {
		t.Errorf("a happy cow should have ^^ eyes:\n%s", got)
	}
	c.Feel(personality.MoodSurprised, 5)
	if got := look(c); !strings.Contains(got, "(OO)") {
		t.Errorf("a surprised cow should have OO eyes:\n%s", got)
	}
}

func TestSleepClosesEyesAndPokeWakes(t *testing.T) {
	c := newCreature(t, "default")
	c.Nap()
	if !c.Asleep() || c.Mood() != personality.MoodSleepy {
		t.Fatalf("Nap should put it to sleep, state=%v mood=%v", c.State(), c.Mood())
	}
	if got := look(c); !strings.Contains(got, "(--)") {
		t.Errorf("a sleeping cow should have closed eyes:\n%s", got)
	}
	step(c, 2)
	if len(c.Puffs()) == 0 {
		t.Error("a sleeping familiar should emit some Zs")
	}
	if !c.Poke() {
		t.Error("Poke should report that it woke the familiar")
	}
	if c.Asleep() {
		t.Error("Poke should wake it")
	}
	if c.Poke() {
		t.Error("poking an awake familiar should not report a wake-up")
	}
}

func TestFallsAsleepWhenLeftAlone(t *testing.T) {
	c := newCreature(t, "owl")
	c.SetNapAfter(20)
	step(c, 15)
	if c.Asleep() {
		t.Fatal("dozed off too early")
	}
	step(c, 15)
	if !c.Asleep() {
		t.Errorf("should be asleep after the nap timer ran out, state=%v", c.State())
	}
}

func TestCharactersWithoutEyeSlotsStillBlink(t *testing.T) {
	c := newCreature(t, "cat") // the cat's eyes are part of its art
	awake := look(c)
	c.Nap()
	if asleep := look(c); asleep == awake {
		t.Errorf("a sleeping cat should look different from an awake one:\n%s", asleep)
	}
}

func TestCelebrationLeavesTheGroundAndLands(t *testing.T) {
	c := newCreature(t, "fox")
	c.React(ActCelebrate)
	peak := 0.0
	for i := 0; i < 100; i++ {
		c.Step(0.05)
		if c.Lift > peak {
			peak = c.Lift
		}
	}
	if peak < 1 {
		t.Errorf("celebrating should involve jumping, peak lift %.2f", peak)
	}
	if c.State() == Acting || c.Lift != 0 {
		t.Errorf("it should be back on the ground and done after 5s: state=%v lift=%.2f", c.State(), c.Lift)
	}
	if c.Mood() != personality.MoodHappy && c.Mood() != personality.MoodExcited {
		t.Errorf("it should still be pleased, mood=%v", c.Mood())
	}
}

func TestThinkingLastsUntilSettled(t *testing.T) {
	c := newCreature(t, "owl")
	c.React(ActThink)
	step(c, 20)
	if c.State() != Acting {
		t.Fatalf("thinking should not time out on its own this soon, state=%v", c.State())
	}
	c.Settle()
	if c.State() == Acting {
		t.Error("Settle should end the act")
	}
}

func TestFacingFlipsDirectionalArt(t *testing.T) {
	c := newCreature(t, "default") // the cow is drawn facing left
	c.faceRight = false
	left := look(c)
	c.faceRight = true
	right := look(c)
	if left == right {
		t.Fatal("a directional character should look different facing the other way")
	}
	if !strings.Contains(left, `(oo)\`) || !strings.Contains(right, `/(oo)`) {
		t.Errorf("art should be mirrored, glyphs included:\n%s\n---\n%s", left, right)
	}
	if c.AnchorX() == c.Char.Anchor.X {
		t.Error("the bubble anchor should move with the head when flipped")
	}

	sym := newCreature(t, "fox") // drawn head-on
	sym.faceRight = false
	a := look(sym)
	sym.faceRight = true
	if a != look(sym) {
		t.Error("a front-facing character should never be mirrored")
	}
}

func TestTalkingMovesTheMouthAndRootsItToTheSpot(t *testing.T) {
	c := newCreature(t, "robot")
	quiet := look(c)
	c.Talking(true, true)
	seen := map[string]bool{}
	for i := 0; i < 20; i++ {
		c.Step(0.05)
		seen[look(c)] = true
	}
	if len(seen) < 2 {
		t.Error("the mouth should move while talking")
	}
	c.Talking(false, false)
	c.blinkFor, c.blinkIn = 0, 10
	if look(c) != quiet {
		t.Errorf("it should look as before once it stops talking:\n%s", look(c))
	}

	x := c.X
	c.Talking(false, true)
	for i := 0; i < 400; i++ {
		c.Talking(false, true)
		c.Step(0.05)
	}
	if c.X != x {
		t.Errorf("it should stay put while its bubble is up, moved from %v to %v", x, c.X)
	}
}

func TestFliersHoverAndWalkersDoNot(t *testing.T) {
	bat := newCreature(t, "bat")
	step(bat, 3)
	if bat.Lift < 0.5 {
		t.Errorf("a bat should hover, lift=%.2f", bat.Lift)
	}
	owl := newCreature(t, "owl")
	step(owl, 3)
	if owl.Lift != 0 {
		t.Errorf("an idle walker should be on the ground, lift=%.2f", owl.Lift)
	}
}
