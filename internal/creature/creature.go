// Package creature turns a static character definition into something that
// lives: it walks, fidgets, blinks, naps, reacts to what happens around it and
// wears its mood on its face.
//
// A Creature knows nothing about terminals, sockets or shells. The world tells
// it how much room it has and what just happened (Poke, React, Pet), calls
// Step once per frame, and asks for the current Sprite.
package creature

import (
	"math"
	"math/rand"
	"sort"
	"strings"

	"github.com/MagikIO/familiar-says/internal/canvas"
	"github.com/MagikIO/familiar-says/internal/personality"
	"github.com/MagikIO/familiar-says/internal/stage"
	"github.com/charmbracelet/lipgloss"
)

// State is what the creature is doing with its body.
type State int

const (
	Idle State = iota
	Walking
	Fidgeting
	Sleeping
	Acting
)

func (s State) String() string {
	return [...]string{"idle", "wandering", "fidgeting", "asleep", "reacting"}[s]
}

// Act is a short choreographed reaction.
type Act int

const (
	ActNone      Act = iota
	ActCelebrate     // bounce around in a shower of confetti
	ActCheer         // one happy hop and a few sparkles
	ActWince         // shudder
	ActNervous       // sweat
	ActLove          // hearts
	ActAlert         // "!" and a start
	ActWave          // greeting
	ActNod           // acknowledgement
	ActThink         // pondering; lasts until Settle is called
	ActSing          // notes
	ActFume          // angry steam
)

// Puff is a request to spawn a particle effect relative to the sprite's
// top-left corner. The world owns the particle pool and drains these.
type Puff struct {
	Kind stage.Kind
	X, Y float64
}

const (
	gravity     = 34.0 // rows/s²
	hopImpulse  = 9.0  // rows/s
	jumpImpulse = 12.0
)

// Creature is the simulated familiar.
type Creature struct {
	Char  *canvas.Character
	Theme personality.Theme

	// Position. X is the sprite's left edge in stage columns; Lift is how
	// many rows it is off the ground.
	X    float64
	Lift float64

	// Restless scales how often it wanders off (0 = statue, 1 = default).
	Restless float64

	state     State
	faceRight bool
	vLift     float64
	targetX   float64
	hover     float64 // desired lift for fliers
	timer     float64 // time left in the current state
	shake     int

	act      Act
	actClock float64
	actBeat  float64 // countdown to the act's next periodic effect

	joy     float64 // -1 (miserable) .. 1 (delighted)
	energy  float64 // 0 (exhausted) .. 1 (fresh)
	drowsy  float64 // seconds since the world last poked us
	napAt   float64 // drowsy threshold for falling asleep
	feeling personality.Mood
	feelFor float64

	blinkIn  float64
	blinkFor float64
	talking  bool
	mouthBit float64

	anim      *canvas.AnimationSequence
	animName  string
	animFrame int
	animClock float64
	animLoop  bool

	minX, maxX   float64
	headroom     float64
	maxW, maxH   int
	inkLo, inkHi int

	traits canvas.LiveTraits
	styles canvas.CharacterStyles
	rng    *rand.Rand
	puffs  []Puff
	cache  map[spriteKey]*stage.Grid
}

type spriteKey struct {
	anim        string
	frame       int
	eyes, mouth string
	flip        bool
}

