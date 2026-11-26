// Package brain decides what the familiar has to say about what it sees.
//
// It is a rule engine, not a model: it watches the stream of shell events,
// keeps a little state (streaks, tallies, where you are) and turns that into
// reactions — a correction for a typo, a note that the long build finished, a
// nervous look at `rm -rf`, the occasional tip. Suggestions that carry real
// information are always delivered; banter is rate-limited and scaled by the
// configured talkativeness.
package brain

import (
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/MagikIO/familiar-says/internal/creature"
	"github.com/MagikIO/familiar-says/internal/ipc"
	"github.com/MagikIO/familiar-says/internal/personality"
)

// Reaction is something for the familiar to do, say, or both.
type Reaction struct {
	Text  string           // what to say; empty for a wordless reaction
	Style string           // bubble template; empty means "say"
	Mood  personality.Mood // expression to wear while it happens
	Act   creature.Act     // body language
	Must  bool             // carries real information: never rate-limited away
	Hold  time.Duration    // override how long the bubble lingers

	// Sticky remarks were asked for (a `tell`, an oracle's answer) and stay
	// up when the next command runs; everything else is commentary on the
	// last command and is cleared away once there is a newer one.
	Sticky bool

	// Continues marks the next page of a remark too long for one bubble.
	Continues bool

	// Notify, if set, is also sent as a desktop notification.
	Notify string

	// Suggest is a ready-to-run command line the remark is proposing. Shell
	// hooks bind a key that drops it onto the prompt.
	Suggest string
}

// Task is slow work (running git, asking an oracle) the brain wants done off
// the render loop. Its result is handed back through Brain.Learn.
type Task func() any

// Config tunes the brain.
type Config struct {
	Name          string        // the familiar's name, for the oracle's persona
	Sounds        []string      // noises it makes
	Tic           string        // verbal tic
	Talkativeness float64       // 0 = only essentials, 1 = chatterbox
	LongCommand   time.Duration // commands at least this long get announced
	BreakEvery    time.Duration // nudge to take a break after this much work; 0 disables
	Tips          bool          // hand out shell tips when idle
	ShareContext  bool          // include cwd and the last failed command in oracle questions
	Oracle        Oracle
	Rules         []Rule // the user's own rules, from the config file
	Notify        string // desktop notifications for long commands: "unfocused" (default), "always" or "never"
	GitHub        bool   // watch pull requests and CI with the gh CLI
}

// DefaultConfig returns sensible defaults.
func DefaultConfig() Config {
	return Config{
		Name:          "a familiar",
		Talkativeness: 0.5,
		LongCommand:   20 * time.Second,
		BreakEvery:    50 * time.Minute,
		Tips:          true,
		ShareContext:  true,
		Notify:        "unfocused",
	}
}

// Brain is the familiar's mind. It is not safe for concurrent use; drive it
// from a single loop.
type Brain struct {
	cfg Config
	mem *Memory
	rng *rand.Rand
	now func() time.Time

	onPath PathIndex
	shell  string

	greeted     bool
	lastSpoke   time.Time
	lastEvent   time.Time
	workSince   time.Time
	nextChatter time.Time
	lateDay     int

	cwd      string
	root     string
	fails    map[string]int
	okStreak int
	lastFail *ipc.Event
	lastOut  string // tail of the last failure's output, when the shell is wrapped
	warned   bool   // the command in flight got a nervous look
	used     map[string]int

	here      *Insight // what is known about the directory the user is in
	gitRoot   string   // root of the git repository the user is in, if any
	focusFile string   // where a wrapped shell reports whether its window has focus
	gh        map[string]*ghMemory
	ghBusy    bool
	ghNext    time.Time
	ghWatch   time.Time // until when CI is being watched closely

	rules    []*Rule // from the config file
	problems []error // what was wrong with any of them
	project  []*Rule // from the current project's .familiar.json, once trusted
	asked    map[string]bool

	tasks []Task
}

// New creates a brain. rng and now are injectable for tests.
func New(cfg Config, mem *Memory, rng *rand.Rand, now func() time.Time) *Brain {
	if mem == nil {
		mem = LoadMemory("")
	}
	if now == nil {
		now = time.Now
	}
	b := &Brain{cfg: cfg, mem: mem, rng: rng, now: now, fails: map[string]int{}, used: map[string]int{}, asked: map[string]bool{}, gh: map[string]*ghMemory{}}
	b.rules, b.problems = CompileRules(cfg.Rules)
	return b
}

