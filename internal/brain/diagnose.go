package brain

import (
	"fmt"
	"regexp"
	"strings"
)

// Diagnosis is what the familiar makes of a failed command's output.
type Diagnosis struct {
	Text    string // what to say
	Suggest string // a command that would likely help, if there is one
}

// remedy pairs a recognisable error with what to do about it. In Say and Fix,
// $1..$9 are the pattern's capture groups.
type remedy struct {
	re  *regexp.Regexp
	Say string
	Fix string
}

func r(pattern, say, fix string) remedy {
	return remedy{regexp.MustCompile(`(?m)` + pattern), say, fix}
}

// remedies is checked in order; the first match wins, so more specific
// errors come before the general ones they resemble.
var remedies = []remedy{
	// ---- git ----
	r(`fatal: not a git repository`, "This isn't inside a git repository. Wrong directory?", ""),
	r(`The most similar commands? (?:is|are)\s+(\S+)`, "git suggests `$1`.", ""),
	r(`rejected\]?.*\((?:non-fast-forward|fetch first)\)`, "The remote has commits you don't. Rebase onto them, then push again.", "git pull --rebase"),
	r(`has no upstream branch`, "This branch isn't tracking a remote yet.", "git push -u origin HEAD"),
	r(`CONFLICT \(`, "Merge conflicts. `git status` lists the files; fix them, `git add`, then continue.", "git status"),
	r(`Please commit your changes or stash them`, "Local changes are in the way. Stash them, do the thing, then `git stash pop`.", "git stash"),
	r(`Please tell me who you are`, "Git doesn't know your name yet.", `git config --global user.name "Your Name"`),
	r(`nothing to commit, working tree clean`, "There's nothing to commit: the tree is clean.", ""),
	r(`no changes added to commit`, "Nothing is staged. `git add -p` lets you choose what goes in.", "git add -p"),
	r(`Permission denied \(publickey\)`, "The server refused your SSH key. Is the right key loaded?", "ssh-add -l"),
	r(`pathspec '([^']+)' did not match`, "Git knows no branch or file called `$1`. `git branch -a` shows what exists.", "git branch -a"),
	r(`detected dubious ownership in repository at '([^']+)'`, "Git distrusts this repo's owner.", "git config --global --add safe.directory $1"),

	// ---- Go ----
	r(`missing go\.sum entry|go: updates to go\.(?:mod|sum) needed`, "go.mod and go.sum are out of step.", "go mod tidy"),
	r(`no required module provides package (\S+?);`, "Nothing in go.mod provides `$1`.", "go get $1"),
	r(`go: cannot find main module`, "No go.mod here or above. Wrong directory, or a new module?", ""),
	r(`declared and not used: (\S+)`, "Go won't compile with `$1` unused. Use it or drop it.", ""),
	r(`"(\S+)" imported and not used`, "Unused import `$1`.", ""),

	// ---- Node ----
	r(`npm (?:ERR!|error) code ERESOLVE`, "npm can't reconcile peer dependencies. Read which packages disagree before forcing it.", "npm install --legacy-peer-deps"),
	r(`npm (?:ERR!|error) code ENOENT[\s\S]*?package\.json`, "No package.json here. Wrong directory?", ""),
	r(`npm (?:ERR!|error) Missing script: "?([\w:.-]+)"?`, "There's no `$1` script in package.json. `npm run` lists the ones there are.", "npm run"),
	r(`Cannot find module '([^'./][^']*)'`, "Node can't find `$1`. Not installed?", "npm install $1"),
	r(`ERR_MODULE_NOT_FOUND|Cannot find module '\.`, "A local import points at a file that isn't there. Check the path and extension.", ""),
	r(`EADDRINUSE.*?:(\d+)|address already in use.*?:(\d+)|[Pp]ort (\d+) is already in use`, "Port $1$2$3 is taken. This shows what has it:", "lsof -i :$1$2$3"),
	r(`ENOSPC: System limit for number of file watchers`, "You've run out of inotify watchers.", "sudo sysctl fs.inotify.max_user_watches=524288"),
	r(`JavaScript heap out of memory`, "Node ran out of heap.", `export NODE_OPTIONS=--max-old-space-size=4096`),

	// ---- Python ----
	r(`ModuleNotFoundError: No module named '([\w.]+)'`, "Python can't import `$1`. Is the right virtualenv active?", "pip install $1"),
	r(`externally-managed-environment`, "This Python belongs to the system. Use a virtualenv.", "python3 -m venv .venv && source .venv/bin/activate"),
	r(`IndentationError: (.+)`, "Indentation error: $1. Mixed tabs and spaces?", ""),

	// ---- Rust ----
	r(`error\[E0\d+\]: (.+)[\s\S]{0,200}?--> (\S+?:\d+)`, "First error at $2: $1.", ""),
	r(`could not find .Cargo\.toml.`, "No Cargo.toml here or above. Wrong directory?", ""),

	// ---- Docker ----
	r(`permission denied while trying to connect to the Docker daemon`, "You can't reach the Docker socket. Add yourself to the docker group (then log in again).", "sudo usermod -aG docker $USER"),
	r(`Cannot connect to the Docker daemon`, "The Docker daemon isn't running.", "sudo systemctl start docker"),
	r(`port is already allocated|Bind for [\d.]+:(\d+) failed`, "Another container already has that port.", "docker ps"),
	r(`pull access denied for (\S+?),`, "Can't pull `$1`. A typo in the image name, or a private image needing `docker login`?", ""),

	// ---- systems ----
	r(`No space left on device`, "The disk is full.", "df -h"),
	r(`Too many open files`, "You've hit the open-file limit.", "ulimit -n 4096"),
	r(`[Cc]ould not resolve host:? (\S+)|Temporary failure in name resolution`, "DNS couldn't resolve that host $1. Typo, or is the network down?", ""),
	r(`Connection refused`, "Connection refused: nothing is listening there. Is the service up?", ""),
	r(`certificate has expired|certificate verify failed`, "A TLS certificate didn't check out. Is the system clock right?", "date"),
	r(`(?:E: Could not open lock file|are you root\?|must be (?:run as )?root|Operation not permitted)`, "That needs root.", "sudo {cmd}"),
	r(`^(?:\S+: )?(?:\S+: )?(\S+): Permission denied`, "No permission for `$1`. `ls -l` shows who can touch it.", "ls -l $1"),
	r(`Is a directory`, "That's a directory, and the command wanted a file.", ""),
	r(`command not found: (\S+)|(\S+): command not found`, "A script tried to run `$1$2`, which isn't installed.", ""),
}

