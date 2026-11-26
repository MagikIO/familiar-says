package brain

import (
	"math/rand"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/MagikIO/familiar-says/internal/creature"
	"github.com/MagikIO/familiar-says/internal/ipc"
)

type clock struct{ t time.Time }

func (c *clock) now() time.Time          { return c.t }
func (c *clock) advance(d time.Duration) { c.t = c.t.Add(d) }

// newBrain returns a brain that has already said hello, on a Wednesday
// afternoon, with a clock the test controls.
func newBrain(t *testing.T, tweak func(*Config)) (*Brain, *clock) {
	t.Helper()
	ck := &clock{t: time.Date(2026, 3, 4, 14, 0, 0, 0, time.Local)}
	cfg := DefaultConfig()
	if tweak != nil {
		tweak(&cfg)
	}
	b := New(cfg, LoadMemory(""), rand.New(rand.NewSource(1)), ck.now)
	b.OnTick(true)
	ck.advance(time.Minute)
	return b, ck
}

func ran(cmd string, exit int) ipc.Event {
	return ipc.Event{Type: ipc.TypeCommand, Cmd: cmd, Exit: exit, Shell: "fish"}
}

func text(rs []Reaction) string {
	var parts []string
	for _, r := range rs {
		parts = append(parts, r.Text)
	}
	return strings.Join(parts, " | ")
}

func TestDistance(t *testing.T) {
	cases := []struct {
		a, b string
		want int
	}{
		{"git", "git", 0}, {"gti", "git", 1}, {"gt", "git", 1}, {"gitt", "git", 1},
		{"stauts", "status", 1}, {"comit", "commit", 1}, {"abc", "xyz", 3}, {"", "abc", 3},
	}
	for _, c := range cases {
		if got := distance(c.a, c.b); got != c.want {
			t.Errorf("distance(%q, %q) = %d, want %d", c.a, c.b, got, c.want)
		}
	}
}

func TestParseLooksThroughWrappers(t *testing.T) {
	p := parse("FOO=1 sudo -E /usr/bin/git  -C repo push --force")
	if p.Prog != "git" || p.Sub != "repo" || !p.Sudo {
		t.Errorf("got prog=%q sub=%q sudo=%v", p.Prog, p.Sub, p.Sudo)
	}
	if p := parse("go test ./..."); p.family() != "go test" || !p.isTest() {
		t.Errorf("go test should be its own family and a test command: %q", p.family())
	}
	if p := parse("make"); !p.isBuild() {
		t.Error("bare make should count as a build")
	}
	if p := parse("   "); p.Prog != "" {
		t.Errorf("blank line should parse to nothing, got %q", p.Prog)
	}
}

func TestTypoGetsCorrectedWithSuggestion(t *testing.T) {
	b, _ := newBrain(t, nil)
	rs := b.OnEvent(ran("gti status", 127))
	if len(rs) != 1 || !strings.Contains(rs[0].Text, "Did you mean `git status`?") {
		t.Fatalf("expected a correction, got %q", text(rs))
	}
	if rs[0].Suggest != "git status" {
		t.Errorf("the fix should be offered as a ready-to-run suggestion, got %q", rs[0].Suggest)
	}

	rs = b.OnEvent(ran("git stauts", 1))
	if len(rs) != 1 || rs[0].Suggest != "git status" {
		t.Errorf("subcommand typos should be corrected too, got %q", text(rs))
	}

	rs = b.OnEvent(ran("sudo dockre ps", 127))
	if len(rs) != 1 || rs[0].Suggest != "sudo docker ps" {
		t.Errorf("the correction should keep sudo, got %+v", rs)
	}
}

func TestUnknownCommandWithNoNearMatch(t *testing.T) {
	b, _ := newBrain(t, nil)
	rs := b.OnEvent(ran("zzqqxxyy --flag", 127))
	if len(rs) != 1 || !strings.Contains(rs[0].Text, "isn't on your PATH") || rs[0].Suggest != "" {
		t.Errorf("got %+v", rs)
	}
}

