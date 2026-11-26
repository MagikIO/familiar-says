package brain

import (
	"fmt"
	"strings"
)

// Accessory is something the familiar can wear on its head.
type Accessory struct {
	Name  string
	Glyph string
	Color string
	Level int // bond level that unlocks it
}

// Accessories lists everything there is to earn, in the order it is earned.
var Accessories = []Accessory{
	{"flower", "✿", "#FF87AF", 1},
	{"bow", "⋈", "#FF5F5F", 2},
	{"tophat", "_▇_", "#B2B2B2", 3},
	{"crown", "♛", "#FFD75F", 4},
	{"halo", "◯", "#FFFFAF", 5},
}

// FindAccessory looks an accessory up by name.
func FindAccessory(name string) (Accessory, bool) {
	for _, a := range Accessories {
		if a.Name == strings.ToLower(strings.TrimSpace(name)) {
			return a, true
		}
	}
	return Accessory{}, false
}

// levels are the stages of the bond: the points needed to reach each, and
// what the familiar considers you to be once there.
var levels = []struct {
	Points int
	Title  string
}{
	{0, "new acquaintance"},
	{60, "companion"},
	{200, "friend"},
	{500, "close friend"},
	{1100, "kindred spirit"},
	{2400, "true familiar"},
}

// Bond describes how close the familiar and its person have become.
type Bond struct {
	Level  int
	Title  string
	Points int
	Floor  int // points at which this level began
	Next   int // points at which the next begins; 0 at the top
	Days   int
}

// Bond measures the relationship. Time together counts most; being present
// for the work, sharing its wins, and affection all add to it. Petting is
// capped per day so the bond cannot simply be farmed.
func (m *Memory) Bond() Bond {
	pets := min(m.Pets, 8*max(m.Days, 1))
	points := m.Days*20 + pets*2 + m.Commands/8 + m.Fixes*4
	b := Bond{Points: points, Days: m.Days}
	for i, l := range levels {
		if points >= l.Points {
			b.Level, b.Title, b.Floor = i, l.Title, l.Points
		}
	}
	if b.Level+1 < len(levels) {
		b.Next = levels[b.Level+1].Points
	}
	return b
}

// Unlocked lists the accessories earned so far.
func (m *Memory) Unlocked() []Accessory {
	level := m.Bond().Level
	var out []Accessory
	for _, a := range Accessories {
		if a.Level <= level {
			out = append(out, a)
		}
	}
	return out
}

// Wear puts an accessory on, or takes it off for "" or "none". It fails for
// accessories that do not exist or have not been earned yet.
func (m *Memory) Wear(name string) error {
	name = strings.ToLower(strings.TrimSpace(name))
	if name == "" || name == "none" || name == "nothing" {
		m.Wearing = ""
		return nil
	}
	a, ok := FindAccessory(name)
	if !ok {
		return fmt.Errorf("there is no %q to wear", name)
	}
	if b := m.Bond(); a.Level > b.Level {
		return fmt.Errorf("the %s is earned at %q; you are at %q", a.Name, levels[a.Level].Title, b.Title)
	}
	m.Wearing = a.Name
	return nil
}

// Worn returns the accessory currently on, if any.
func (m *Memory) Worn() (Accessory, bool) {
	if m.Wearing == "" {
		return Accessory{}, false
	}
	return FindAccessory(m.Wearing)
}

// Summary describes the bond in a sentence.
func (b Bond) Summary() string {
	s := fmt.Sprintf("%s together. I'd call us %s", plural(b.Days, "day"), withArticle(b.Title))
	if b.Next > 0 {
		s += fmt.Sprintf(" (%d/%d to %s).", b.Points-b.Floor, b.Next-b.Floor, levels[b.Level+1].Title)
	} else {
		s += "."
	}
	return s
}

func withArticle(title string) string {
	if strings.HasPrefix(title, "true") || strings.HasPrefix(title, "close") {
		return title + "s"
	}
	return title + "s"
}