// New brings a character to life.
func New(char *canvas.Character, theme personality.Theme, colors *canvas.CharacterColors, rng *rand.Rand) *Creature {
	c := &Creature{
		Char:     char,
		Theme:    theme,
		Restless: 1,
		energy:   1,
		napAt:    300,
		rng:      rng,
		cache:    map[spriteKey]*stage.Grid{},
		styles:   canvas.ResolveCharacterStyles(canvas.MergeColors(char.Colors, colors), theme.CharacterStyle),
	}
	if char.Live != nil {
		c.traits = *char.Live
	}
	if c.traits.Speed <= 0 {
		c.traits.Speed = 7
	}
	if c.traits.Fidgets == nil {
		c.traits.Fidgets = defaultFidgets(char, c.traits.Gait)
	}
	c.faceRight = c.traits.Facing == "right"

	c.maxW, c.maxH = char.Width(), char.Height()
	for _, a := range char.Animations {
		for _, f := range a.Frames {
			if len(f.Art) > c.maxH {
				c.maxH = len(f.Art)
			}
			for _, l := range f.Art {
				if w := canvas.StringWidth(l); w > c.maxW {
					c.maxW = w
				}
			}
		}
	}
	c.inkLo, c.inkHi = inkExtent(char, c.maxW)
	if c.traits.Facing == "left" || c.traits.Facing == "right" {
		// The art can be mirrored, so its footprint is too.
		lo, hi := c.inkLo, c.inkHi
		c.inkLo = min(lo, c.maxW-1-hi)
		c.inkHi = max(hi, c.maxW-1-lo)
	}
	c.blinkIn = 2 + rng.Float64()*3
	c.timer = 1.5
	if c.Flies() {
		c.hover = 1
	}
	return c
}

// defaultFidgets picks idle animations for characters that do not list any.
func defaultFidgets(char *canvas.Character, gait string) []string {
	var out []string
	for name := range char.Animations {
		switch name {
		case "idle", "blink", "wave", "nod", "hop", "jump", gait:
		default:
			out = append(out, name)
		}
	}
	sort.Strings(out)
	return out
}

// inkExtent finds the first and last columns that hold anything other than
// blank padding in any frame of the character's art.
func inkExtent(char *canvas.Character, width int) (lo, hi int) {
	lo, hi = width, -1
	scan := func(art []string) {
		for _, l := range art {
			t := strings.TrimRight(l, " ")
			if t == "" {
				continue
			}
			if lead := len(t) - len(strings.TrimLeft(t, " ")); lead < lo {
				lo = lead
			}
			if w := canvas.StringWidth(t) - 1; w > hi {
				hi = w
			}
		}
	}
	scan(char.Art)
	for _, a := range char.Animations {
		for _, f := range a.Frames {
			scan(f.Art)
		}
	}
	if hi < lo {
		return 0, max(0, width-1)
	}
	return lo, hi
}

// Extent returns the first and last sprite columns that can hold ink. Art is
// often drawn with blank padding around it; layout should go by this, not by
// the raw sprite width.
func (c *Creature) Extent() (lo, hi int) { return c.inkLo, c.inkHi }

// Size returns the largest footprint the sprite can have.
func (c *Creature) Size() (w, h int) { return c.maxW, c.maxH }

// Traits returns the character's live-mode traits with defaults filled in.
func (c *Creature) Traits() canvas.LiveTraits { return c.traits }

// State reports what the creature is doing.
func (c *Creature) State() State { return c.state }

// Asleep reports whether it is napping.
func (c *Creature) Asleep() bool { return c.state == Sleeping }

// Flies reports whether this character hovers rather than walks.
func (c *Creature) Flies() bool { return c.traits.Locomotion == "fly" }

// FacingRight reports which way it is currently looking.
func (c *Creature) FacingRight() bool { return c.faceRight }

// Play runs one of the character's animations, as a fidget. It reports
// whether the character has an animation by that name.
func (c *Creature) Play(name string) bool {
	seq := c.Char.GetAnimation(name)
	if seq == nil || len(seq.Frames) == 0 {
		return false
	}
	if c.state == Sleeping {
		c.state = Idle
	}
	c.act = ActNone
	c.play(name, seq.Loop)
	c.state = Fidgeting
	c.timer = 4
	return true
}

// Wander sends it off for a walk.
func (c *Creature) Wander() {
	if c.state == Sleeping {
		c.state = Idle
	}
	c.wander()
}

// Face turns it to look right or left.
func (c *Creature) Face(right bool) { c.faceRight = right }

// AnchorX is the sprite column a speech bubble's tail should point at.
func (c *Creature) AnchorX() int {
	x := c.Char.Anchor.X
	if x < 0 {
		x = 0
	}
	if x >= c.maxW {
		x = c.maxW - 1
	}
	if c.flipped() {
		x = c.maxW - 1 - x
	}
	return x
}