func TestPathIndexFindsLessCommonTools(t *testing.T) {
	b, _ := newBrain(t, nil)
	b.Learn(PathIndex{"ripgrep-all": true, "shellcheck": true})
	rs := b.OnEvent(ran("shellchekc script.sh", 127))
	if len(rs) != 1 || rs[0].Suggest != "shellcheck script.sh" {
		t.Errorf("should correct against what is installed, got %+v", rs)
	}
}

func TestBenignFailuresAreNotCommentedOn(t *testing.T) {
	b, _ := newBrain(t, nil)
	for _, cmd := range []string{"grep needle haystack", "diff a b", "test -f nope", "git diff --quiet", "rg TODO"} {
		if rs := b.OnEvent(ran(cmd, 1)); len(rs) != 0 {
			t.Errorf("%q exiting 1 is an answer, not an error; got %q", cmd, text(rs))
		}
	}
	if b.LastFailure() != nil {
		t.Error("benign failures should not be remembered as failures")
	}
}

func TestCtrlCIsNotAFailure(t *testing.T) {
	b, _ := newBrain(t, func(c *Config) { c.Talkativeness = 0 })
	if rs := b.OnEvent(ran("sleep 100", 130)); len(rs) != 0 {
		t.Errorf("a quiet familiar should ignore Ctrl-C, got %q", text(rs))
	}
	if b.LastFailure() != nil {
		t.Error("an interrupt should not count as the last failure")
	}
}

func TestExitCodesAreExplained(t *testing.T) {
	cases := map[string]struct {
		cmd  string
		exit int
		want string
	}{
		"oom":      {"./train", 137, "SIGKILL"},
		"segv":     {"./a.out", 139, "Segfault"},
		"curl dns": {"curl https://exmaple.invalid", 6, "resolve"},
		"ssh":      {"ssh box", 255, "ssh -v"},
		"push":     {"git push", 1, "git pull --rebase"},
	}
	for name, c := range cases {
		b, _ := newBrain(t, nil)
		rs := b.OnEvent(ran(c.cmd, c.exit))
		if !strings.Contains(text(rs), c.want) {
			t.Errorf("%s: want mention of %q, got %q", name, c.want, text(rs))
		}
	}
}

func TestNotExecutableOffersChmod(t *testing.T) {
	b, _ := newBrain(t, nil)
	rs := b.OnEvent(ran("./deploy.sh prod", 126))
	if len(rs) != 1 || rs[0].Suggest != "chmod +x ./deploy.sh" {
		t.Errorf("got %+v", rs)
	}
}

func TestMissingPathSuggestsNeighbour(t *testing.T) {
	dir := t.TempDir()
	os.Mkdir(filepath.Join(dir, "internal"), 0o755)
	os.WriteFile(filepath.Join(dir, "README.md"), nil, 0o644)

	b, _ := newBrain(t, nil)
	ev := ran("cd intenral", 1)
	ev.Cwd = dir
	rs := b.OnEvent(ev)
	if len(rs) != 1 || rs[0].Suggest != "cd internal" {
		t.Fatalf("got %+v", rs)
	}

	ev = ran("cat READNE.md", 1)
	ev.Cwd = dir
	if rs := b.OnEvent(ev); len(rs) != 1 || rs[0].Suggest != "cat README.md" {
		t.Errorf("got %+v", rs)
	}

	ev = ran("cd READNE.md", 1) // cd should only ever be pointed at directories
	ev.Cwd = dir
	for _, r := range b.OnEvent(ev) {
		if r.Suggest != "" {
			t.Errorf("cd must not be offered a file: %+v", r)
		}
	}
}

func TestLongCommandsAreAnnounced(t *testing.T) {
	b, _ := newBrain(t, nil)
	ev := ran("cargo build --release", 0)
	ev.DurationMs = 151_000
	rs := b.OnEvent(ev)
	if len(rs) != 1 || !strings.Contains(rs[0].Text, "2m31s") || !rs[0].Must {
		t.Errorf("got %+v", rs)
	}
	ev.Exit, ev.Cmd = 2, "./slow-thing"
	if rs := b.OnEvent(ev); !strings.Contains(text(rs), "2m31s") {
		t.Errorf("a long failure should mention how long it took, got %q", text(rs))
	}
}

