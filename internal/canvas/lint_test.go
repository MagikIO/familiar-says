package canvas

import (
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
)

func TestBuiltinCharactersAreSound(t *testing.T) {
	for _, name := range ListBuiltinCharacters() {
		ch, ok := GetBuiltinCharacter(name)
		if !ok {
			t.Fatalf("cannot load %s", name)
		}
		for _, problem := range Lint(ch) {
			t.Errorf("%s: %s", name, problem)
		}
	}
}

func TestLintExplainsCommonMistakes(t *testing.T) {
	ch := &Character{
		Name:   "blob",
		Art:    []string{" (@@) ", " /  \\ "},
		Anchor: Anchor{X: 40},
		Eyes:   &Slot{Line: 0, Col: 3, Width: 3, Placeholder: "@@"},
		Mouth:  &Slot{Line: 5, Placeholder: "Y", Width: 1},
		Animations: map[string]*AnimationSequence{
			"wave":  {Frames: []AnimationFrame{{DurationMs: 0, Art: []string{"one line"}}}},
			"empty": {},
		},
		Expressions: map[string]Face{"grumpy": {Eyes: "><"}, "happy": {Eyes: "^_^"}},
		Live:        &LiveTraits{Facing: "sideways", Locomotion: "swim", Gait: "trot", Fidgets: []string{"wave", "scratch"}},
	}
	got := strings.Join(Lint(ch), "\n")
	for _, want := range []string{
		`placeholder "@@" is at column 2, but col says 3`,
		`width is 3 but the placeholder "@@" is 2 wide`,
		"mouth: line 5 is outside the art",
		"anchor: x is 40",
		`animation "wave", frame 1: needs a duration`,
		`animation "wave", frame 1: has 1 lines of art, the character has 2`,
		`animation "empty" has no frames`,
		`"grumpy" is not a mood`,
		`happy eyes "^_^" are 3 wide, the eye slot is 3`,
		`live.facing: "sideways"`,
		`live.locomotion: "swim"`,
		`live.gait: there is no animation called "trot"`,
		`live.fidgets: there is no animation called "scratch"`,
	} {
		if want == `happy eyes "^_^" are 3 wide, the eye slot is 3` {
			if strings.Contains(got, want) {
				t.Errorf("eyes that fit the slot should not be reported")
			}
			continue
		}
		if !strings.Contains(got, want) {
			t.Errorf("missing %q in:\n%s", want, got)
		}
	}
}

func TestCharacterFacesOutrankTheTheme(t *testing.T) {
	cat, _ := GetBuiltinCharacter("cat")
	if eyes, _ := cat.Expression("happy", "^^", "  "); eyes != "^Y^" {
		t.Errorf("the cat has its own happy eyes, got %q", eyes)
	}
	if _, mouth := cat.Expression("happy", "^^", "  "); mouth != "^" {
		t.Errorf("a blank theme mouth must not erase the cat's nose, got %q", mouth)
	}
	cow, _ := GetBuiltinCharacter("default")
	if eyes, mouth := cow.Expression("happy", "^^", "  "); eyes != "^^" || mouth != "  " {
		t.Errorf("a character without faces of its own uses the theme's: %q %q", eyes, mouth)
	}

	for expr, want := range map[string]string{"^^": "^", "[o_o]": "o", ">:(": ":", "-": "-", "": ""} {
		if got := OneGlyph(expr); got != want {
			t.Errorf("OneGlyph(%q) = %q, want %q", expr, got, want)
		}
	}
	suse, _ := GetBuiltinCharacter("suse")
	lines := suse.ToCanvas("^^", "__", lipgloss.NewStyle()).RenderPlain()
	if !strings.Contains(lines[1], "/^") || !strings.Contains(lines[3], "@@") {
		t.Errorf("suse's eye should take the mood and its tail should stay a tail:\n%s", strings.Join(lines, "\n"))
	}
}
