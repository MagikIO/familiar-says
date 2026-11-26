package brain

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// GitState is a snapshot of a working tree.
type GitState struct {
	Root      string
	Branch    string
	Detached  bool
	Upstream  bool
	Ahead     int
	Behind    int
	Changed   int // tracked files with staged or unstaged changes
	Untracked int
	Conflicts int
}

// Insight is what a look around a directory turned up.
type Insight struct {
	Dir     string
	Trigger string // "cd" when the user arrived here, "git" after a git command
	Git     *GitState
	Kind    string   // e.g. "Go module"; empty when the directory is nothing special
	Handy   []string // a few commands worth knowing here

	Project    *Project // the project's own instructions for the familiar, if it has any
	ProjectErr error    // what was wrong with them
}

// Root identifies the project the insight is about.
func (in Insight) Root() string {
	if in.Git != nil && in.Git.Root != "" {
		return in.Git.Root
	}
	return in.Dir
}

// Scout inspects dir. It shells out to git and reads a few small files, so it
// must not run on the render loop.
func Scout(dir, trigger string) Insight {
	in := Insight{Dir: dir, Trigger: trigger}
	in.Project, in.ProjectErr = FindProject(dir)
	if trigger == "rescan" {
		return in
	}
	in.Git = gitState(dir)
	root := dir
	if in.Git != nil && in.Git.Root != "" {
		root = in.Git.Root
	}
	in.Kind, in.Handy = project(root)
	if in.Kind == "" && root != dir {
		in.Kind, in.Handy = project(dir)
	}
	return in
}

func gitState(dir string) *GitState {
	ctx, cancel := context.WithTimeout(context.Background(), 1500*time.Millisecond)
	defer cancel()
	cmd := exec.CommandContext(ctx, "git", "-C", dir, "status", "--porcelain=v2", "--branch")
	cmd.Env = append(os.Environ(), "GIT_OPTIONAL_LOCKS=0", "LC_ALL=C")
	out, err := cmd.Output()
	if err != nil {
		return nil
	}
	st := parseGitStatus(string(out))
	if top, err := exec.CommandContext(ctx, "git", "-C", dir, "rev-parse", "--show-toplevel").Output(); err == nil {
		st.Root = strings.TrimSpace(string(top))
	}
	return st
}

// parseGitStatus reads `git status --porcelain=v2 --branch` output.
func parseGitStatus(out string) *GitState {
	st := &GitState{}
	sc := bufio.NewScanner(strings.NewReader(out))
	for sc.Scan() {
		line := sc.Text()
		switch {
		case strings.HasPrefix(line, "# branch.head "):
			st.Branch = strings.TrimPrefix(line, "# branch.head ")
			if st.Branch == "(detached)" {
				st.Detached = true
			}
		case strings.HasPrefix(line, "# branch.upstream "):
			st.Upstream = true
		case strings.HasPrefix(line, "# branch.ab "):
			for _, f := range strings.Fields(strings.TrimPrefix(line, "# branch.ab ")) {
				n, _ := strconv.Atoi(f[1:])
				if f[0] == '+' {
					st.Ahead = n
				} else {
					st.Behind = n
				}
			}
		case strings.HasPrefix(line, "1 "), strings.HasPrefix(line, "2 "):
			st.Changed++
		case strings.HasPrefix(line, "u "):
			st.Conflicts++
		case strings.HasPrefix(line, "? "):
			st.Untracked++
		}
	}
	return st
}

// Summary describes the tree in a short clause, or "" if there is nothing
// worth mentioning.
func (g *GitState) Summary() string {
	if g == nil {
		return ""
	}
	var parts []string
	if g.Conflicts > 0 {
		parts = append(parts, plural(g.Conflicts, "conflict"))
	}
	if g.Changed > 0 {
		parts = append(parts, fmt.Sprintf("%d changed", g.Changed))
	}
	if g.Untracked > 0 {
		parts = append(parts, fmt.Sprintf("%d untracked", g.Untracked))
	}
	if g.Ahead > 0 {
		parts = append(parts, fmt.Sprintf("%d unpushed", g.Ahead))
	}
	if g.Behind > 0 {
		parts = append(parts, fmt.Sprintf("%d behind", g.Behind))
	}
	where := "On `" + g.Branch + "`"
	if g.Detached {
		where = "Detached HEAD"
	}
	if len(parts) == 0 {
		return ""
	}
	return where + ": " + strings.Join(parts, ", ") + "."
}