func TestRecoveryAfterAStreakIsCelebrated(t *testing.T) {
	b, ck := newBrain(t, nil)
	for i := 0; i < 3; i++ {
		b.OnEvent(ran("go test ./...", 1))
		ck.advance(time.Minute)
	}
	rs := b.OnEvent(ran("go test ./pkg", 0)) // same family, different arguments
	if len(rs) != 1 || rs[0].Act != creature.ActCelebrate || !strings.Contains(rs[0].Text, "4") {
		t.Errorf("want a celebration that counts the attempts, got %+v", rs)
	}
	if rs := b.OnEvent(ran("go test ./pkg", 0)); len(rs) > 0 && rs[0].Act == creature.ActCelebrate {
		t.Error("the streak should be over after one celebration")
	}
}

func TestThirdFailureGetsAdvice(t *testing.T) {
	b, ck := newBrain(t, func(c *Config) { c.Talkativeness = 0 })
	var last []Reaction
	for i := 0; i < 3; i++ {
		last = b.OnEvent(ran("./flaky", 1))
		ck.advance(time.Minute)
	}
	if !strings.Contains(text(last), "3") {
		t.Errorf("even a quiet familiar should speak up on the third failure, got %q", text(last))
	}
}

func TestBanterIsRateLimitedButAdviceIsNot(t *testing.T) {
	b, _ := newBrain(t, func(c *Config) { c.Talkativeness = 1 })
	spoke := 0
	for i := 0; i < 40; i++ { // forty passing test runs inside the same instant
		for _, r := range b.OnEvent(ran("go test ./...", 0)) {
			if r.Text != "" {
				spoke++
			}
		}
	}
	if spoke > 1 {
		t.Errorf("banter should be limited to one remark per quiet period, got %d", spoke)
	}
	if rs := b.OnEvent(ran("gti log", 127)); len(rs) != 1 || rs[0].Text == "" {
		t.Errorf("a correction must get through regardless, got %+v", rs)
	}
}

func TestQuietFamiliarOnlyGivesEssentials(t *testing.T) {
	b, ck := newBrain(t, func(c *Config) { c.Talkativeness = 0 })
	for i := 0; i < 200; i++ {
		ck.advance(time.Minute)
		for _, r := range append(b.OnEvent(ran("go test ./...", 0)), b.OnTick(true)...) {
			if r.Text != "" && !r.Must {
				t.Fatalf("talkativeness 0 should silence banter, got %q", r.Text)
			}
		}
	}
}

func TestRepeatedCommandGetsAShorthandOnce(t *testing.T) {
	b, ck := newBrain(t, nil)
	var tips []Reaction
	for i := 0; i < 12; i++ {
		ck.advance(time.Minute)
		for _, r := range b.OnEvent(ran("docker compose up -d", 0)) {
			if r.Suggest != "" {
				tips = append(tips, r)
			}
		}
	}
	if len(tips) != 1 {
		t.Fatalf("want exactly one shorthand tip, got %d", len(tips))
	}
	if tips[0].Suggest != "abbr -a dcu docker compose up -d" {
		t.Errorf("fish users should be offered an abbr, got %q", tips[0].Suggest)
	}

	if tip, ok := shorthand("git status --short", "bash", nil); !ok || tip != "alias gs='git status --short'" {
		t.Errorf("bash users should be offered an alias, got %q", tip)
	}
	if _, ok := shorthand("git status --short", "bash", map[string]bool{"gs": true}); ok {
		t.Error("should not suggest a name that is already a command")
	}
	if _, ok := shorthand("cat a | grep b", "fish", nil); ok {
		t.Error("pipelines are not candidates for an abbreviation")
	}
}