var (
	// file:line[:col]: message — the shape most compilers and linters use.
	located = regexp.MustCompile(`(?m)^\s*([\w./\\@+-]+\.\w+)[:(](\d+)(?:[:,]\d+)?\)?:?\s+(?:-\s+)?(?:fatal )?(?:error|Error)?:?\s*(.+)$`)
	// Lines that announce themselves as errors.
	announced = regexp.MustCompile(`(?m)^\s*(?:\S+: )?(?:fatal|FATAL|error|Error|ERROR|panic)(?:\[\w+\])?:\s*(.+)$`)
	// Python's final traceback line.
	pyError  = regexp.MustCompile(`(?m)^(\w+(?:\.\w+)*(?:Error|Exception)): (.+)$`)
	pyWhere  = regexp.MustCompile(`(?m)^\s+File "([^"]+)", line (\d+)`)
	failures = []*regexp.Regexp{
		regexp.MustCompile(`(?m)^\s*--- FAIL: (\S+)`),                   // go test
		regexp.MustCompile(`(?m)^FAILED (\S+)`),                         // pytest
		regexp.MustCompile(`(?m)^test (\S+) \.\.\. FAILED`),             // cargo test
		regexp.MustCompile(`(?m)^\s*(?:✕|×|✗) (.+?)(?: \(\d+ ?ms\))?$`), // jest, vitest
	}
)

// Diagnose reads the output of a failed command and says what went wrong, if
// it can tell. Known errors come with a fix. Otherwise, for output long
// enough to have scrolled, it points at the error worth reading first.
func Diagnose(output string, p parsed) (Diagnosis, bool) {
	if strings.TrimSpace(output) == "" {
		return Diagnosis{}, false
	}
	for _, rem := range remedies {
		m := rem.re.FindStringSubmatchIndex(output)
		if m == nil {
			continue
		}
		expand := func(tmpl string) string {
			s := string(rem.re.ExpandString(nil, strings.ReplaceAll(tmpl, "$USER", "$$USER"), output, m))
			return strings.ReplaceAll(strings.TrimSpace(s), "{cmd}", p.Line)
		}
		return Diagnosis{Text: tidy(expand(rem.Say)), Suggest: expand(rem.Fix)}, true
	}

	// Which tests failed is worth knowing however short the output.
	for _, re := range failures {
		if ms := re.FindAllStringSubmatch(output, -1); len(ms) > 0 {
			names := make([]string, 0, 3)
			for _, m := range ms[:min(3, len(ms))] {
				names = append(names, "`"+clip(m[1], 40)+"`")
			}
			text := "Failing: " + strings.Join(names, ", ")
			if extra := len(ms) - len(names); extra > 0 {
				text += fmt.Sprintf(" and %d more", extra)
			}
			return Diagnosis{Text: text + "."}, true
		}
	}

	// Beyond this point we would only be reading the screen back to them,
	// which helps only if the screen has scrolled.
	if strings.Count(output, "\n") < 12 {
		return Diagnosis{}, false
	}
	if m := pyError.FindAllStringSubmatch(output, -1); len(m) > 0 {
		last := m[len(m)-1]
		text := fmt.Sprintf("%s: %s", last[1], clip(last[2], 90))
		if w := pyWhere.FindAllStringSubmatch(output, -1); len(w) > 0 {
			at := w[len(w)-1]
			text += fmt.Sprintf(" (%s:%s)", shortPath(at[1]), at[2])
		}
		return Diagnosis{Text: text}, true
	}
	if m := located.FindStringSubmatch(output); m != nil {
		n := len(located.FindAllStringIndex(output, -1))
		text := fmt.Sprintf("First error is at %s:%s: %s", shortPath(m[1]), m[2], clip(strings.TrimSpace(m[3]), 90))
		if n > 1 {
			text += fmt.Sprintf(" (%d in all; later ones often follow from the first.)", n)
		}
		return Diagnosis{Text: text}, true
	}
	if m := announced.FindStringSubmatch(output); m != nil {
		return Diagnosis{Text: "The first error was: " + clip(strings.TrimSpace(m[1]), 110)}, true
	}
	return Diagnosis{}, false
}

// tidy cleans up a message assembled from optional capture groups.
func tidy(s string) string {
	s = strings.Join(strings.Fields(s), " ")
	s = strings.ReplaceAll(s, " .", ".")
	return strings.ReplaceAll(s, "``", "it")
}

// shortPath keeps the last two elements of a path.
func shortPath(p string) string {
	parts := strings.Split(p, "/")
	if len(parts) > 2 {
		parts = parts[len(parts)-2:]
	}
	return strings.Join(parts, "/")
}

// tail returns the last n lines of text.
func tail(text string, n int) string {
	lines := strings.Split(strings.TrimSpace(text), "\n")
	if len(lines) > n {
		lines = lines[len(lines)-n:]
	}
	return strings.Join(lines, "\n")
}