// SetBounds tells the creature how much room it has: the range of columns its
// left edge may occupy and how many rows of air there are above its head.
func (c *Creature) SetBounds(minX, maxX float64, headroom int) {
	if maxX < minX {
		maxX = minX
	}
	c.minX, c.maxX = minX, maxX
	c.headroom = float64(headroom)
	c.X = clamp(c.X, minX, maxX)
	c.targetX = clamp(c.targetX, minX, maxX)
}

// SetNapAfter sets how many quiet seconds pass before it dozes off.
func (c *Creature) SetNapAfter(seconds float64) {
	if seconds > 0 {
		c.napAt = seconds
	}
}

// Busy reports whether anything is visibly in motion, which lets the caller
// drop to a slower frame rate when it is not.
func (c *Creature) Busy() bool {
	return c.state == Walking || c.state == Acting || c.Lift > 0.05 || c.anim != nil || c.talking || c.blinkFor > 0
}

// Puffs returns and clears the particle effects requested since the last call.
func (c *Creature) Puffs() []Puff {
	p := c.puffs
	c.puffs = nil
	return p
}

// ---- stimuli ---------------------------------------------------------------

// Poke tells the creature that something happened in the world. It resets the
// nap timer and wakes it if it was asleep. It reports whether it woke up.
func (c *Creature) Poke() bool {
	c.drowsy = 0
	if c.state != Sleeping {
		return false
	}
	c.state = Idle
	c.timer = 1.5
	c.energy = math.Max(c.energy, 0.6)
	c.React(ActAlert)
	return true
}

// Nap sends it to sleep right now.
func (c *Creature) Nap() {
	c.stopAnim()
	c.state = Sleeping
	c.act = ActNone
	c.actBeat = 0.4
	c.feelFor = 0
}

// Pet is affection from the user.
func (c *Creature) Pet() {
	c.Poke()
	c.joy = clamp(c.joy+0.25, -1, 1)
	c.React(ActLove)
}

// Cheer nudges its underlying happiness up (positive) or down (negative).
func (c *Creature) Cheer(delta float64) { c.joy = clamp(c.joy+delta, -1, 1) }

// Feel overrides the displayed mood for a few seconds.
func (c *Creature) Feel(m personality.Mood, seconds float64) {
	if m == "" {
		return
	}
	c.feeling, c.feelFor = m, seconds
}

// Talking tells the creature whether it is mid-sentence, which animates its
// mouth and keeps it from wandering off while a bubble is up.
func (c *Creature) Talking(typing, bubbleUp bool) {
	c.talking = typing
	if bubbleUp && c.state == Walking {
		c.state = Idle
		c.stopAnim()
	}
	if bubbleUp && c.state == Idle && c.timer < 0.5 {
		c.timer = 0.5
	}
}

// Settle ends an open-ended act such as ActThink.
func (c *Creature) Settle() {
	if c.state == Acting {
		c.endAct()
	}
}