// Memory exposes what the brain remembers, for saving.
func (b *Brain) Memory() *Memory { return b.mem }

// Tasks returns and clears the slow work requested since the last call.
func (b *Brain) Tasks() []Task {
	t := b.tasks
	b.tasks = nil
	return t
}

// LastFailure returns the most recent failed command, if any.
func (b *Brain) LastFailure() *ipc.Event { return b.lastFail }

// ---- phrasing --------------------------------------------------------------

// pick chooses a line for an occasion, avoiding an immediate repeat.
func (b *Brain) pick(key string) string {
	opts := lines[key]
	if len(opts) == 0 {
		return ""
	}
	i := b.rng.Intn(len(opts))
	if last, ok := b.used[key]; ok && len(opts) > 1 && i == last {
		i = (i + 1) % len(opts)
	}
	b.used[key] = i
	return opts[i]
}

func (b *Brain) pickf(key string, args ...any) string {
	return fmt.Sprintf(b.pick(key), args...)
}

// chance rolls for optional banter, scaled so that the default talkativeness
// of 0.5 leaves p unchanged.
func (b *Brain) chance(p float64) bool {
	return b.rng.Float64() < p*b.cfg.Talkativeness*2
}

// banter builds an optional remark and sometimes adds the character's tic.
func (b *Brain) banter(text string, act creature.Act, mood personality.Mood) Reaction {
	if b.cfg.Tic != "" && text != "" && b.rng.Float64() < 0.2 {
		text += " " + b.cfg.Tic
	}
	return Reaction{Text: text, Act: act, Mood: mood}
}

// deliver applies rate limiting: optional remarks made too soon after the last
// one keep their body language but lose their words.
func (b *Brain) deliver(rs ...Reaction) []Reaction {
	now := b.now()
	out := rs[:0]
	for _, r := range rs {
		if r.Suggest != "" && b.shell != "" && b.mem.Hinted < 3 {
			// The shell hooks bind Alt-G; say so until it has sunk in.
			b.mem.Hinted++
			r.Text += " (Alt-G accepts.)"
		}
		if r.Text != "" && !r.Must && now.Sub(b.lastSpoke) < 25*time.Second {
			r.Text = ""
		}
		if r.Text == "" && r.Act == creature.ActNone && r.Mood == "" && r.Notify == "" {
			continue
		}
		if r.Text != "" {
			b.lastSpoke = now
			b.scheduleChatter()
		}
		out = append(out, r)
	}
	return out
}

func (b *Brain) scheduleChatter() {
	if b.cfg.Talkativeness <= 0 {
		b.nextChatter = time.Time{}
		return
	}
	gap := (4 + b.rng.Float64()*8) / (b.cfg.Talkativeness * 2)
	b.nextChatter = b.now().Add(time.Duration(gap * float64(time.Minute)))
}

// ---- events ----------------------------------------------------------------

// OnEvent reacts to something that happened in a shell.
func (b *Brain) OnEvent(ev ipc.Event) []Reaction {
	now := b.now()
	if ev.Shell != "" {
		b.shell = ev.Shell
	}
	var rs []Reaction

	switch ev.Type {
	case ipc.TypePreexec:
		rs = b.onPreexec(ev)
	case ipc.TypeCommand:
		b.mem.Today(now)
		if !b.lastEvent.IsZero() && now.Sub(b.lastEvent) > 45*time.Minute {
			// The closer you are, the gladder it is to see you.
			hello := Reaction{Text: b.pick("back"), Act: creature.ActWave, Must: true}
			if b.mem.Bond().Level >= 2 {
				hello = Reaction{Text: b.pick("back_fond"), Act: creature.ActLove, Must: true}
			}
			rs = append(rs, hello)
			b.workSince = now
		}
		if b.workSince.IsZero() || now.Sub(b.lastEvent) > 12*time.Minute {
			b.workSince = now
		}
		rs = append(rs, b.onCommand(ev)...)
		b.lastEvent = now
	case ipc.TypeSay:
		if strings.TrimSpace(ev.Text) != "" {
			rs = append(rs, Reaction{Text: ev.Text, Style: ev.Style, Mood: personality.Mood(ev.Mood), Must: true, Sticky: true})
		}
	case ipc.TypeAsk:
		b.At(ev.Cwd) // the question is about wherever it was asked from
		rs = b.Ask(ev.Text)
	case ipc.TypePet:
		b.mem.Pets++
		rs = append(rs, b.banter(b.pick("pet"), creature.ActLove, personality.MoodHappy))
	case ipc.TypeWear:
		if err := b.mem.Wear(ev.Text); err != nil {
			rs = append(rs, Reaction{Text: "I can't: " + err.Error() + ".", Mood: personality.MoodSad, Must: true, Sticky: true})
		} else if b.mem.Wearing == "" {
			rs = append(rs, Reaction{Text: b.pick("undress"), Act: creature.ActNod, Must: true, Sticky: true})
		} else {
			rs = append(rs, Reaction{Text: b.pick("dress"), Act: creature.ActCheer, Must: true, Sticky: true})
		}
	case ipc.TypeBond:
		rs = append(rs, Reaction{Text: b.mem.Bond().Summary(), Act: creature.ActLove, Must: true, Sticky: true})
	}
	return b.deliver(rs...)
}