func TestAcceptKeyHintFadesOut(t *testing.T) {
	b, ck := newBrain(t, nil)
	hinted := 0
	for i := 0; i < 6; i++ {
		ck.advance(time.Minute)
		if strings.Contains(text(b.OnEvent(ran("gti status", 127))), "Alt-G") {
			hinted++
		}
	}
	if hinted != 3 {
		t.Errorf("the Alt-G hint should be shown three times and then dropped, shown %d", hinted)
	}
}

func TestDangerousCommandsGetALook(t *testing.T) {
	b, _ := newBrain(t, nil)
	pre := func(cmd string) []Reaction { return b.OnEvent(ipc.Event{Type: ipc.TypePreexec, Cmd: cmd}) }

	rs := pre("git push --force origin main")
	if len(rs) != 1 || rs[0].Act != creature.ActNervous || !strings.Contains(rs[0].Text, "--force-with-lease") {
		t.Errorf("force push should get advice, got %+v", rs)
	}
	for _, cmd := range []string{"rm -rf build", "sudo rm -fr /var/tmp/x", "git reset --hard HEAD~3", "kubectl delete ns prod"} {
		if rs := pre(cmd); len(rs) != 1 || rs[0].Act != creature.ActNervous {
			t.Errorf("%q should make it nervous, got %+v", cmd, rs)
		}
	}
	for _, cmd := range []string{"rm notes.txt", "git push", "ls -rf", "git clean -n"} {
		if rs := pre(cmd); len(rs) != 0 {
			t.Errorf("%q is harmless, got %+v", cmd, rs)
		}
	}
}

func TestGreetingSuitsTheHour(t *testing.T) {
	for hour, want := range map[int][]string{8: lines["morning"], 15: lines["afternoon"], 20: lines["evening"], 2: lines["night"]} {
		ck := &clock{t: time.Date(2026, 3, 4, hour, 0, 0, 0, time.Local)}
		b := New(DefaultConfig(), nil, rand.New(rand.NewSource(1)), ck.now)
		rs := b.OnTick(true)
		if len(rs) != 1 {
			t.Fatalf("hour %d: want one greeting, got %d", hour, len(rs))
		}
		found := false
		for _, w := range want {
			found = found || rs[0].Text == w
		}
		if !found {
			t.Errorf("hour %d: %q is not one of that hour's greetings", hour, rs[0].Text)
		}
		if again := b.OnTick(true); len(again) != 0 {
			t.Errorf("hour %d: should only greet once, got %q", hour, text(again))
		}
	}
}

func TestBreakReminderAfterSteadyWork(t *testing.T) {
	b, ck := newBrain(t, func(c *Config) { c.Talkativeness = 0; c.BreakEvery = 30 * time.Minute })
	reminded := false
	for i := 0; i < 40; i++ {
		ck.advance(time.Minute)
		b.OnEvent(ran("ls", 0))
		if strings.Contains(text(b.OnTick(true)), "30m") {
			reminded = true
			break
		}
	}
	if !reminded {
		t.Error("expected a break reminder after 30 minutes of steady commands")
	}
}

func TestIdleChatterArrivesEventuallyButNotWhileAsleep(t *testing.T) {
	b, ck := newBrain(t, nil)
	for i := 0; i < 30; i++ {
		ck.advance(time.Minute)
		if rs := b.OnTick(false); len(rs) != 0 {
			t.Fatalf("a sleeping familiar should not chatter, got %q", text(rs))
		}
	}
	var said []Reaction
	for i := 0; i < 30 && len(said) == 0; i++ {
		ck.advance(time.Minute)
		said = b.OnTick(true)
	}
	if len(said) == 0 {
		t.Error("an awake, idle familiar should say or do something within half an hour")
	}
}

