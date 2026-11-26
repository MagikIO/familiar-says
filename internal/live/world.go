// Package live runs the familiar in real time. A World ties the creature, its
// brain, the particle effects and the speech bubble together into something
// that can be stepped and drawn; the front-ends in this package (the
// full-screen companion and the headless prompt perch) just drive a World and
// put its frames somewhere.
package live

import (
	"fmt"
	"math"
	"math/rand"
	"strings"
	"time"

	"github.com/MagikIO/familiar-says/internal/brain"
	"github.com/MagikIO/familiar-says/internal/canvas"
	"github.com/MagikIO/familiar-says/internal/creature"
	"github.com/MagikIO/familiar-says/internal/ipc"
	"github.com/MagikIO/familiar-says/internal/personality"
	"github.com/MagikIO/familiar-says/internal/stage"
	"github.com/charmbracelet/lipgloss"
)

// Options configures a World.
type Options struct {
	Character   *canvas.Character
	Theme       personality.Theme
	Colors      *canvas.CharacterColors
	Brain       brain.Config
	Memory      *brain.Memory
	BubbleWidth int     // widest a line of speech may be
	Scenery     bool    // draw the ground and the night sky
	Compact     bool    // short stage: bubbles go beside the familiar, not above
	Silent      bool    // never speak unprompted: no greeting, chatter or reminders
	Restless    float64 // how much it wanders; 1 is normal

	// LoadCharacter resolves a character by name, for changing shape while
	// running. Without it the familiar cannot become anything else.
	LoadCharacter func(name string) (*canvas.Character, error)
	NapAfter      time.Duration // quiet time before it dozes off
	Seed          int64
	Now           func() time.Time
}

// World is the simulation.
type World struct {
	W, H int

	C     *creature.Creature
	Brain *brain.Brain

	opt     Options
	rng     *rand.Rand
	parts   stage.Particles
	scenery *stage.Scenery

	bubble     *stage.Bubble
	bubbleSide int  // 0: above the familiar, 1: to its right, -1: to its left
	sticky     bool // the bubble on screen outlives the next command
	queue      []brain.Reaction
	muted      bool

	second  float64
	minute  float64
	tasks   []brain.Task
	suggest string
}

// NewWorld builds a world around a character.
func NewWorld(opt Options) *World {
	if opt.Now == nil {
		opt.Now = time.Now
	}
	if opt.BubbleWidth <= 0 {
		opt.BubbleWidth = 40
	}
	if opt.Restless == 0 {
		opt.Restless = 1
	}
	rng := rand.New(rand.NewSource(opt.Seed))
	w := &World{opt: opt, rng: rng, scenery: stage.NewScenery(opt.Seed)}
	w.C = creature.New(opt.Character, opt.Theme, opt.Colors, rng)
	w.C.Restless = opt.Restless
	if opt.NapAfter > 0 {
		w.C.SetNapAfter(opt.NapAfter.Seconds())
	}

	bc := opt.Brain
	if bc.Name == "" || bc.Name == brain.DefaultConfig().Name {
		bc.Name = "a " + opt.Character.Name
	}
	traits := w.C.Traits()
	if bc.Sounds == nil {
		bc.Sounds = traits.Sounds
	}
	if bc.Tic == "" {
		bc.Tic = traits.Tic
	}
	w.Brain = brain.New(bc, opt.Memory, rng, opt.Now)
	w.tasks = append(w.tasks, func() any { return brain.ScanPath() })

	w.Resize(80, w.MinHeight())
	lo, hi := w.C.Extent()
	w.C.X = float64(w.W-(hi-lo))/2 - float64(lo)
	w.C.SetBounds(w.bounds())
	return w
}

// MinHeight is the shortest stage the familiar fits on.
func (w *World) MinHeight() int {
	_, h := w.C.Size()
	return h + 1
}

// tallest is how many rows a compact stage may grow to for a long remark.
const tallest = 12

// WantHeight is how tall a compact stage should be right now: just enough
// for the familiar, or taller while it is saying something long. A prompt
// perch follows this, so a long answer is shown whole rather than a couple
// of lines at a time.
func (w *World) WantHeight() int {
	h := w.MinHeight()
	if w.opt.Compact && w.bubble != nil {
		_, rows := w.bubble.Size()
		h = max(h, min(rows, tallest))
	}
	return h
}

// Resize changes the stage dimensions.
func (w *World) Resize(width, height int) {
	if width < 10 {
		width = 10
	}
	if height < 2 {
		height = 2
	}
	w.W, w.H = width, height
	w.C.SetBounds(w.bounds())
}

func (w *World) hasGround() bool {
	_, sh := w.C.Size()
	return w.opt.Scenery && !w.opt.Compact && w.H >= sh+3
}