func (b *Brain) onPreexec(ev ipc.Event) []Reaction {
	p := parse(ev.Cmd)
	b.warned = false
	if r, ok := b.taught("before", ev, ""); ok {
		return []Reaction{r}
	}
	advice, risky := danger(p)
	if !risky {
		return nil
	}
	b.warned = true
	if advice != "" {
		return []Reaction{{Text: advice, Act: creature.ActNervous, Must: true}}
	}
	r := Reaction{Act: creature.ActNervous}
	if b.chance(0.6) {
		r.Text = b.pick("danger")
	}
	return []Reaction{r}
}

func (b *Brain) onCommand(ev ipc.Event) []Reaction {
	p := parse(ev.Cmd)
	if p.Prog == "" {
		return nil
	}
	b.mem.Commands++

	// Notice arrivals: a new directory is worth a look around.
	arrived := b.At(ev.Cwd)

	// In a wrapped shell the command's output is on its way to a file. Wait
	// for it (off the main loop) so the verdict can be based on what the
	// command actually said.
	if ev.Transcript != "" {
		b.focusFile = ev.Transcript + ".focus"
		b.tasks = append(b.tasks, func() any { return ReadTranscript(ev) })
		return nil
	}
	return b.judge(ev, "", false, arrived)
}

// judge decides what, if anything, to say about a finished command. output
// is what the command printed, if that is known.
func (b *Brain) judge(ev ipc.Event, output string, unfocused, arrived bool) []Reaction {
	rs := b.verdict(ev, output, arrived)
	if r, ok := b.taught(outcome(ev.Exit), ev, output); ok {
		rs = []Reaction{r} // what the user taught it outranks what it worked out itself
	}

	// Somebody who has wandered off wants to know the long thing is done.
	took := time.Duration(ev.DurationMs) * time.Millisecond
	long := b.cfg.LongCommand > 0 && took >= b.cfg.LongCommand
	if long && (b.cfg.Notify == "always" || (b.cfg.Notify == "unfocused" && unfocused)) {
		note := fmt.Sprintf("Finished after %s: %s", human(took), clip(ev.Cmd, 60))
		if ev.Exit != 0 {
			note = fmt.Sprintf("Failed (exit %d) after %s: %s", ev.Exit, human(took), clip(ev.Cmd, 60))
		}
		if len(rs) == 0 {
			rs = append(rs, Reaction{})
		}
		rs[0].Notify = note
	}
	return rs
}

func outcome(exit int) string {
	if exit != 0 {
		return "failure"
	}
	return "success"
}

// taught looks for a rule the user wrote that covers this moment. Project
// rules are consulted before the user's general ones.
func (b *Brain) taught(when string, ev ipc.Event, output string) (Reaction, bool) {
	line := strings.TrimSpace(ev.Cmd)
	now := b.now()
	for _, set := range [][]*Rule{b.project, b.rules} {
		for _, rule := range set {
			if rule.applies(when, ev.Exit, line, output, now) {
				rule.last = now
				return rule.reaction(line, ev.Exit, time.Duration(ev.DurationMs)*time.Millisecond, ev.Cwd), true
			}
		}
	}
	return Reaction{}, false
}