func plural(n int, word string) string {
	if n == 1 {
		return "1 " + word
	}
	return fmt.Sprintf("%d %ss", n, word)
}

func exists(dir string, names ...string) bool {
	for _, n := range names {
		if _, err := os.Stat(filepath.Join(dir, n)); err == nil {
			return true
		}
	}
	return false
}

// project recognises what kind of project lives in dir and which commands are
// most likely to be useful there.
func project(dir string) (kind string, handy []string) {
	add := func(cmds ...string) { handy = append(handy, cmds...) }

	switch {
	case exists(dir, "go.mod"):
		kind = "Go module"
		add("go test ./...")
	case exists(dir, "Cargo.toml"):
		kind = "Rust crate"
		add("cargo test", "cargo run")
	case exists(dir, "package.json"):
		kind = "Node project"
		add(nodeScripts(dir)...)
	case exists(dir, "pyproject.toml", "setup.py", "requirements.txt"):
		kind = "Python project"
		switch {
		case exists(dir, "uv.lock"):
			add("uv run pytest")
		case exists(dir, "poetry.lock"):
			add("poetry run pytest")
		case exists(dir, "tests", "test"):
			add("pytest")
		}
	case exists(dir, "Gemfile"):
		kind = "Ruby project"
		add("bundle exec rake")
	case exists(dir, "mix.exs"):
		kind = "Elixir project"
		add("mix test")
	case exists(dir, "build.zig"):
		kind = "Zig project"
		add("zig build test")
	case exists(dir, "CMakeLists.txt"):
		kind = "CMake project"
		add("cmake -B build && cmake --build build")
	}

	if targets := makeTargets(filepath.Join(dir, "Makefile")); len(targets) > 0 {
		if kind == "" {
			kind = "Makefile"
		}
		for _, t := range targets {
			add("make " + t)
		}
	}
	if exists(dir, "justfile", "Justfile", ".justfile") {
		if kind == "" {
			kind = "justfile"
		}
		add("just --list")
	}
	if exists(dir, "compose.yaml", "compose.yml", "docker-compose.yml", "docker-compose.yaml") {
		if kind == "" {
			kind = "Compose stack"
		}
		add("docker compose up")
	}
	if len(handy) > 3 {
		handy = handy[:3]
	}
	return kind, handy
}

func nodeScripts(dir string) []string {
	data, err := os.ReadFile(filepath.Join(dir, "package.json"))
	if err != nil || len(data) > 1<<20 {
		return nil
	}
	var pkg struct {
		Scripts map[string]string `json:"scripts"`
	}
	if json.Unmarshal(data, &pkg) != nil {
		return nil
	}
	pm, run := "npm", "npm run "
	switch {
	case exists(dir, "pnpm-lock.yaml"):
		pm, run = "pnpm", "pnpm "
	case exists(dir, "yarn.lock"):
		pm, run = "yarn", "yarn "
	case exists(dir, "bun.lockb", "bun.lock"):
		pm, run = "bun", "bun run "
	}
	var out []string
	for _, name := range []string{"dev", "start", "test", "build", "lint"} {
		if _, ok := pkg.Scripts[name]; !ok {
			continue
		}
		if pm == "npm" && (name == "test" || name == "start") {
			out = append(out, "npm "+name)
		} else {
			out = append(out, run+name)
		}
	}
	return out
}

var makeTarget = regexp.MustCompile(`^([A-Za-z0-9][A-Za-z0-9_-]*)\s*:([^=]|$)`)

// makeTargets returns the most useful-looking targets of a Makefile.
func makeTargets(path string) []string {
	f, err := os.Open(path)
	if err != nil {
		return nil
	}
	defer f.Close()
	seen := map[string]bool{}
	var all []string
	sc := bufio.NewScanner(f)
	for sc.Scan() && len(all) < 60 {
		if m := makeTarget.FindStringSubmatch(sc.Text()); m != nil && !seen[m[1]] {
			seen[m[1]] = true
			all = append(all, m[1])
		}
	}
	var out []string
	for _, want := range []string{"test", "build", "run", "dev", "lint", "check"} {
		if seen[want] {
			out = append(out, want)
		}
	}
	if len(out) == 0 && len(all) > 0 {
		out = all[:min(2, len(all))]
	}
	if len(out) > 2 {
		out = out[:2]
	}
	return out
}