// React starts a choreographed reaction, interrupting whatever it was doing.
func (c *Creature) React(a Act) {
	if a == ActNone {
		return
	}
	if c.state == Sleeping {
		c.state = Idle
	}
	c.stopAnim()
	c.state = Acting
	c.act = a
	c.actClock = 0
	c.actBeat = 0
	c.shake = 0
	head := float64(c.AnchorX())

	switch a {
	case ActCelebrate:
		c.timer = 2.6
		c.Feel(personality.MoodExcited, 5)
		c.joy = clamp(c.joy+0.3, -1, 1)
		c.puff(stage.Confetti, head, 0)
		c.jump(jumpImpulse)
	case ActCheer:
		c.timer = 0.9
		c.Feel(personality.MoodHappy, 4)
		c.joy = clamp(c.joy+0.12, -1, 1)
		c.puff(stage.Sparkle, head, -1)
		c.jump(hopImpulse)
	case ActWince:
		c.timer = 0.7
		c.Feel(personality.MoodSad, 4)
		c.joy = clamp(c.joy-0.12, -1, 1)
		c.puff(stage.Sweat, head+2, 0)
	case ActNervous:
		c.timer = 2.4
		c.Feel(personality.MoodSurprised, 3.5)
	case ActLove:
		c.timer = 1.4
		c.Feel(personality.MoodHappy, 4)
	case ActAlert:
		c.timer = 0.7
		c.Feel(personality.MoodSurprised, 1.6)
		c.puff(stage.Alert, head, -1)
		c.jump(hopImpulse * 0.7)
	case ActWave:
		c.timer = 0.2
		c.Feel(personality.MoodHappy, 3)
		if !c.play("wave", false) {
			c.timer = 0.8
			c.jump(hopImpulse)
		}
	case ActNod:
		c.timer = 0.2
		if !c.play("nod", false) {
			c.timer = 0.4
		}
	case ActThink:
		c.timer = 90 // until Settle
	case ActSing:
		c.timer = 2.4
		c.Feel(personality.MoodHappy, 4)
	case ActFume:
		c.timer = 1.6
		c.Feel(personality.MoodAngry, 3.5)
	}
}

// ---- simulation ------------------------------------------------------------

// Step advances the creature by dt seconds.
func (c *Creature) Step(dt float64) {
	c.drowsy += dt
	if c.feelFor > 0 {
		c.feelFor -= dt
	}
	c.joy -= c.joy * 0.01 * dt // feelings fade back toward neutral
	if c.state == Sleeping {
		c.energy = math.Min(1, c.energy+dt/240)
	} else {
		c.energy = math.Max(0, c.energy-dt/5400)
	}

	c.stepBlink(dt)
	c.stepAnim(dt)
	c.stepLift(dt)
	if c.talking {
		c.mouthBit += dt
	}

	switch c.state {
	case Idle:
		c.stepIdle(dt)
	case Walking:
		c.stepWalk(dt)
	case Fidgeting:
		c.timer -= dt
		if c.timer <= 0 || (c.anim == nil && !c.animLoop) {
			c.rest()
		}
	case Sleeping:
		c.actBeat -= dt
		if c.actBeat <= 0 {
			c.actBeat = 1.5
			c.puff(stage.Zzz, float64(c.AnchorX())+1, -1)
		}
	case Acting:
		c.stepAct(dt)
	}
}

func (c *Creature) stepIdle(dt float64) {
	if c.drowsy > c.napAt || c.energy < 0.05 {
		c.Nap()
		return
	}
	c.timer -= dt
	if c.timer > 0 || c.talking {
		return
	}
	roll := c.rng.Float64()
	room := c.maxX - c.minX
	switch {
	case roll < 0.5*c.Restless && room >= 4:
		c.wander()
	case roll < 0.5*c.Restless+0.3 && len(c.traits.Fidgets) > 0:
		c.fidget()
	default:
		c.rest()
	}
}

func (c *Creature) rest() {
	c.stopAnim()
	c.state = Idle
	c.timer = 2 + c.rng.Float64()*5
	if c.Restless > 0 {
		c.timer /= math.Max(0.4, c.Restless)
	}
}

func (c *Creature) wander() {
	span := c.maxX - c.minX
	for i := 0; i < 4; i++ {
		c.targetX = c.minX + c.rng.Float64()*span
		if math.Abs(c.targetX-c.X) >= math.Min(6, span/2) {
			break
		}
	}
	c.state = Walking
	c.faceRight = c.targetX > c.X
	if c.traits.Gait != "" {
		c.play(c.traits.Gait, true)
	}
	if c.Flies() {
		c.hover = 1 + c.rng.Float64()*math.Max(0, c.headroom-1)
	}
}

func (c *Creature) fidget() {
	name := c.traits.Fidgets[c.rng.Intn(len(c.traits.Fidgets))]
	seq := c.Char.GetAnimation(name)
	if seq == nil || len(seq.Frames) == 0 {
		c.rest()
		return
	}
	c.play(name, seq.Loop)
	c.state = Fidgeting
	c.timer = 2 + c.rng.Float64()*2.5
}