// verdict is the familiar's own opinion of a finished command.
func (b *Brain) verdict(ev ipc.Event, output string, arrived bool) []Reaction {
	p := parse(ev.Cmd)
	took := time.Duration(ev.DurationMs) * time.Millisecond
	long := b.cfg.LongCommand > 0 && took >= b.cfg.LongCommand
	fam := p.family()
	warned := b.warned
	b.warned = false

	if ev.Exit != 0 {
		if p.benignFailure(ev.Exit) {
			return nil
		}
		return b.onFailure(ev, p, fam, took, long, output)
	}

	// ---- success ----
	b.okStreak++
	streak := b.fails[fam]
	delete(b.fails, fam)
	if streak > 0 {
		b.mem.Fixes++
	}
	if p.Prog == "git" && p.Sub == "push" {
		// A push usually starts CI: keep a closer eye on it for a while.
		now := b.now()
		b.ghWatch, b.ghNext = now.Add(20*time.Minute), now.Add(25*time.Second)
	}

	if p.Prog == "git" && !arrived && ev.Cwd != "" {
		switch p.Sub {
		case "commit", "merge", "rebase", "pull", "cherry-pick", "revert":
			dir := ev.Cwd
			b.tasks = append(b.tasks, func() any { return Scout(dir, "git") })
		}
	}

	switch {
	case streak >= 3:
		return []Reaction{{Text: b.pickf("recover_big", streak+1), Act: creature.ActCelebrate, Must: true}}
	case streak >= 1 && (p.isTest() || p.isBuild()):
		return []Reaction{{Text: b.pick("recover"), Act: creature.ActCelebrate, Must: true}}
	case streak >= 1:
		return []Reaction{b.banter(b.pick("recover"), creature.ActCheer, personality.MoodHappy)}
	case long:
		return []Reaction{{Text: b.pickf("long_done", human(took)), Act: creature.ActCheer, Must: true}}
	case p.isTest():
		r := Reaction{Act: creature.ActCheer}
		if b.chance(0.6) {
			r = b.banter(b.pick("test_pass"), creature.ActCheer, personality.MoodHappy)
		}
		return []Reaction{r}
	case p.Prog == "git" && p.Sub == "push":
		now := b.now()
		if now.Weekday() == time.Friday && now.Hour() >= 15 && b.chance(0.8) {
			return []Reaction{b.banter(b.pick("friday"), creature.ActNervous, "")}
		}
		r := Reaction{Act: creature.ActCheer}
		if b.chance(0.5) {
			r = b.banter(b.pick("push"), creature.ActCheer, personality.MoodHappy)
		}
		return []Reaction{r}
	case warned:
		r := Reaction{Mood: personality.MoodHappy}
		if b.chance(0.4) {
			r.Text = b.pick("relief")
		}
		return []Reaction{r}
	}

	if tip, ok := b.shorthandFor(p.Line); ok {
		return []Reaction{tip}
	}
	switch b.okStreak {
	case 15, 40, 100:
		return []Reaction{b.banter(b.pickf("streak", b.okStreak), creature.ActCheer, personality.MoodHappy)}
	}
	if p.Prog == "git" && p.Sub == "commit" && b.chance(0.25) {
		return []Reaction{b.banter(b.pick("commit"), creature.ActNod, personality.MoodHappy)}
	}
	if p.Sudo && b.chance(0.08) {
		return []Reaction{b.banter(b.pick("sudo"), creature.ActNone, "")}
	}
	if b.chance(0.1) {
		return []Reaction{{Act: creature.ActNod}}
	}
	return nil
}

