package brain

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"
)

// distance is the optimal-string-alignment edit distance: insertions,
// deletions, substitutions and swaps of adjacent characters each cost one.
// That last rule is what makes "gti" one step from "git".
func distance(a, b string) int {
	ra, rb := []rune(a), []rune(b)
	prev2 := make([]int, len(rb)+1)
	prev := make([]int, len(rb)+1)
	cur := make([]int, len(rb)+1)
	for j := range prev {
		prev[j] = j
	}
	for i := 1; i <= len(ra); i++ {
		cur[0] = i
		for j := 1; j <= len(rb); j++ {
			cost := 1
			if ra[i-1] == rb[j-1] {
				cost = 0
			}
			cur[j] = min(prev[j]+1, cur[j-1]+1, prev[j-1]+cost)
			if i > 1 && j > 1 && ra[i-1] == rb[j-2] && ra[i-2] == rb[j-1] {
				cur[j] = min(cur[j], prev2[j-2]+1)
			}
		}
		prev2, prev, cur = prev, cur, prev2
	}
	return prev[len(rb)]
}

// nearest returns the candidate closest to word, provided it is close enough
// to be a plausible typo. Ties go to the earlier candidate, so callers should
// list likelier answers first.
func nearest(word string, candidates []string) (string, bool) {
	limit := 2
	if len(word) <= 4 {
		limit = 1
	}
	best, bestD := "", limit+1
	for _, c := range candidates {
		if c == word {
			return "", false
		}
		if d := len(c) - len(word); d > limit || d < -limit {
			continue
		}
		if d := distance(word, c); d < bestD {
			best, bestD = c, d
		}
	}
	return best, best != ""
}

// common lists everyday commands, likeliest first, so that a typo resolves to
// the command people actually mean rather than to an obscure binary that
// happens to be one keystroke away.
var common = []string{
	"git", "ls", "cd", "cat", "grep", "make", "go", "npm", "node", "python", "python3", "pip",
	"docker", "ssh", "curl", "vim", "nvim", "code", "sudo", "cargo", "rustc", "kubectl", "pnpm",
	"yarn", "bun", "mkdir", "touch", "mv", "cp", "rm", "echo", "less", "tail", "head", "find",
	"sed", "awk", "tar", "ps", "kill", "top", "htop", "man", "which", "clear", "history", "wget",
	"chmod", "chown", "diff", "sort", "uniq", "wc", "xargs", "tmux", "fish", "bash", "zsh", "gh",
	"terraform", "ansible", "systemctl", "journalctl", "apt", "brew", "dnf", "pacman", "jq", "rg", "fd",
}

// subcommands lists the verbs of multi-command tools, likeliest first.
var subcommands = map[string][]string{
	"git": {"status", "commit", "push", "pull", "add", "checkout", "switch", "branch", "diff", "log",
		"merge", "rebase", "stash", "fetch", "clone", "reset", "restore", "show", "tag", "remote",
		"cherry-pick", "revert", "init", "bisect", "blame", "clean", "config", "grep", "mv", "rm",
		"worktree", "reflog", "submodule", "describe", "apply", "am"},
	"go": {"build", "test", "run", "mod", "get", "install", "fmt", "vet", "generate", "work",
		"clean", "doc", "env", "list", "tool", "version"},
	"npm": {"install", "run", "test", "start", "ci", "init", "publish", "update", "uninstall",
		"audit", "link", "outdated", "exec", "version", "pack", "login"},
	"pnpm": {"install", "add", "run", "test", "dev", "build", "remove", "update", "exec", "dlx", "store"},
	"yarn": {"install", "add", "run", "test", "dev", "build", "remove", "upgrade", "start"},
	"cargo": {"build", "run", "test", "check", "clippy", "fmt", "add", "update", "install", "doc",
		"bench", "clean", "new", "init", "publish"},
	"docker": {"run", "build", "ps", "images", "pull", "push", "exec", "logs", "compose", "stop",
		"start", "rm", "rmi", "inspect", "network", "volume", "system", "tag", "login", "restart"},
	"kubectl": {"get", "describe", "apply", "delete", "logs", "exec", "create", "edit", "rollout",
		"scale", "config", "port-forward", "top", "explain", "patch"},
	"brew":      {"install", "update", "upgrade", "uninstall", "search", "list", "info", "doctor", "cleanup", "services"},
	"apt":       {"install", "update", "upgrade", "remove", "search", "show", "autoremove", "purge", "list"},
	"systemctl": {"status", "start", "stop", "restart", "enable", "disable", "reload", "daemon-reload", "list-units"},
	"gh":        {"pr", "issue", "repo", "run", "workflow", "auth", "api", "release", "gist", "browse"},
}