func TestGitStatusParsing(t *testing.T) {
	out := "# branch.oid abc\n# branch.head feature/x\n# branch.upstream origin/feature/x\n# branch.ab +2 -1\n" +
		"1 .M N... 100644 100644 100644 a b file.go\n2 R. N... 100644 100644 100644 a b R100 new.go\told.go\n" +
		"u UU N... 100644 100644 100644 100644 a b c conflict.go\n? notes.txt\n? scratch/\n"
	st := parseGitStatus(out)
	if st.Branch != "feature/x" || !st.Upstream || st.Ahead != 2 || st.Behind != 1 ||
		st.Changed != 2 || st.Untracked != 2 || st.Conflicts != 1 {
		t.Errorf("parsed wrongly: %+v", st)
	}
	if got := st.Summary(); got != "On `feature/x`: 1 conflict, 2 changed, 2 untracked, 2 unpushed, 1 behind." {
		t.Errorf("summary: %q", got)
	}
	if got := parseGitStatus("# branch.head main\n").Summary(); got != "" {
		t.Errorf("a clean tree has nothing to report, got %q", got)
	}
}

func TestProjectDetection(t *testing.T) {
	write := func(dir, name, body string) { os.WriteFile(filepath.Join(dir, name), []byte(body), 0o644) }

	goDir := t.TempDir()
	write(goDir, "go.mod", "module x\n")
	write(goDir, "Makefile", ".PHONY: build test\nVERSION := 1\nbuild:\n\tgo build\ntest: build\n\tgo test\nclean:\n")
	if kind, handy := project(goDir); kind != "Go module" || strings.Join(handy, "; ") != "go test ./...; make test; make build" {
		t.Errorf("go: %q %q", kind, handy)
	}

	nodeDir := t.TempDir()
	write(nodeDir, "package.json", `{"scripts":{"dev":"vite","test":"vitest","build":"vite build","prepare":"husky"}}`)
	write(nodeDir, "pnpm-lock.yaml", "")
	if kind, handy := project(nodeDir); kind != "Node project" || strings.Join(handy, "; ") != "pnpm dev; pnpm test; pnpm build" {
		t.Errorf("node: %q %q", kind, handy)
	}

	if kind, handy := project(t.TempDir()); kind != "" || len(handy) != 0 {
		t.Errorf("an empty directory is not a project: %q %q", kind, handy)
	}
}

func TestArrivingInAProjectGetsOneBriefing(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not installed")
	}
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module x\n"), 0o644)
	git := func(args ...string) {
		cmd := exec.Command("git", append([]string{"-C", dir}, args...)...)
		cmd.Env = append(os.Environ(), "GIT_CONFIG_GLOBAL=/dev/null", "GIT_CONFIG_SYSTEM=/dev/null")
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
	git("init", "-q", "-b", "main")

	b, ck := newBrain(t, nil)
	ev := ran("cd "+dir, 0)
	ev.Cwd = dir
	b.OnEvent(ev)
	tasks := b.Tasks()
	if len(tasks) != 1 {
		t.Fatalf("arriving somewhere new should schedule one look around, got %d tasks", len(tasks))
	}
	rs := b.Learn(tasks[0]())
	got := text(rs)
	for _, want := range []string{filepath.Base(dir), "Go module", "`go test ./...`", "On `main`", "1 untracked"} {
		if !strings.Contains(got, want) {
			t.Errorf("briefing should mention %q: %q", want, got)
		}
	}

	// Running more commands in the same place, or coming straight back,
	// must not repeat the briefing.
	ck.advance(time.Minute)
	b.OnEvent(ev)
	if len(b.Tasks()) != 0 {
		t.Error("staying put should not trigger another look")
	}
	ev2 := ran("cd /", 0)
	ev2.Cwd = "/"
	b.OnEvent(ev2)
	b.Tasks()
	b.OnEvent(ev)
	for _, task := range b.Tasks() {
		if rs := b.Learn(task()); len(rs) != 0 {
			t.Errorf("a project briefed minutes ago should not be briefed again: %q", text(rs))
		}
	}
}