func (b *Brain) onFailure(ev ipc.Event, p parsed, fam string, took time.Duration, long bool, output string) []Reaction {
	b.mem.Failures++
	b.okStreak = 0
	b.fails[fam]++
	streak := b.fails[fam]
	failed := ev
	b.lastFail = &failed
	b.lastOut = tail(output, 40)

	if ev.Exit == 130 { // Ctrl-C is a decision, not a failure
		delete(b.fails, fam)
		b.lastFail = nil
		if b.chance(0.15) {
			return []Reaction{b.banter(b.pick("interrupt"), creature.ActNone, "")}
		}
		return nil
	}

	wince := creature.ActWince
	if fix, ok := correction(p, ev.Exit, b.onPath); ok {
		return []Reaction{{Text: fmt.Sprintf("Did you mean `%s`?", clip(fix, 60)), Act: creature.ActAlert, Must: true, Suggest: fix}}
	}
	if ev.Exit == 127 {
		return []Reaction{{Text: fmt.Sprintf("`%s` isn't on your PATH. Not installed, or a different name?", clip(p.Prog, 24)), Act: wince, Must: true}}
	}
	if fix, ok := pathCorrection(p, ev.Cwd); ok {
		return []Reaction{{Text: fmt.Sprintf("No `%s` here. Did you mean `%s`?", clip(p.Sub, 24), clip(fix, 32)), Act: creature.ActAlert, Must: true, Suggest: p.withPath(fix)}}
	}
	if ev.Exit == 126 && len(p.Words) > 0 && strings.ContainsRune(p.Words[0], '/') {
		return []Reaction{{Text: fmt.Sprintf("That file isn't executable. `chmod +x %s`?", clip(p.Words[0], 30)), Act: wince, Must: true, Suggest: "chmod +x " + p.Words[0]}}
	}
	// With the command's own words to go on, say what they mean.
	if d, ok := Diagnose(output, p); ok {
		return []Reaction{{Text: d.Text, Suggest: d.Suggest, Act: wince, Must: true}}
	}
	if streak >= 3 && streak%3 == 0 {
		return []Reaction{{Text: b.pickf("fail_many", streak, p.Prog), Act: wince, Mood: personality.MoodSad, Must: true}}
	}
	if why, ok := exitMeaning(p, ev.Exit); ok && streak == 1 {
		return []Reaction{{Text: why, Act: wince, Must: true}}
	}
	if long {
		return []Reaction{{Text: b.pickf("long_fail", human(took)), Act: wince, Mood: personality.MoodSad, Must: true}}
	}

	r := Reaction{Act: wince}
	switch {
	case p.isTest() && b.chance(0.5):
		r = b.banter(b.pick("test_fail"), wince, personality.MoodSad)
	case p.isBuild() && b.chance(0.5):
		r = b.banter(b.pick("build_fail"), wince, personality.MoodSad)
	case b.chance(0.25):
		text := b.pick("fail")
		if strings.Contains(text, "%d") {
			text = fmt.Sprintf(text, ev.Exit)
		}
		r = b.banter(text, wince, personality.MoodSad)
	}
	return []Reaction{r}
}

// shorthandFor tallies a command line and, the sixth time it is typed in
// full, offers an abbreviation for it.
func (b *Brain) shorthandFor(line string) (Reaction, bool) {
	if len(line) > 80 {
		return Reaction{}, false
	}
	b.mem.Counts[line]++
	if b.mem.Counts[line] < 6 || b.mem.Suggested[line] {
		return Reaction{}, false
	}
	tip, ok := shorthand(line, b.shell, b.onPath)
	b.mem.Suggested[line] = true // ask once, whether or not a name was free
	if !ok {
		return Reaction{}, false
	}
	return Reaction{
		Text: fmt.Sprintf("You've typed `%s` %d times. Try: %s", line, b.mem.Counts[line], tip),
		Act:  creature.ActAlert, Must: true, Suggest: tip,
	}, true
}

// ---- things learned asynchronously ------------------------------------------

// Learn takes the result of a Task.
func (b *Brain) Learn(v any) []Reaction {
	switch v := v.(type) {
	case PathIndex:
		b.onPath = v
	case Insight:
		return b.deliver(b.onInsight(v)...)
	case GitHubStatus:
		return b.deliver(b.onGitHub(v)...)
	case Transcript:
		return b.deliver(b.judge(v.Event, v.Output, v.Unfocused, false)...)
	case Answer:
		if v.Err != nil {
			return b.deliver(Reaction{Text: "The oracle is silent: " + v.Err.Error(), Mood: personality.MoodSad, Must: true, Sticky: true})
		}
		return b.deliver(Reaction{Text: v.Text, Act: creature.ActNod, Must: true, Sticky: true})
	}
	return nil
}