// floorY is the row the familiar's feet stand on.
func (w *World) floorY() int {
	if w.hasGround() {
		return w.H - 2
	}
	return w.H - 1
}

func (w *World) bounds() (minX, maxX float64, headroom int) {
	_, sh := w.C.Size()
	lo, hi := w.C.Extent()
	headroom = w.floorY() - sh + 1
	if headroom < 0 {
		headroom = 0
	}
	if !w.opt.Compact && headroom > 5 {
		headroom = 5 // leave the sky to the speech bubble
	}
	return float64(-lo), float64(w.W - 1 - hi), headroom
}

// Tasks returns slow jobs that should be run off the main loop, with each
// result passed back to Deliver.
func (w *World) Tasks() []brain.Task {
	t := append(w.tasks, w.Brain.Tasks()...)
	w.tasks = nil
	return t
}

// Deliver hands a finished task's result to the world.
func (w *World) Deliver(v any) {
	if v == nil {
		return
	}
	if _, ok := v.(brain.Answer); ok {
		// The wait is over: stop pondering, and don't announce that we are
		// about to think when the answer is already in hand.
		w.C.Settle()
		w.dismiss()
		keep := w.queue[:0]
		for _, r := range w.queue {
			if r.Act != creature.ActThink {
				keep = append(keep, r)
			}
		}
		w.queue = keep
	}
	w.perform(w.Brain.Learn(v))
}

// Suggestion is the command line the familiar most recently proposed, or ""
// once the moment has passed.
func (w *World) Suggestion() string { return w.suggest }

// Handle feeds an external event to the familiar.
func (w *World) Handle(ev ipc.Event) {
	if (ev.Type == ipc.TypeCommand || ev.Type == ipc.TypePreexec) && brain.IsSelf(ev.Cmd) {
		w.C.Poke()
		w.Brain.At(ev.Cwd) // not work to comment on, but it does say where the user is
		return
	}
	if ev.Type == ipc.TypeCommand {
		// A new command makes commentary on the previous one stale.
		w.suggest = ""
		if w.bubble != nil && !w.sticky {
			w.bubble = nil
		}
		keep := w.queue[:0]
		for _, r := range w.queue {
			if r.Sticky {
				keep = append(keep, r)
			}
		}
		w.queue = keep
	}
	switch ev.Type {
	case ipc.TypeBecome:
		w.become(ev.Text)
		return
	case ipc.TypeRescan:
		// Trust is granted from another process, through the shared memory.
		_ = w.Brain.Memory().Save()
		w.Brain.Rescan()
		return
	case ipc.TypeNap:
		w.C.Nap()
		return
	case ipc.TypeWake:
		w.C.Poke()
		return
	case ipc.TypePet:
		w.C.Pet()
	case ipc.TypeSay, ipc.TypeAsk, ipc.TypeWear, ipc.TypeBond:
		// The user is talking to it: whatever it was musing about can wait.
		w.C.Poke()
		if !w.sticky {
			w.dismiss()
		}
	default:
		if w.C.Poke() {
			w.perform(w.Brain.Woke())
		}
	}
	w.perform(w.Brain.OnEvent(ev))
}

// Become swaps the familiar's body for another character's, keeping its
// place on stage, its mood for the day and everything it remembers.
func (w *World) Become(char *canvas.Character) {
	w.Replace(char)
	x := w.C.X
	traits := w.C.Traits()
	w.C.React(creature.ActCheer)
	w.parts.Emit(stage.Sparkle, x+float64(w.C.AnchorX()), float64(w.spriteTop(0)), w.rng)
	if len(traits.Sounds) > 0 {
		w.perform([]brain.Reaction{{Text: traits.Sounds[w.rng.Intn(len(traits.Sounds))], Must: true, Sticky: true}})
	}
}

// Replace swaps the body without any ceremony.
func (w *World) Replace(char *canvas.Character) {
	x, right := w.C.X, w.C.FacingRight()
	w.opt.Character = char
	w.C = creature.New(char, w.opt.Theme, w.opt.Colors, w.rng)
	w.C.Restless = w.opt.Restless
	if w.opt.NapAfter > 0 {
		w.C.SetNapAfter(w.opt.NapAfter.Seconds())
	}
	w.C.X = x
	w.C.Face(right)
	w.parts.Clear()
	w.bubble, w.queue = nil, nil
	traits := w.C.Traits()
	w.Brain.SetVoice("a "+char.Name, traits.Sounds, traits.Tic)
	w.Resize(w.W, max(w.H, w.MinHeight()))
}

func (w *World) become(name string) {
	if w.opt.LoadCharacter == nil || name == "" || name == w.opt.Character.Name {
		return
	}
	char, err := w.opt.LoadCharacter(name)
	if err != nil {
		w.perform([]brain.Reaction{{Text: "I don't know how to be a " + name + ".", Mood: personality.MoodSad, Must: true, Sticky: true}})
		return
	}
	w.Become(char)
}