func (c *Creature) stepWalk(dt float64) {
	dir := 1.0
	if c.targetX < c.X {
		dir = -1
	}
	c.faceRight = dir > 0
	step := c.traits.Speed * dt
	if math.Abs(c.targetX-c.X) <= step {
		c.X = c.targetX
		c.rest()
		c.timer = 0.8 + c.rng.Float64()*2.5
		return
	}
	c.X += dir * step

	grounded := c.Lift <= 0 && c.vLift == 0
	if c.traits.Locomotion == "hop" && grounded {
		c.jump(hopImpulse * 0.75)
		c.puff(stage.Dust, c.rearX(), float64(c.maxH-1))
	}
}

func (c *Creature) stepAct(dt float64) {
	c.actClock += dt
	c.actBeat -= dt
	c.timer -= dt
	head := float64(c.AnchorX())
	grounded := c.Lift <= 0 && c.vLift == 0

	switch c.act {
	case ActCelebrate:
		if grounded && c.timer > 0.5 {
			c.jump(jumpImpulse * (0.7 + c.rng.Float64()*0.3))
			c.faceRight = !c.faceRight
		}
		if c.actBeat <= 0 {
			c.actBeat = 0.5
			c.puff(stage.Sparkle, head, -1)
		}
	case ActWince:
		c.shake = int(c.actClock/0.06)%2*2 - 1
	case ActNervous:
		c.shake = 0
		if int(c.actClock/0.09)%4 == 0 {
			c.shake = 1
		}
		if c.actBeat <= 0 {
			c.actBeat = 0.4
			c.puff(stage.Sweat, head+2, 0)
		}
	case ActLove:
		if c.actBeat <= 0 {
			c.actBeat = 0.22
			c.puff(stage.Hearts, head, -1)
		}
	case ActThink:
		if c.actBeat <= 0 {
			c.actBeat = 0.9
			c.puff(stage.Thought, head+1, -1)
		}
	case ActSing:
		if c.actBeat <= 0 {
			c.actBeat = 0.35
			c.puff(stage.Notes, head+1, -1)
		}
		if grounded && int(c.actClock/0.6)%2 == 0 {
			c.jump(hopImpulse * 0.5)
		}
	case ActFume:
		c.shake = int(c.actClock/0.08)%2*2 - 1
		if c.actBeat <= 0 {
			c.actBeat = 0.25
			c.puff(stage.Fume, head, -1)
		}
	case ActWave, ActNod:
		if c.anim != nil {
			c.timer = 0.2 // hold the act open until the animation finishes
		}
	}

	if c.timer <= 0 && (grounded || c.Flies()) {
		c.endAct()
	}
}

func (c *Creature) endAct() {
	c.act = ActNone
	c.shake = 0
	c.rest()
	c.timer = 1 + c.rng.Float64()*2
}

func (c *Creature) jump(v float64) {
	if c.Lift <= 0.01 || c.Flies() {
		c.vLift = v
	}
}

func (c *Creature) stepLift(dt float64) {
	if c.Flies() && c.state != Sleeping {
		// Fliers ease toward their hover height with a gentle bob.
		want := c.hover
		if c.state == Idle || c.state == Fidgeting {
			want = math.Max(1, math.Min(c.hover, 1+c.headroom/3))
		}
		want = math.Min(want, c.headroom)
		c.Lift += (want - c.Lift) * math.Min(1, dt*2.5)
		if c.vLift != 0 {
			c.Lift += c.vLift * dt
			c.vLift -= gravity * dt
			if c.vLift < 0 {
				c.vLift = 0
			}
		}
		c.Lift = clamp(c.Lift, 0, c.headroom)
		return
	}
	if c.Lift <= 0 && c.vLift <= 0 {
		c.Lift, c.vLift = 0, 0
		return
	}
	c.Lift += c.vLift * dt
	c.vLift -= gravity * dt
	if c.Lift <= 0 {
		c.Lift, c.vLift = 0, 0
	}
	if c.Lift > c.headroom {
		c.Lift = c.headroom
		if c.vLift > 0 {
			c.vLift = 0
		}
	}
}