func (b *Brain) onInsight(in Insight) []Reaction {
	if in.Trigger != "rescan" {
		seen := in
		b.here = &seen
		b.gitRoot = ""
		if in.Git != nil {
			b.gitRoot = in.Git.Root
		}
	}
	if in.Trigger != "git" {
		if rs := b.adopt(in.Project, in.ProjectErr); len(rs) > 0 || in.Trigger == "rescan" {
			return rs
		}
	}
	if in.Trigger == "git" {
		g := in.Git
		if g == nil || g.Ahead == 0 {
			return nil
		}
		if g.Ahead == 5 || g.Ahead == 10 || g.Ahead == 20 {
			return []Reaction{{Text: fmt.Sprintf("That's %d unpushed commits on `%s`. A push would back them up.", g.Ahead, g.Branch), Must: true}}
		}
		return nil
	}

	root := in.Root()
	if root == b.root {
		return nil
	}
	b.root = root
	now := b.now()
	if last, ok := b.mem.Introduced[root]; ok && now.Sub(time.Unix(last, 0)) < 8*time.Hour {
		return nil
	}

	var parts []string
	if in.Kind != "" && len(in.Handy) > 0 {
		handy := make([]string, len(in.Handy))
		for i, h := range in.Handy {
			handy[i] = "`" + h + "`"
		}
		parts = append(parts, fmt.Sprintf("%s. Handy here: %s.", in.Kind, strings.Join(handy, ", ")))
	}
	if s := in.Git.Summary(); s != "" {
		parts = append(parts, s)
	}
	if in.Git != nil && in.Git.Behind > 0 && in.Git.Changed == 0 {
		parts = append(parts, "`git pull` would catch you up.")
	}
	if len(parts) == 0 {
		return nil
	}
	b.mem.Introduced[root] = now.Unix()
	return []Reaction{{Text: filepath.Base(root) + ": " + strings.Join(parts, " "), Act: creature.ActAlert, Must: true}}
}

// At tells the brain which directory the user is in. It reports whether that
// is news, in which case it also takes a look around the place.
//
// Commands report their directory when they finish, but that is not the only
// way to learn it: a shell can be opened straight into a project, and the
// first thing typed there may be a question for the familiar.
func (b *Brain) At(dir string) bool {
	if dir == "" || dir == b.cwd {
		return false
	}
	b.cwd = dir
	b.tasks = append(b.tasks, func() any { return Scout(dir, "cd") })
	return true
}

// facts describes a directory in the terms an oracle will find useful.
func (in Insight) facts() []string {
	var out []string
	if in.Kind != "" {
		out = append(out, fmt.Sprintf("project: %s (%s)", filepath.Base(in.Root()), in.Kind))
	}
	if len(in.Handy) > 0 {
		out = append(out, "useful commands here: "+strings.Join(in.Handy, ", "))
	}
	if g := in.Git; g != nil {
		state := "git: on branch " + g.Branch
		if sum := g.Summary(); sum != "" {
			state = "git: " + sum
		}
		out = append(out, state)
	}
	return out
}

// Ask puts a question to the oracle.
func (b *Brain) Ask(question string) []Reaction {
	question = strings.TrimSpace(question)
	if question == "" {
		return nil
	}
	if !b.cfg.Oracle.Configured() {
		return []Reaction{{Text: b.pick("no_oracle"), Mood: personality.MoodSad, Must: true}}
	}
	var background []string
	if b.shell != "" {
		background = append(background, "shell: "+b.shell)
	}
	dir := ""
	var here *Insight
	var after []string
	if b.cfg.ShareContext {
		dir = b.cwd
		if dir != "" {
			background = append(background, "working directory: "+dir)
		}
		if in := b.here; in != nil && in.Dir == dir {
			here = in
		}
		if f := b.lastFail; f != nil {
			after = append(after, fmt.Sprintf("last failed command: %s (exit %d)", f.Cmd, f.Exit))
			if b.lastOut != "" {
				after = append(after, "the end of its output:\n"+b.lastOut)
			}
		}
	}
	oracle, persona := b.cfg.Oracle, b.cfg.Name
	b.tasks = append(b.tasks, func() any {
		// Tell the oracle what the familiar knows about this place, so even
		// one that cannot read files knows where it is. If the familiar has
		// only just learned where it is, have that look around now.
		if dir != "" {
			if here == nil {
				seen := Scout(dir, "ask")
				here = &seen
			}
			background = append(background, here.facts()...)
		}
		background = append(background, after...)
		return oracle.Ask(question, persona, strings.Join(background, "\n"), dir)
	})
	return []Reaction{{Text: b.pick("think"), Style: "think", Act: creature.ActThink, Must: true, Sticky: true, Hold: 2 * time.Second}}
}