func TestOracleAnswersAndFailsGracefully(t *testing.T) {
	if _, err := exec.LookPath("sh"); err != nil {
		t.Skip("no sh")
	}
	b, _ := newBrain(t, func(c *Config) {
		c.Oracle = Oracle{Command: []string{"sh", "-c", `grep -c "Question: why" >/dev/null && echo "Because   of\nreasons."`}}
	})
	b.OnEvent(ran("make deploy", 2))
	rs := b.OnEvent(ipc.Event{Type: ipc.TypeAsk, Text: "why"})
	if len(rs) != 1 || rs[0].Act != creature.ActThink {
		t.Fatalf("asking should start it thinking, got %+v", rs)
	}
	tasks := b.Tasks()
	if len(tasks) != 1 {
		t.Fatalf("want one oracle task, got %d", len(tasks))
	}
	ans := b.Learn(tasks[0]())
	if len(ans) != 1 || ans[0].Text != "Because of reasons." || !ans[0].Sticky {
		t.Errorf("answer should be tidied and sticky, got %+v", ans)
	}

	b, _ = newBrain(t, func(c *Config) { c.Oracle = Oracle{Command: []string{"sh", "-c", "echo nope >&2; exit 3"}} })
	b.OnEvent(ipc.Event{Type: ipc.TypeAsk, Text: "why"})
	if got := text(b.Learn(b.Tasks()[0]())); !strings.Contains(got, "nope") {
		t.Errorf("an oracle failure should be reported, got %q", got)
	}

	b, _ = newBrain(t, nil)
	if got := text(b.OnEvent(ipc.Event{Type: ipc.TypeAsk, Text: "why"})); !strings.Contains(got, "oracle") {
		t.Errorf("with no oracle configured it should say so, got %q", got)
	}
	if len(b.Tasks()) != 0 {
		t.Error("nothing should be run when no oracle is configured")
	}
}

func TestOracleContextIsOptIn(t *testing.T) {
	dir := t.TempDir()
	out := filepath.Join(dir, "prompt")
	oracle := Oracle{Command: []string{"sh", "-c", "cat > " + out + "; echo ok"}}

	ask := func(share bool) string {
		b, _ := newBrain(t, func(c *Config) { c.Oracle = oracle; c.ShareContext = share })
		ev := ran("deploy --token hunter2", 1)
		ev.Cwd = "/srv/secret-project"
		b.OnEvent(ev)
		b.Tasks()
		b.OnEvent(ipc.Event{Type: ipc.TypeAsk, Text: "what now"})
		b.Tasks()[0]()
		data, _ := os.ReadFile(out)
		return string(data)
	}
	if p := ask(true); !strings.Contains(p, "hunter2") || !strings.Contains(p, "/srv/secret-project") {
		t.Errorf("with sharing on, the prompt should carry context:\n%s", p)
	}
	if p := ask(false); strings.Contains(p, "hunter2") || strings.Contains(p, "secret-project") || !strings.Contains(p, "what now") {
		t.Errorf("with sharing off, only the question may be sent:\n%s", p)
	}
}

func TestMemoryRoundTripAndPruning(t *testing.T) {
	path := filepath.Join(t.TempDir(), "state", "memory.json")
	m := LoadMemory(path)
	m.Counts["git status"] = 9
	m.Suggested["git status"] = true
	m.Pets = 3
	for i := 0; i < 500; i++ {
		m.Counts[strings.Repeat("x", i+1)] = 1
	}
	if err := m.Save(); err != nil {
		t.Fatal(err)
	}
	back := LoadMemory(path)
	if back.Counts["git status"] != 9 || !back.Suggested["git status"] || back.Pets != 3 {
		t.Errorf("memory did not survive a round trip: %+v", back)
	}
	if len(back.Counts) > 300 {
		t.Errorf("command tally should be pruned, has %d entries", len(back.Counts))
	}
	if LoadMemory(filepath.Join(t.TempDir(), "missing.json")).Counts == nil {
		t.Error("a missing file should yield a usable blank memory")
	}
}