func (c *Creature) stepBlink(dt float64) {
	if c.blinkFor > 0 {
		c.blinkFor -= dt
		return
	}
	c.blinkIn -= dt
	if c.blinkIn <= 0 {
		c.blinkFor = 0.14
		c.blinkIn = 2.5 + c.rng.Float64()*4.5
		if c.rng.Float64() < 0.15 {
			c.blinkIn = 0.3 // the occasional double blink
		}
	}
}

// ---- animation playback ----------------------------------------------------

func (c *Creature) play(name string, loop bool) bool {
	seq := c.Char.GetAnimation(name)
	if seq == nil || len(seq.Frames) == 0 {
		return false
	}
	c.anim, c.animName, c.animFrame, c.animClock, c.animLoop = seq, name, 0, 0, loop
	return true
}

func (c *Creature) stopAnim() {
	c.anim, c.animName, c.animFrame, c.animClock, c.animLoop = nil, "", 0, 0, false
}

func (c *Creature) stepAnim(dt float64) {
	if c.anim == nil {
		return
	}
	c.animClock += dt * 1000
	for {
		d := float64(c.anim.Frames[c.animFrame].DurationMs)
		if d < 30 {
			d = 30
		}
		if c.animClock < d {
			return
		}
		c.animClock -= d
		c.animFrame++
		if c.animFrame >= len(c.anim.Frames) {
			if !c.animLoop {
				c.stopAnim()
				return
			}
			c.animFrame = 0
		}
	}
}

// ---- appearance ------------------------------------------------------------

// Mood is the feeling currently shown on its face.
func (c *Creature) Mood() personality.Mood {
	switch {
	case c.state == Sleeping:
		return personality.MoodSleepy
	case c.feelFor > 0:
		return c.feeling
	case c.joy > 0.45:
		return personality.MoodHappy
	case c.joy < -0.45:
		return personality.MoodSad
	case c.energy < 0.2 || c.drowsy > c.napAt*0.8:
		return personality.MoodSleepy
	case c.drowsy > c.napAt*0.5:
		return personality.MoodBored
	}
	return personality.MoodNeutral
}

func (c *Creature) flipped() bool {
	switch c.traits.Facing {
	case "left":
		return c.faceRight
	case "right":
		return !c.faceRight
	}
	return false
}

// rearX is the sprite column at the creature's trailing edge.
func (c *Creature) rearX() float64 {
	if c.faceRight {
		return 0
	}
	return float64(c.maxW - 1)
}

func (c *Creature) puff(k stage.Kind, x, y float64) {
	c.puffs = append(c.puffs, Puff{Kind: k, X: x, Y: y})
}

// Offset returns the sprite's draw offset: whole columns from the left edge of
// the stage (including any shake) and whole rows of lift.
func (c *Creature) Offset() (x, lift int) {
	return int(math.Round(c.X)) + c.shake, int(math.Round(c.Lift))
}

func isPlaceholder(s string, r rune) bool {
	return s != "" && strings.Trim(s, string(r)) == ""
}

// fit stretches a two-glyph expression such as "^^" across a wider slot.
func fit(expr string, width int) string {
	rs := []rune(expr)
	if len(rs) == 2 && width >= 3 && canvas.StringWidth(expr) == 2 {
		return string(rs[0]) + strings.Repeat(" ", width-2) + string(rs[1])
	}
	if width == 1 {
		return canvas.OneGlyph(expr)
	}
	return expr
}