// Pet, Chat, Explain, Ask and ToggleNap are the things a user can do to the
// familiar directly.
func (w *World) Pet()           { w.Handle(ipc.Event{Type: ipc.TypePet}) }
func (w *World) Chat()          { w.C.Poke(); w.dismiss(); w.perform(w.Brain.Chat()) }
func (w *World) Explain()       { w.C.Poke(); w.dismiss(); w.perform(w.Brain.Explain()) }
func (w *World) Ask(q string)   { w.Handle(ipc.Event{Type: ipc.TypeAsk, Text: q}) }
func (w *World) Say(t string)   { w.Handle(ipc.Event{Type: ipc.TypeSay, Text: t}) }
func (w *World) Muted() bool    { return w.muted }
func (w *World) ToggleMute()    { w.muted = !w.muted }
func (w *World) Speaking() bool { return w.bubble != nil }

// ToggleNap puts the familiar to sleep, or wakes it.
func (w *World) ToggleNap() {
	if w.C.Asleep() {
		w.C.Poke()
	} else {
		w.dismiss()
		w.C.Nap()
	}
}

// SkipBubble finishes typing the current bubble, or dismisses it if it is
// already fully shown.
func (w *World) SkipBubble() {
	if w.bubble == nil {
		return
	}
	if w.bubble.Typing() {
		w.bubble.Skip()
	} else {
		w.bubble.Dismiss()
	}
}

func (w *World) dismiss() {
	if w.bubble != nil {
		w.bubble.Dismiss()
	}
}

// perform carries out reactions: body language now, words in turn.
func (w *World) perform(rs []brain.Reaction) {
	for _, r := range rs {
		if r.Act != creature.ActNone {
			w.C.React(r.Act)
		}
		if r.Mood != "" {
			w.C.Feel(r.Mood, 4)
		}
		if r.Suggest != "" {
			w.suggest = r.Suggest
		}
		if r.Notify != "" {
			title, body := "Your "+w.opt.Character.Name+" says", r.Notify
			w.tasks = append(w.tasks, func() any { brain.Notify(title, body); return nil })
		}
		if r.Text == "" || (w.muted && !r.Must) {
			continue
		}
		if len(w.queue) >= 4 {
			if !r.Must {
				continue
			}
			w.queue = w.queue[1:]
		}
		w.queue = append(w.queue, r)
	}
}

// Step advances the world by dt.
func (w *World) Step(dt time.Duration) {
	sec := dt.Seconds()

	w.second += sec
	w.minute += sec
	if w.minute >= 60 {
		// Other familiars share this memory; trade notes with them.
		w.minute = 0
		_ = w.Brain.Memory().Save()
	}
	if w.second >= 1 {
		w.second = 0
		if !w.opt.Silent {
			w.perform(w.Brain.OnTick(!w.C.Asleep()))
		}
	}

	if w.bubble != nil {
		w.bubble.Step(dt)
		// Don't let one remark hog the stage while others are waiting.
		// (The next page of the same remark is not "others": a page gets
		// its full reading time before the following one replaces it.)
		waiting := len(w.queue) > 0 && !w.queue[0].Continues
		if w.bubble.Done() || (waiting && !w.bubble.Typing() && w.bubble.Held() > 1500*time.Millisecond) {
			w.bubble = nil
		}
	}
	if w.bubble == nil && len(w.queue) > 0 {
		r := w.queue[0]
		w.queue = w.queue[1:]
		w.speak(r)
	}

	if w.opt.Compact {
		if h := w.WantHeight(); h != w.H {
			w.Resize(w.W, h)
		}
	}

	w.C.Talking(w.bubble != nil && w.bubble.Typing(), w.bubble != nil)
	w.C.Step(sec)

	cx, lift := w.C.Offset()
	top := w.spriteTop(lift)
	for _, p := range w.C.Puffs() {
		w.parts.Emit(p.Kind, float64(cx)+p.X, float64(top)+p.Y, w.rng)
	}
	w.parts.Step(sec)
	w.scenery.Step(sec)
}

func (w *World) spriteTop(lift int) int {
	_, sh := w.C.Size()
	return w.floorY() - sh + 1 - lift
}

