package stage

import "math/rand"

// Scenery is the backdrop behind the familiar: a strip of ground and, at
// night, a few stars that twinkle.
type Scenery struct {
	seed  int64
	clock float64
}

// NewScenery creates scenery whose layout is fixed by seed.
func NewScenery(seed int64) *Scenery { return &Scenery{seed: seed} }

// Step advances the twinkle clock.
func (s *Scenery) Step(dt float64) { s.clock += dt }

// Draw paints the backdrop. groundY is the row of the ground strip; hour is
// the local hour of day, which decides whether the stars are out.
func (s *Scenery) Draw(g *Grid, groundY, hour int) {
	night := hour >= 20 || hour < 6
	rng := rand.New(rand.NewSource(s.seed))

	if night && groundY > 3 {
		n := g.W * groundY / 55
		for i := 0; i < n; i++ {
			x, y := rng.Intn(g.W), rng.Intn(groundY-1)
			period := 2.5 + rng.Float64()*5
			offset := rng.Float64() * period
			glyph := []rune{'.', '·', '+', '*'}[rng.Intn(4)]
			t := s.clock + offset
			phase := t - float64(int(t/period))*period
			p := Paint{FG: "#8787AF", Dim: true}
			if phase < 0.35 { // a brief glint once per period
				p = Paint{FG: "#FFFFD7", Bold: true}
				glyph = '✦'
			}
			g.Set(x, y, glyph, p)
		}
	}

	if groundY >= 0 && groundY < g.H {
		soil := Paint{FG: "#5F875F", Dim: true}
		if night {
			soil = Paint{FG: "#5F5F87", Dim: true}
		}
		tufts := []rune{'_', '_', '_', '_', '_', '.', ',', '\'', '"', '_', '_'}
		for x := 0; x < g.W; x++ {
			g.Set(x, groundY, tufts[rng.Intn(len(tufts))], soil)
		}
	}
}