// face works out which eyes and mouth to draw right now.
func (c *Creature) face() (eyes, mouth string, closed bool) {
	closed = c.state == Sleeping || c.blinkFor > 0

	mood := c.Mood()
	own := c.Char.Expressions[string(mood)]
	if s := c.Char.Eyes; s != nil {
		eyes = fit(c.Theme.GetExpression(mood).Eyes, s.Width)
		if own.Eyes != "" {
			eyes = own.Eyes // the character's own face outranks the theme's
		}
		if closed {
			eyes = strings.Repeat("-", max(1, s.Width))
			if shut := c.Char.Expressions["closed"].Eyes; shut != "" {
				eyes = shut
			}
		}
	}
	if s := c.Char.Mouth; s != nil {
		// Slots marked with the stock "Y" placeholder are blank at rest;
		// anything else is part of the drawing and stays put.
		mouth = s.Placeholder
		if isPlaceholder(s.Placeholder, 'Y') {
			mouth = strings.Repeat(" ", max(1, s.Width))
		}
		if own.Mouth != "" {
			mouth = own.Mouth
		}
		if c.talking && int(c.mouthBit/0.13)%2 == 0 {
			mouth = fit("o", s.Width)
			if s.Width >= 2 {
				mouth = strings.Repeat("o", s.Width)
			}
		}
	}
	return eyes, mouth, closed
}

// Sprite renders the creature as it looks this instant.
func (c *Creature) Sprite() *stage.Grid {
	eyes, mouth, closed := c.face()

	name, frame := c.animName, c.animFrame
	var fr *canvas.AnimationFrame
	if c.anim != nil {
		fr = &c.anim.Frames[c.animFrame]
	} else if closed && c.Char.Eyes == nil {
		// No eye slot to rewrite: borrow the shut-eyed frame from the
		// character's own blink animation.
		if seq, i := c.shutFrame(); seq != nil {
			fr, name, frame = &seq.Frames[i], "blink", i
		}
	}

	art := c.Char.Art
	if fr != nil {
		if len(fr.Art) > 0 {
			art = fr.Art
		}
		// A frame that restates the placeholder means "whatever the mood is".
		if fr.Eyes != "" && !isPlaceholder(fr.Eyes, '@') && !c.restates(c.Char.Eyes, fr.Eyes) && !closed {
			eyes = fr.Eyes
		}
		if fr.Mouth != "" && !isPlaceholder(fr.Mouth, 'Y') && !c.talking {
			mouth = fr.Mouth
		}
	}

	key := spriteKey{name, frame, eyes, mouth, c.flipped()}
	if g, ok := c.cache[key]; ok {
		return g
	}

	ch := c.Char
	if fr != nil && len(fr.Art) > 0 {
		clone := *c.Char
		clone.Art = art
		ch = &clone
	}
	cv := ch.ToCanvasStyled(eyes, mouth, c.styles)
	g := stage.FromLines(cv.RenderPlain(), c.maxW, func(x, y int) stage.Paint {
		return paintOf(cv.Get(x, y).Style)
	})
	if key.flip {
		g = g.Flip()
	}
	if len(c.cache) > 512 {
		c.cache = map[spriteKey]*stage.Grid{}
	}
	c.cache[key] = g
	return g
}

// restates reports whether a frame's value is just the slot's placeholder.
func (c *Creature) restates(slot *canvas.Slot, value string) bool {
	return slot != nil && slot.Placeholder != "" && value == slot.Placeholder
}

// shutFrame finds the eyes-closed frame of the character's blink animation:
// by convention it is the briefest one.
func (c *Creature) shutFrame() (*canvas.AnimationSequence, int) {
	seq := c.Char.GetAnimation("blink")
	if seq == nil || len(seq.Frames) < 2 {
		return nil, 0
	}
	best := -1
	for i, f := range seq.Frames {
		if len(f.Art) == 0 {
			continue
		}
		if best < 0 || f.DurationMs < seq.Frames[best].DurationMs {
			best = i
		}
	}
	if best < 0 {
		return nil, 0
	}
	return seq, best
}

// paintOf converts a lipgloss style into a stage paint.
func paintOf(s lipgloss.Style) stage.Paint {
	p := stage.Paint{Bold: s.GetBold(), Dim: s.GetFaint()}
	if col, ok := s.GetForeground().(lipgloss.Color); ok {
		p.FG = string(col)
	}
	return p
}

func clamp(v, lo, hi float64) float64 {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}