// speak opens a bubble for r, choosing where it goes and how wide it can be.
// Text too long for the space is split, and the rest queued to follow.
func (w *World) speak(r brain.Reaction) {
	_, sh := w.C.Size()
	lo, hi := w.C.Extent()
	cx, _ := w.C.Offset()
	style := r.Style
	if style == "" {
		style = "say"
	}

	width, maxLines := w.opt.BubbleWidth, 1
	skyRows := w.floorY() - sh + 1
	if !w.opt.Compact && skyRows >= 4 {
		w.bubbleSide = 0
		width = min(width, w.W-6)
		maxLines = skyRows - 3 // two border rows and the tail
	} else {
		right := w.W - (cx + hi + 1) - 2
		left := cx + lo - 2
		w.bubbleSide = 1
		room := right
		if left > right {
			w.bubbleSide, room = -1, left
		}
		width = min(width, room-4)
		maxLines = w.H - 2
		if w.opt.Compact {
			// The stage will grow to fit (see WantHeight), and a long
			// remark may use the full width on that side.
			maxLines = tallest - 2
			if len(r.Text) > 3*width {
				width = min(room-4, 76)
			}
		}
	}
	width = max(width, 10)
	maxLines = max(maxLines, 1)

	wrapped := canvas.WrapText(r.Text, width)
	if len(wrapped) > maxLines {
		rest := r
		rest.Text = strings.Join(wrapped[maxLines:], " ")
		rest.Act, rest.Must, rest.Continues = creature.ActNone, true, true
		w.queue = append([]brain.Reaction{rest}, w.queue...)
		r.Text = strings.Join(wrapped[:maxLines], " ")
		r.Hold = 0
	}

	w.sticky = r.Sticky
	border := paintOf(w.opt.Theme.BubbleStyle)
	ink := stage.Paint{FG: border.FG, Bold: true}
	w.bubble = stage.NewBubble(r.Text, stage.BubbleOptions{Style: style, Width: width, CPS: 42, Hold: r.Hold}, border, ink)
}

// Active reports whether anything on stage is moving, so callers can idle at
// a lower frame rate when nothing is.
func (w *World) Active() bool {
	return w.C.Busy() || len(w.queue) > 0 || (w.bubble != nil && w.bubble.Typing()) ||
		(w.parts.Len() > 0 && !w.C.Asleep())
}

// Status is a one-line description of the familiar's state.
func (w *World) Status() string {
	s := fmt.Sprintf("%s · %s · %s · %s", w.opt.Character.Name, w.C.Mood(), w.C.State(), w.Brain.Memory().Bond().Title)
	if w.muted {
		s += " · muted"
	}
	return s
}

// Render draws the current frame.
func (w *World) Render() *stage.Grid {
	g := stage.NewGrid(w.W, w.H)
	if w.hasGround() {
		w.scenery.Draw(g, w.floorY()+1, w.opt.Now().Hour())
	}

	sprite := w.C.Sprite()
	cx, lift := w.C.Offset()
	top := w.spriteTop(lift)
	g.Blit(sprite, cx, top)
	w.parts.Draw(g)

	if w.bubble != nil {
		w.drawBubble(g, cx, lift)
	}
	if a, ok := w.Brain.Memory().Worn(); ok && top > 0 {
		width := len([]rune(a.Glyph))
		g.Text(cx+w.C.AnchorX()-width/2, top-1, a.Glyph, stage.Paint{FG: a.Color, Bold: true})
	}
	return g
}

func (w *World) drawBubble(g *stage.Grid, cx, lift int) {
	bg := w.bubble.Grid()
	lo, hi := w.C.Extent()
	if !w.C.Flies() {
		lift = 0 // a hop shouldn't drag the bubble up and down with it
	}
	top := w.spriteTop(lift)
	tail := w.bubble.Template.Connector
	tailPaint := w.bubble.Border

	if w.bubbleSide == 0 {
		anchor := clampInt(cx+w.C.AnchorX(), 1, w.W-2)
		bx := clampInt(anchor-4, 0, max(0, w.W-bg.W))
		by := max(0, top-1-bg.H)
		g.Blit(bg, bx, by)
		if ty := by + bg.H; ty < top {
			g.Text(clampInt(anchor, bx+1, bx+bg.W-2), ty, tail, tailPaint)
		}
		return
	}

	if tail == "\\" || tail == "/" || tail == "" {
		tail = "-"
	}
	var bx, tx int
	if w.bubbleSide > 0 {
		tx = cx + hi + 2
		bx = tx + 1
	} else {
		tx = cx + lo - 2
		bx = tx - bg.W
	}
	bx = clampInt(bx, 0, max(0, w.W-bg.W))
	by := clampInt(top+1-bg.H/2, 0, max(0, w.H-bg.H))
	g.Blit(bg, bx, by)
	g.Text(tx, clampInt(top+1, by+1, by+bg.H-2), tail, tailPaint)
}

func clampInt(v, lo, hi int) int {
	if hi < lo {
		hi = lo
	}
	return int(math.Max(float64(lo), math.Min(float64(hi), float64(v))))
}

func paintOf(s lipgloss.Style) stage.Paint {
	p := stage.Paint{Bold: s.GetBold(), Dim: s.GetFaint()}
	if col, ok := s.GetForeground().(lipgloss.Color); ok {
		p.FG = string(col)
	}
	return p
}
