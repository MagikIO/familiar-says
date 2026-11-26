package canvas

import (
	"fmt"
	"sort"
	"strings"
)

var moods = map[string]bool{"neutral": true, "happy": true, "sad": true, "angry": true, "surprised": true,
	"bored": true, "excited": true, "sleepy": true, "closed": true}

// Lint checks a character definition for the mistakes that are easy to make
// when drawing one by hand, and describes each in plain words. An empty
// result means the character is sound.
func Lint(ch *Character) []string {
	var out []string
	warn := func(format string, args ...any) { out = append(out, fmt.Sprintf(format, args...)) }

	if len(ch.Art) == 0 {
		return []string{"there is no art"}
	}
	width := ch.Width()
	for i, line := range ch.Art {
		for _, r := range line {
			if StringWidth(string(r)) != 1 {
				warn("art line %d contains %q, which is not one cell wide and will skew the drawing", i+1, r)
				break
			}
		}
	}

	slot := func(name string, s *Slot) {
		if s == nil {
			return
		}
		if s.Line < 0 || s.Line >= len(ch.Art) {
			warn("%s: line %d is outside the art (lines are numbered from 0 to %d)", name, s.Line, len(ch.Art)-1)
			return
		}
		at := strings.Index(ch.Art[s.Line], s.Placeholder)
		switch {
		case s.Placeholder == "":
			warn("%s: needs a placeholder, the text in the art that gets replaced", name)
		case at < 0:
			warn("%s: placeholder %q is not on art line %d", name, s.Placeholder, s.Line)
		case at != s.Col:
			warn("%s: placeholder %q is at column %d, but col says %d", name, s.Placeholder, at, s.Col)
		}
		if w := StringWidth(s.Placeholder); s.Width != w && s.Placeholder != "" {
			warn("%s: width is %d but the placeholder %q is %d wide", name, s.Width, s.Placeholder, w)
		}
	}
	slot("eyes", ch.Eyes)
	slot("mouth", ch.Mouth)

	if ch.Anchor.X < 0 || ch.Anchor.X >= width {
		warn("anchor: x is %d, but the art is only %d wide", ch.Anchor.X, width)
	}

	names := make([]string, 0, len(ch.Animations))
	for name := range ch.Animations {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		anim := ch.Animations[name]
		if anim == nil || len(anim.Frames) == 0 {
			warn("animation %q has no frames", name)
			continue
		}
		for i, f := range anim.Frames {
			if f.DurationMs <= 0 {
				warn("animation %q, frame %d: needs a duration in milliseconds", name, i+1)
			}
			if len(f.Art) > 0 && len(f.Art) != len(ch.Art) {
				warn("animation %q, frame %d: has %d lines of art, the character has %d", name, i+1, len(f.Art), len(ch.Art))
			}
			if len(f.Art) == 0 && f.Eyes == "" && f.Mouth == "" && f.OffsetX == 0 && f.OffsetY == 0 && len(anim.Frames) > 1 && i > 0 && i < len(anim.Frames)-1 {
				warn("animation %q, frame %d: changes nothing", name, i+1)
			}
		}
	}

	for mood, face := range ch.Expressions {
		if !moods[mood] {
			warn("expressions: %q is not a mood (use neutral, happy, sad, angry, surprised, bored, excited, sleepy or closed)", mood)
		}
		if ch.Eyes != nil && face.Eyes != "" && StringWidth(face.Eyes) != ch.Eyes.Width {
			warn("expressions: %s eyes %q are %d wide, the eye slot is %d", mood, face.Eyes, StringWidth(face.Eyes), ch.Eyes.Width)
		}
		if ch.Eyes == nil && face.Eyes != "" {
			warn("expressions: %s has eyes, but the character has no eye slot to put them in", mood)
		}
	}

	if l := ch.Live; l != nil {
		switch l.Facing {
		case "", "left", "right", "front":
		default:
			warn("live.facing: %q should be left, right or front", l.Facing)
		}
		switch l.Locomotion {
		case "", "walk", "hop", "fly":
		default:
			warn("live.locomotion: %q should be walk, hop or fly", l.Locomotion)
		}
		if l.Gait != "" && !ch.HasAnimation(l.Gait) {
			warn("live.gait: there is no animation called %q", l.Gait)
		}
		for _, f := range l.Fidgets {
			if !ch.HasAnimation(f) {
				warn("live.fidgets: there is no animation called %q", f)
			}
		}
	}
	sort.Strings(out)
	return out
}