func TestMemoryMergesWithOtherFamiliars(t *testing.T) {
	path := filepath.Join(t.TempDir(), "memory.json")
	a, b := LoadMemory(path), LoadMemory(path) // two familiars, one memory file

	a.Commands += 10
	a.Pets += 2
	a.Counts["git status"] += 3
	a.Today(time.Date(2026, 3, 4, 9, 0, 0, 0, time.UTC))

	b.Commands += 5
	b.Counts["git status"] += 2
	b.Counts["make"]++
	b.Today(time.Date(2026, 3, 4, 10, 0, 0, 0, time.UTC)) // the same day, noticed separately
	b.Trusted["/repo/.familiar.json"] = "abc"

	if err := a.Save(); err != nil {
		t.Fatal(err)
	}
	if err := b.Save(); err != nil {
		t.Fatal(err)
	}
	if err := a.Save(); err != nil { // a catches up with what b wrote
		t.Fatal(err)
	}
	for name, m := range map[string]*Memory{"a": a, "b": b, "disk": LoadMemory(path)} {
		if m.Commands != 15 || m.Pets != 2 || m.Counts["git status"] != 5 || m.Counts["make"] != 1 {
			t.Errorf("%s: tallies should add up across familiars: %+v", name, m)
		}
		if m.Days != 1 {
			t.Errorf("%s: a day both familiars saw is still one day, got %d", name, m.Days)
		}
		if m.Trusted["/repo/.familiar.json"] != "abc" {
			t.Errorf("%s: trust granted in one shell should reach the others", name)
		}
	}

	// Saving again with nothing new must not inflate anything.
	a.Save()
	b.Save()
	if m := LoadMemory(path); m.Commands != 15 || m.Counts["git status"] != 5 {
		t.Errorf("repeated saves double-counted: %+v", m)
	}
}

func TestBondGrowsAndUnlocksAccessories(t *testing.T) {
	m := LoadMemory("")
	if b := m.Bond(); b.Level != 0 || b.Title != "new acquaintance" {
		t.Fatalf("a new familiar should start as an acquaintance, got %+v", b)
	}
	if err := m.Wear("flower"); err == nil {
		t.Error("nothing should be wearable before it is earned")
	}

	m.Days, m.Commands, m.Pets, m.Fixes = 3, 80, 500, 1
	b := m.Bond()
	if b.Level != 1 {
		t.Fatalf("three days of steady use should make companions, got level %d (%d points)", b.Level, b.Points)
	}
	spam := *m
	spam.Pets = 50_000
	if spam.Bond().Points != b.Points {
		t.Error("petting beyond the daily cap should not add to the bond")
	}

	if err := m.Wear("Flower"); err != nil || m.Wearing != "flower" {
		t.Errorf("the flower should be wearable now: %v", err)
	}
	if err := m.Wear("crown"); err == nil {
		t.Error("the crown is not earned yet")
	}
	if err := m.Wear("monocle"); err == nil {
		t.Error("unknown accessories should be refused")
	}
	if err := m.Wear("none"); err != nil || m.Wearing != "" {
		t.Error("none should take everything off")
	}
}

func TestLevelUpIsCelebratedOnce(t *testing.T) {
	b, ck := newBrain(t, func(c *Config) { c.Talkativeness = 0 })
	b.mem.Days, b.mem.Commands = 3, 80

	ck.advance(time.Minute)
	rs := b.OnTick(true)
	if len(rs) != 1 || rs[0].Act != creature.ActCelebrate || !strings.Contains(rs[0].Text, "companion") {
		t.Fatalf("reaching a new stage should be celebrated, got %+v", rs)
	}
	if rs[0].Suggest != "familiar wear flower" {
		t.Errorf("it should offer to show what it found, got %q", rs[0].Suggest)
	}
	ck.advance(time.Minute)
	if again := b.OnTick(true); len(again) != 0 {
		t.Errorf("the same stage should not be celebrated twice: %q", text(again))
	}

	rs = b.OnEvent(ipc.Event{Type: ipc.TypeWear, Text: "flower"})
	if b.mem.Wearing != "flower" || len(rs) != 1 {
		t.Errorf("wear event should dress it: wearing=%q %+v", b.mem.Wearing, rs)
	}
	rs = b.OnEvent(ipc.Event{Type: ipc.TypeWear, Text: "halo"})
	if b.mem.Wearing != "flower" || !strings.Contains(text(rs), "can't") {
		t.Errorf("an unearned accessory should be declined, got %q", text(rs))
	}
}