// Explain asks the oracle about the most recent failure.
func (b *Brain) Explain() []Reaction {
	f := b.lastFail
	if f == nil {
		return b.deliver(Reaction{Text: "Nothing has failed lately. Enjoy it.", Must: true})
	}
	return b.deliver(b.Ask(fmt.Sprintf("My command `%s` exited with status %d. What most likely went wrong, and what should I try?", f.Cmd, f.Exit))...)
}

// ---- the passage of time ---------------------------------------------------

// OnTick is called about once a second. It produces greetings, reminders and
// idle chatter. awake is false while the familiar is napping.
func (b *Brain) OnTick(awake bool) []Reaction {
	now := b.now()
	if b.greeted && len(b.problems) > 0 {
		// Rules that could not be used should not fail silently.
		msg := fmt.Sprintf("I couldn't follow %s in your config: %v", plural(len(b.problems), "rule"), b.problems[0])
		b.problems = nil
		return b.deliver(Reaction{Text: msg, Mood: personality.MoodSad, Must: true, Sticky: true})
	}
	if !b.greeted {
		b.greeted = true
		b.scheduleChatter()
		h := now.Hour()
		key := "night"
		switch {
		case h >= 5 && h < 12:
			key = "morning"
		case h >= 12 && h < 18:
			key = "afternoon"
		case h >= 18 && h < 23:
			key = "evening"
		}
		return b.deliver(Reaction{Text: b.pick(key), Act: creature.ActWave, Must: true})
	}
	b.pollGitHub(now)
	if !awake {
		return nil
	}
	if r, ok := b.grown(); ok {
		return b.deliver(r)
	}
	working := !b.lastEvent.IsZero() && now.Sub(b.lastEvent) < 10*time.Minute

	if b.cfg.BreakEvery > 0 && working && !b.workSince.IsZero() && now.Sub(b.workSince) >= b.cfg.BreakEvery {
		worked := now.Sub(b.workSince).Round(time.Minute)
		b.workSince = now
		return b.deliver(Reaction{Text: b.pickf("break", human(worked)), Act: creature.ActAlert, Must: true, Sticky: true})
	}
	if h := now.Hour(); h < 5 && working && b.lateDay != now.YearDay() && b.cfg.Talkativeness > 0 {
		b.lateDay = now.YearDay()
		return b.deliver(Reaction{Text: b.pickf("late", now.Format("15:04")), Style: "whisper", Must: true})
	}
	if b.nextChatter.IsZero() || now.Before(b.nextChatter) {
		return nil
	}
	b.scheduleChatter()
	return b.deliver(b.chatter())
}

// chatter is what it says when nothing is happening.
func (b *Brain) chatter() Reaction {
	roll := b.rng.Float64()
	switch {
	case b.cfg.Tips && roll < 0.45:
		pool := lines["tip"]
		switch b.shell {
		case "fish":
			pool = append(append([]string(nil), lines["tip_fish"]...), pool...)
		case "bash", "zsh":
			pool = append(append([]string(nil), lines["tip_bash"]...), pool...)
		}
		tip := pool[b.mem.Tip%len(pool)]
		b.mem.Tip++
		return Reaction{Text: "Tip: " + tip, Hold: 9 * time.Second}
	case len(b.cfg.Sounds) > 0 && roll < 0.75:
		return Reaction{Text: b.cfg.Sounds[b.rng.Intn(len(b.cfg.Sounds))]}
	case roll < 0.85:
		return Reaction{Act: creature.ActSing}
	}
	return Reaction{Text: b.pick("musing"), Style: "think"}
}

// Woke is what it mumbles when something wakes it up.
func (b *Brain) Woke() []Reaction {
	if !b.chance(0.5) {
		return nil
	}
	return b.deliver(b.banter(b.pick("woke"), creature.ActNone, personality.MoodSleepy))
}

