package stage

import (
	"math"
	"math/rand"
)

// Particle is a short-lived glyph with simple physics. Positions are in cells,
// velocities in cells per second, and Y grows downward.
type Particle struct {
	X, Y   float64
	VX, VY float64
	AY     float64 // vertical acceleration (gravity is positive)
	Sway   float64 // amplitude of a sideways sine wobble, in cells
	Life   float64 // seconds
	Glyphs []rune  // played start-to-end across the particle's life
	P      Paint

	age, phase float64
}

// Particles is a pool of live particles.
type Particles struct {
	list []Particle
}

// Len reports how many particles are alive.
func (ps *Particles) Len() int { return len(ps.list) }

// Clear removes every particle.
func (ps *Particles) Clear() { ps.list = ps.list[:0] }

// Add spawns a particle.
func (ps *Particles) Add(p Particle) {
	if p.Life <= 0 || len(p.Glyphs) == 0 {
		return
	}
	if len(ps.list) >= 400 { // hard cap; nothing here should ever need more
		return
	}
	ps.list = append(ps.list, p)
}

// Step advances the simulation by dt seconds and drops dead particles.
func (ps *Particles) Step(dt float64) {
	keep := ps.list[:0]
	for _, p := range ps.list {
		p.age += dt
		if p.age >= p.Life {
			continue
		}
		p.VY += p.AY * dt
		p.X += p.VX * dt
		p.Y += p.VY * dt
		keep = append(keep, p)
	}
	ps.list = keep
}

// Draw paints the particles onto g.
func (ps *Particles) Draw(g *Grid) {
	for _, p := range ps.list {
		x := p.X
		if p.Sway != 0 {
			x += math.Sin(p.phase+p.age*4) * p.Sway
		}
		i := int(p.age / p.Life * float64(len(p.Glyphs)))
		if i >= len(p.Glyphs) {
			i = len(p.Glyphs) - 1
		}
		paint := p.P
		if p.age > p.Life*0.7 {
			paint.Dim = true
		}
		g.Set(int(math.Round(x)), int(math.Round(p.Y)), p.Glyphs[i], paint)
	}
}

// Kind names a canned particle effect.
type Kind int

const (
	Hearts Kind = iota
	Zzz
	Confetti
	Sparkle
	Sweat
	Notes
	Alert
	Question
	Dust
	Fume
	Thought
)

var confettiColors = []string{"#FF5F87", "#FFD75F", "#5FD7FF", "#87FF5F", "#D787FF", "#FFAF5F"}

// Emit spawns the canned effect kind at (x, y).
func (ps *Particles) Emit(kind Kind, x, y float64, rng *rand.Rand) {
	jit := func(n float64) float64 { return (rng.Float64()*2 - 1) * n }
	switch kind {
	case Hearts:
		ps.Add(Particle{X: x + jit(2), Y: y, VY: -1.6 - rng.Float64(), Sway: 0.8, Life: 1.6,
			Glyphs: []rune{'♥'}, P: Paint{FG: "#FF5F87", Bold: true}, phase: rng.Float64() * 6})
	case Zzz:
		ps.Add(Particle{X: x, Y: y, VX: 1.2, VY: -1.1, Life: 2.4,
			Glyphs: []rune{'z', 'z', 'Z', 'Z'}, P: Paint{FG: "#87AFFF"}})
	case Confetti:
		for i := 0; i < 26; i++ {
			glyphs := [][]rune{{'*'}, {'+'}, {'o', '.'}, {'✦', '·'}, {'"', '\''}}[rng.Intn(5)]
			ps.Add(Particle{X: x + jit(1.5), Y: y, VX: jit(9), VY: -4 - rng.Float64()*6, AY: 11,
				Life: 1.2 + rng.Float64()*0.9, Glyphs: glyphs,
				P: Paint{FG: confettiColors[rng.Intn(len(confettiColors))], Bold: true}})
		}
	case Sparkle:
		for i := 0; i < 5; i++ {
			ps.Add(Particle{X: x + jit(5), Y: y + jit(2), Life: 0.5 + rng.Float64()*0.6,
				Glyphs: []rune{'·', '✧', '✦', '✧', '·'}, P: Paint{FG: "#FFD75F", Bold: true}})
		}
	case Sweat:
		ps.Add(Particle{X: x + jit(1), Y: y, VX: jit(3), VY: -1.5, AY: 9, Life: 0.8,
			Glyphs: []rune{'\'', ',', '.'}, P: Paint{FG: "#5FD7FF"}})
	case Notes:
		ps.Add(Particle{X: x + jit(2), Y: y, VY: -1.4, Sway: 1, Life: 1.8,
			Glyphs: []rune{[]rune{'♪', '♫'}[rng.Intn(2)]}, P: Paint{FG: "#D787FF", Bold: true}, phase: rng.Float64() * 6})
	case Alert:
		ps.Add(Particle{X: x, Y: y, VY: -0.6, Life: 0.9, Glyphs: []rune{'!'}, P: Paint{FG: "#FFD75F", Bold: true}})
	case Question:
		ps.Add(Particle{X: x, Y: y, VY: -0.5, Life: 1.2, Glyphs: []rune{'?'}, P: Paint{FG: "#5FD7FF", Bold: true}})
	case Dust:
		ps.Add(Particle{X: x, Y: y, VX: jit(1.5), VY: -0.4, Life: 0.45,
			Glyphs: []rune{'.', '·'}, P: Paint{Dim: true}})
	case Fume:
		ps.Add(Particle{X: x + jit(1), Y: y, VY: -1.2, Sway: 0.5, Life: 0.9,
			Glyphs: []rune{'#', '*', '\''}, P: Paint{FG: "#FF5F5F", Bold: true}, phase: rng.Float64() * 6})
	case Thought:
		ps.Add(Particle{X: x, Y: y, VX: 0.6, VY: -0.9, Life: 1.3,
			Glyphs: []rune{'.', 'o', 'O'}, P: Paint{Dim: true}})
	}
}