// wrappers are prefixes that run another command and can be looked through.
var wrappers = map[string]bool{"sudo": true, "command": true, "time": true, "nohup": true, "env": true, "builtin": true, "exec": true, "doas": true, "nice": true}

var envAssign = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*=`)

// parsed is a command line reduced to what the rules care about.
type parsed struct {
	Line  string
	Words []string // program and arguments, wrappers and env assignments removed
	Prog  string   // base name of the program
	Sub   string   // first non-flag argument
	Sudo  bool
}

func parse(line string) parsed {
	p := parsed{Line: strings.TrimSpace(line)}
	words := strings.Fields(p.Line)
	for len(words) > 0 {
		w := words[0]
		switch {
		case envAssign.MatchString(w):
		case wrappers[w]:
			if w == "sudo" || w == "doas" {
				p.Sudo = true
			}
		default:
			goto done
		}
		words = words[1:]
		for len(words) > 0 && strings.HasPrefix(words[0], "-") && p.Sudo {
			words = words[1:] // sudo's own flags
		}
	}
done:
	p.Words = words
	if len(words) > 0 {
		p.Prog = filepath.Base(words[0])
		for _, w := range words[1:] {
			if !strings.HasPrefix(w, "-") {
				p.Sub = w
				break
			}
		}
	}
	return p
}

// IsSelf reports whether a command line is the user talking to the familiar
// (`familiar pet`, `familiar-says tell ...`). Those are conversation, not
// work, and the familiar should not comment on them as if they were.
func IsSelf(line string) bool {
	p := parse(line)
	return p.Prog == "familiar" || p.Prog == "familiar-says"
}

// family groups invocations that are "the same thing" for streak counting:
// `go test ./a` and `go test ./b` are one family, `go build` is another.
func (p parsed) family() string {
	if _, multi := subcommands[p.Prog]; multi && p.Sub != "" {
		return p.Prog + " " + p.Sub
	}
	if p.Prog == "make" || p.Prog == "just" || p.Prog == "task" {
		return p.Prog + " " + p.Sub
	}
	return p.Prog
}

func (p parsed) has(flags ...string) bool {
	for _, w := range p.Words {
		for _, f := range flags {
			if w == f {
				return true
			}
		}
	}
	return false
}

var (
	testRe  = regexp.MustCompile(`(^|\s)(go test|cargo test|cargo nextest|npm (run )?test|pnpm (run )?test|yarn (run )?test|bun test|pytest|py\.test|jest|vitest|rspec|phpunit|mix test|make (test|check)|just test|tox|ctest|gradle test|mvn test|dotnet test|deno test|playwright test)(\s|$)`)
	buildRe = regexp.MustCompile(`(^|\s)(go (build|install|vet)|cargo (build|check|clippy)|npm run build|pnpm (run )?build|yarn (run )?build|bun run build|make( all| build)?|tsc|gcc|g\+\+|clang|cmake --build|ninja|gradle build|mvn (package|compile)|docker build|docker compose build|zig build|just build)(\s|$)`)
)

func (p parsed) isTest() bool  { return testRe.MatchString(strings.Join(p.Words, " ")) }
func (p parsed) isBuild() bool { return buildRe.MatchString(strings.Join(p.Words, " ")) }

// benignFailure reports commands whose non-zero exit is an answer, not an
// accident: grep finding nothing, diff finding a difference, and so on.
func (p parsed) benignFailure(exit int) bool {
	if exit != 1 {
		return false
	}
	switch p.Prog {
	case "grep", "egrep", "fgrep", "rg", "ag", "ack", "diff", "cmp", "test", "[", "[[", "false",
		"which", "type", "pgrep", "pidof", "fgconsole", "expr", "string", "contains", "set", "not":
		return true
	case "git":
		return p.Sub == "diff" || p.Sub == "grep"
	case "systemctl":
		return strings.HasPrefix(p.Sub, "is-")
	}
	return false
}

// correction looks for a typo in the program name or its subcommand.
func correction(p parsed, exit int, onPath map[string]bool) (string, bool) {
	if p.Prog == "" {
		return "", false
	}
	if exit == 127 {
		if fix, ok := nearest(p.Prog, common); ok {
			return swapWord(p, 0, fix), true
		}
		names := make([]string, 0, len(onPath))
		for n := range onPath {
			names = append(names, n)
		}
		sort.Strings(names)
		if fix, ok := nearest(p.Prog, names); ok {
			return swapWord(p, 0, fix), true
		}
		return "", false
	}
	if subs, ok := subcommands[p.Prog]; ok && p.Sub != "" {
		for _, s := range subs {
			if s == p.Sub {
				return "", false
			}
		}
		if fix, ok := nearest(p.Sub, subs); ok {
			for i, w := range p.Words {
				if i > 0 && w == p.Sub {
					return swapWord(p, i, fix), true
				}
			}
		}
	}
	return "", false
}

func swapWord(p parsed, i int, fix string) string {
	words := append([]string(nil), p.Words...)
	words[i] = fix
	out := strings.Join(words, " ")
	if p.Sudo {
		out = "sudo " + out
	}
	return out
}

// pathReaders take a file or directory as their first argument and fail when
// it does not exist.
var pathReaders = map[string]bool{"cd": true, "cat": true, "bat": true, "less": true, "more": true, "head": true,
	"tail": true, "vim": true, "nvim": true, "vi": true, "nano": true, "code": true, "ls": true, "source": true,
	"python": true, "python3": true, "node": true, "bash": true, "sh": true, "open": true, "xdg-open": true,
	"wc": true, "stat": true, "file": true, "z": true, "pushd": true, "rmdir": true}

// pathCorrection suggests the neighbouring file the user probably meant when
// a command failed on a path that does not exist.
func pathCorrection(p parsed, cwd string) (string, bool) {
	if !pathReaders[p.Prog] || p.Sub == "" || cwd == "" || strings.ContainsAny(p.Sub, "*?$~{") {
		return "", false
	}
	target := p.Sub
	if !filepath.IsAbs(target) {
		target = filepath.Join(cwd, target)
	}
	if _, err := os.Stat(target); err == nil {
		return "", false
	}
	entries, err := os.ReadDir(filepath.Dir(target))
	if err != nil || len(entries) > 2000 {
		return "", false
	}
	names := make([]string, 0, len(entries))
	for _, e := range entries {
		if p.Prog == "cd" && !e.IsDir() && e.Type()&os.ModeSymlink == 0 {
			continue
		}
		names = append(names, e.Name())
	}
	fix, ok := nearest(filepath.Base(target), names)
	if !ok {
		return "", false
	}
	return filepath.Join(filepath.Dir(p.Sub), fix), true
}

// withPath rewrites the command line with its path argument replaced.
func (p parsed) withPath(fix string) string {
	words := strings.Fields(p.Line)
	for i, w := range words {
		if w == p.Sub {
			words[i] = fix
			break
		}
	}
	return strings.Join(words, " ")
}

// exitMeaning explains exit statuses that have a conventional meaning.
func exitMeaning(p parsed, exit int) (string, bool) {
	switch exit {
	case 126:
		return "Exit 126: found it, but it isn't executable.", true
	case 124:
		return "Exit 124: that timed out.", true
	case 137:
		return "Exit 137 means SIGKILL. If nobody killed it, the OOM killer did.", true
	case 139:
		return "Segfault (exit 139). `coredumpctl` may have kept the body.", true
	case 143:
		return "Exit 143: something sent it SIGTERM.", true
	}
	switch p.Prog {
	case "curl":
		switch exit {
		case 6:
			return "curl couldn't resolve that host. Typo in the URL, or DNS?", true
		case 7:
			return "curl: connection refused. Is the server actually listening?", true
		case 22:
			return "curl got an HTTP error (-f). Drop -f or add -i to see the response.", true
		case 28:
			return "curl timed out. `--max-time` or `-v` will tell you more.", true
		case 35, 60:
			return "curl hit a TLS problem. `-v` shows the handshake.", true
		}
	case "ssh", "scp", "sftp":
		if exit == 255 {
			return "ssh never got a session. `ssh -v` shows where it stalls.", true
		}
	case "git":
		switch p.Sub {
		case "push":
			return "Push bounced. If the remote moved on: `git pull --rebase`, then push again.", true
		case "commit":
			return "No commit made. Nothing staged? `git add -p` lets you pick hunks.", true
		case "pull", "merge", "rebase":
			return "That stopped partway. `git status` shows what needs resolving.", true
		case "checkout", "switch":
			return "Couldn't switch. Uncommitted changes in the way? `git stash` parks them.", true
		}
	case "npm", "pnpm", "yarn":
		if p.Sub == "install" || p.Sub == "ci" || p.Sub == "add" {
			return "Install failed. The first error in the log is usually the real one.", true
		}
	case "docker":
		if exit == 125 {
			return "Exit 125 is docker itself failing, not your container. Is the daemon up?", true
		}
	}
	return "", false
}

// danger recognises command lines that deserve a nervous look, and returns
// advice when there is a safer way to do the same thing.
func danger(p parsed) (advice string, risky bool) {
	line := " " + strings.Join(p.Words, " ") + " "
	switch {
	case p.Prog == "git" && p.Sub == "push" && (p.has("--force", "-f") || strings.Contains(line, " +")):
		return "Force push! `--force-with-lease` only overwrites what you've already seen.", true
	case p.Prog == "git" && p.Sub == "reset" && p.has("--hard"):
		return "", true
	case p.Prog == "git" && p.Sub == "clean" && !p.has("-n", "--dry-run"):
		return "`git clean -n` first shows what would be deleted.", true
	case p.Prog == "rm" && rmRecursiveForce(p):
		return "", true
	case p.Prog == "chmod" && p.has("-R") && strings.Contains(line, " 777 "):
		return "777 on everything? Something narrower usually does the job.", true
	case p.Prog == "dd" && strings.Contains(line, " of=/dev/"):
		return "", true
	case p.Prog == "terraform" && (p.Sub == "destroy" || (p.Sub == "apply" && p.has("-auto-approve"))):
		return "", true
	case p.Prog == "kubectl" && p.Sub == "delete":
		return "", true
	case p.Prog == "docker" && p.Sub == "system" && strings.Contains(line, " prune "):
		return "", true
	case dropRe.MatchString(line):
		return "", true
	}
	return "", false
}

var dropRe = regexp.MustCompile(`(?i)\bdrop (table|database)\b`)

func rmRecursiveForce(p parsed) bool {
	var r, f bool
	for _, w := range p.Words[1:] {
		switch {
		case w == "--recursive":
			r = true
		case w == "--force":
			f = true
		case strings.HasPrefix(w, "-") && !strings.HasPrefix(w, "--"):
			r = r || strings.ContainsAny(w, "rR")
			f = f || strings.ContainsRune(w, 'f')
		}
	}
	return r && f
}

var simpleCommand = regexp.MustCompile(`^[\w./+-]+( [\w./:=@+,-]+)+$`)

// shorthand proposes an alias for a command the user keeps typing in full.
func shorthand(line, shell string, taken map[string]bool) (string, bool) {
	if len(line) < 9 || len(line) > 48 || !simpleCommand.MatchString(line) {
		return "", false
	}
	words := strings.Fields(line)
	if wrappers[words[0]] {
		return "", false
	}
	var name strings.Builder
	for _, w := range words {
		if strings.HasPrefix(w, "-") {
			continue // flags don't earn a letter: "docker compose up -d" is dcu
		}
		w = strings.TrimLeft(w, "./")
		if w == "" {
			continue
		}
		c := w[0]
		if (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') {
			name.WriteByte(c | 0x20)
		}
		if name.Len() == 4 {
			break
		}
	}
	short := name.String()
	if len(short) < 2 || taken[short] {
		return "", false
	}
	if shell == "fish" {
		return fmt.Sprintf("abbr -a %s %s", short, line), true
	}
	return fmt.Sprintf("alias %s='%s'", short, line), true
}

// PathIndex is the set of executable names found on $PATH.
type PathIndex map[string]bool

// ScanPath lists the executables on $PATH. It touches a few thousand directory
// entries, so run it off the render loop.
func ScanPath() PathIndex {
	idx := PathIndex{}
	for _, dir := range filepath.SplitList(os.Getenv("PATH")) {
		entries, err := os.ReadDir(dir)
		if err != nil {
			continue
		}
		for _, e := range entries {
			if !e.IsDir() {
				idx[e.Name()] = true
			}
		}
	}
	return idx
}

func clip(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return string(r[:n-1]) + "…"
}

// human formats a duration the way a person would say it.
func human(d time.Duration) string {
	d = d.Round(time.Second)
	switch {
	case d < time.Minute:
		return fmt.Sprintf("%ds", int(d.Seconds()))
	case d < time.Hour:
		return fmt.Sprintf("%dm%02ds", int(d.Minutes()), int(d.Seconds())%60)
	}
	return fmt.Sprintf("%dh%02dm", int(d.Hours()), int(d.Minutes())%60)
}