// Chat forces a line of idle chatter, for when the user asks it to talk.
func (b *Brain) Chat() []Reaction {
	r := b.chatter()
	for r.Text == "" {
		r = b.chatter()
	}
	r.Must = true
	return b.deliver(r)
}

// grown notices when the bond has reached a new level, and marks the
// occasion: once per level, and with whatever that level unlocks.
func (b *Brain) grown() (Reaction, bool) {
	bond := b.mem.Bond()
	if bond.Level <= b.mem.Level {
		return Reaction{}, false
	}
	b.mem.Level = bond.Level
	r := Reaction{Act: creature.ActCelebrate, Must: true, Sticky: true, Hold: 12 * time.Second}
	r.Text = fmt.Sprintf("%s together, and I think we're %s now.", plural(bond.Days, "day"), withArticle(bond.Title))
	for _, a := range Accessories {
		if a.Level == bond.Level {
			r.Text += fmt.Sprintf(" I found a %s %s. Want to see? `familiar wear %s`", a.Name, a.Glyph, a.Name)
			r.Suggest = "familiar wear " + a.Name
		}
	}
	return r, true
}

// SetVoice changes who the brain speaks as, for when the familiar changes shape.
func (b *Brain) SetVoice(name string, sounds []string, tic string) {
	b.cfg.Name, b.cfg.Sounds, b.cfg.Tic = name, sounds, tic
}

// adopt takes on the rules of the project the user has just walked into, if
// they have said this project's rules can be trusted. A rule file can put
// words in the familiar's mouth and commands on the prompt, so one that
// arrives with a cloned repository is not followed until the user has seen it.
func (b *Brain) adopt(p *Project, err error) []Reaction {
	b.project = nil
	if p == nil {
		return nil
	}
	if b.mem.Trusted[p.Path] != p.Hash {
		key := p.Path + p.Hash
		if b.asked[key] {
			return nil
		}
		b.asked[key] = true
		what := "This project has instructions for me"
		if _, seen := b.mem.Trusted[p.Path]; seen {
			what = "This project's instructions for me have changed"
		}
		return []Reaction{{Text: what + " (" + ProjectFile + "). To review and accept them: `familiar trust`",
			Suggest: "familiar trust", Act: creature.ActAlert, Must: true}}
	}
	if err != nil {
		return []Reaction{{Text: "I can't read this project's instructions: " + clip(err.Error(), 120), Mood: personality.MoodSad, Must: true, Sticky: true}}
	}
	rules, problems := CompileRules(p.Rules)
	b.project = rules
	key := "welcome:" + p.Path + p.Hash
	switch {
	case len(problems) > 0 && !b.asked[key]:
		b.asked[key] = true
		return []Reaction{{Text: fmt.Sprintf("Some of this project's rules don't work: %v", problems[0]), Mood: personality.MoodSad, Must: true, Sticky: true}}
	case p.Welcome != "" && !b.asked[key]:
		b.asked[key] = true
		return []Reaction{{Text: p.Welcome, Act: creature.ActWave, Must: true, Sticky: true}}
	}
	return nil
}

// Rescan looks at the current directory again, for when its rule file has
// just been trusted or edited.
func (b *Brain) Rescan() {
	if b.cwd == "" {
		return
	}
	dir := b.cwd
	b.asked = map[string]bool{}
	b.tasks = append(b.tasks, func() any { return Scout(dir, "rescan") })
}

// Transcript is a finished command together with what it printed.
type Transcript struct {
	Event     ipc.Event
	Output    string
	Unfocused bool // the terminal was not the focused window when it finished
}

// ReadTranscript collects the output a wrapped shell recorded for ev. The
// wrapper writes the file as the command ends, a moment before or after the
// shell hook reports it, so this waits briefly for the right one to appear.
func ReadTranscript(ev ipc.Event) Transcript {
	t := Transcript{Event: ev}
	want := fmt.Sprint(ev.Seq)
	for i := 0; i < 30; i++ {
		if data, err := os.ReadFile(ev.Transcript); err == nil {
			if seq, rest, _ := strings.Cut(string(data), "\n"); seq == want {
				t.Output = rest
				break
			}
		}
		time.Sleep(10 * time.Millisecond)
	}
	if focus, err := os.ReadFile(ev.Transcript + ".focus"); err == nil {
		t.Unfocused = strings.TrimSpace(string(focus)) == "0"
	}
	return t
}
