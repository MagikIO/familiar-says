package brain

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/MagikIO/familiar-says/internal/creature"
	"github.com/MagikIO/familiar-says/internal/personality"
)

// Rule is something the user has taught the familiar: when a command matching
// Match happens, say or suggest this. Rules live in the config file under
// live.rules, and per project in a .familiar.json file.
type Rule struct {
	Match           string `json:"match"`                     // regular expression the command line must match
	On              string `json:"on,omitempty"`              // "success", "failure", "before" (as it starts) or "always" (the default: either outcome)
	Exit            *int   `json:"exit,omitempty"`            // only for this exit status
	Output          string `json:"output,omitempty"`          // regular expression the command's output must match (needs `familiar-says wrap`)
	Say             string `json:"say,omitempty"`             // what to say; {cmd}, {exit}, {duration}, {dir} and {1}..{9} are filled in
	Suggest         string `json:"suggest,omitempty"`         // a command to offer; Alt-G puts it on the prompt
	Act             string `json:"act,omitempty"`             // celebrate, cheer, wince, nervous, love, alert, wave, nod, sing or fume
	Mood            string `json:"mood,omitempty"`            // happy, sad, angry, surprised, bored, excited, sleepy
	Style           string `json:"style,omitempty"`           // bubble style: say, think, shout, whisper, song
	Notify          bool   `json:"notify,omitempty"`          // also send a desktop notification
	CooldownMinutes int    `json:"cooldownMinutes,omitempty"` // stay quiet this long after firing

	re, outRe *regexp.Regexp
	last      time.Time
}

var acts = map[string]creature.Act{
	"celebrate": creature.ActCelebrate, "cheer": creature.ActCheer, "wince": creature.ActWince,
	"nervous": creature.ActNervous, "love": creature.ActLove, "alert": creature.ActAlert,
	"wave": creature.ActWave, "nod": creature.ActNod, "sing": creature.ActSing, "fume": creature.ActFume,
}

// CompileRules validates rules and prepares them for matching. Rules with
// problems are left out and reported, so one typo does not silence the rest.
func CompileRules(rules []Rule) (ok []*Rule, problems []error) {
	for i := range rules {
		rule := rules[i]
		bad := func(format string, args ...any) {
			problems = append(problems, fmt.Errorf("rule %d (%q): %s", i+1, rule.Match, fmt.Sprintf(format, args...)))
		}
		if rule.Match == "" {
			bad("needs a \"match\" pattern")
			continue
		}
		if rule.Say == "" && rule.Suggest == "" && rule.Act == "" {
			bad("does nothing: give it \"say\", \"suggest\" or \"act\"")
			continue
		}
		var err error
		if rule.re, err = regexp.Compile(rule.Match); err != nil {
			bad("bad \"match\": %v", err)
			continue
		}
		if rule.Output != "" {
			if rule.outRe, err = regexp.Compile(rule.Output); err != nil {
				bad("bad \"output\": %v", err)
				continue
			}
		}
		switch rule.On {
		case "", "always", "success", "failure", "before":
		default:
			bad("\"on\" must be success, failure, before or always, not %q", rule.On)
			continue
		}
		if _, known := acts[rule.Act]; rule.Act != "" && !known {
			bad("unknown \"act\" %q", rule.Act)
			continue
		}
		ok = append(ok, &rule)
	}
	return ok, problems
}

// applies reports whether the rule fires for this moment.
func (r *Rule) applies(when string, exit int, line, output string, now time.Time) bool {
	switch r.On {
	case "before":
		if when != "before" {
			return false
		}
	case "success", "failure":
		if when != r.On {
			return false
		}
	default:
		if when == "before" {
			return false
		}
	}
	if r.Exit != nil && (when == "before" || *r.Exit != exit) {
		return false
	}
	if r.CooldownMinutes > 0 && !r.last.IsZero() && now.Sub(r.last) < time.Duration(r.CooldownMinutes)*time.Minute {
		return false
	}
	if !r.re.MatchString(line) {
		return false
	}
	return r.outRe == nil || r.outRe.MatchString(output)
}

// reaction builds what the rule asks for.
func (r *Rule) reaction(line string, exit int, took time.Duration, dir string) Reaction {
	groups := r.re.FindStringSubmatch(line)
	fill := func(s string) string {
		pairs := []string{"{cmd}", line, "{exit}", strconv.Itoa(exit), "{duration}", human(took), "{dir}", filepath.Base(dir)}
		for i := 1; i <= 9; i++ {
			g := ""
			if i < len(groups) {
				g = groups[i]
			}
			pairs = append(pairs, "{"+strconv.Itoa(i)+"}", g)
		}
		return strings.NewReplacer(pairs...).Replace(s)
	}
	out := Reaction{
		Text: fill(r.Say), Suggest: fill(r.Suggest), Style: r.Style,
		Act: acts[r.Act], Mood: personality.Mood(r.Mood), Must: true,
	}
	if r.Notify {
		out.Notify = out.Text
	}
	return out
}

// ProjectFile is the name of a project's own instructions for the familiar.
const ProjectFile = ".familiar.json"

// Project is the content of a .familiar.json file.
type Project struct {
	Welcome string `json:"welcome,omitempty"` // said on arriving in the project
	Rules   []Rule `json:"rules,omitempty"`

	Path string `json:"-"`
	Hash string `json:"-"` // of the file's bytes; trust is granted to a hash
}

// FindProject looks for a .familiar.json in dir or the directories above it.
// It returns nil if there is none.
func FindProject(dir string) (*Project, error) {
	home, _ := os.UserHomeDir()
	for i := 0; i < 24 && dir != "" && dir != home; i++ {
		path := filepath.Join(dir, ProjectFile)
		if data, err := os.ReadFile(path); err == nil {
			p := &Project{Path: path}
			sum := sha256.Sum256(data)
			p.Hash = hex.EncodeToString(sum[:])
			if len(data) > 256*1024 {
				return p, fmt.Errorf("%s is implausibly large", path)
			}
			if err := json.Unmarshal(data, p); err != nil {
				return p, fmt.Errorf("%s: %w", path, err)
			}
			return p, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return nil, nil
}
