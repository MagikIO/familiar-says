package shell

import (
	"os/exec"
	"strings"
	"testing"
)

var opts = Options{Bin: "/opt/it's here/familiar-says", Dir: "/run/user/1000/familiar-says"}

func TestScriptsRenderForEveryShell(t *testing.T) {
	for _, name := range Supported {
		out, err := Script(name, opts)
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		for _, want := range []string{"preexec", "--exit", "--duration", "--seq", "live.hint", "__familiar_accept"} {
			if !strings.Contains(out, want) {
				t.Errorf("%s hook should contain %q", name, want)
			}
		}
		if strings.Contains(out, "{{") || strings.Contains(out, "<no value>") {
			t.Errorf("%s hook has unrendered template bits", name)
		}
	}
}

func TestBinaryPathIsQuoted(t *testing.T) {
	fish, _ := Script("fish", opts)
	if !strings.Contains(fish, `'/opt/it\'s here/familiar-says'`) {
		t.Error("fish hook should quote the binary path fish-style")
	}
	bash, _ := Script("bash", opts)
	if !strings.Contains(bash, `'/opt/it'\''s here/familiar-says'`) {
		t.Error("bash hook should quote the binary path POSIX-style")
	}
}

func TestPerchIsFishOnly(t *testing.T) {
	perch := opts
	perch.Perch = true
	perch.PerchArgs = []string{"--character", "owl"}

	out, err := Script("fish", perch)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"function fish_prompt", "--on-signal SIGUSR2", "fish_transient_prompt", "perch --pid $fish_pid", "--character' 'owl'", "shell-$fish_pid.sock"} {
		if !strings.Contains(out, want) {
			t.Errorf("perch hook should contain %q", want)
		}
	}
	plain, _ := Script("fish", opts)
	if strings.Contains(plain, "fish_prompt") || strings.Contains(plain, "perch --pid") {
		t.Error("without --perch the hook must leave the prompt alone")
	}

	if _, err := Script("bash", perch); err == nil {
		t.Error("bash cannot redraw its prompt asynchronously; --perch should be refused")
	}
	zsh, err := Script("zsh", perch)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"TRAPUSR2()", "--format zsh", "zle reset-prompt", "line-finish", "prompt_subst", "shell-$$.sock"} {
		if !strings.Contains(zsh, want) {
			t.Errorf("zsh perch hook should contain %q", want)
		}
	}
	if plain, _ := Script("zsh", opts); strings.Contains(plain, "TRAPUSR2") || strings.Contains(plain, "perch --pid") {
		t.Error("without --perch the zsh hook must leave the prompt alone")
	}
	if _, err := Script("powershell", opts); err == nil {
		t.Error("unknown shells should be refused")
	}
}

// The generated scripts are loaded into people's interactive shells, so at
// the very least they must parse.
func TestScriptsParse(t *testing.T) {
	perch := opts
	perch.Perch = true
	cases := []struct {
		shell string
		opt   Options
		check []string
	}{
		{"fish", opts, []string{"fish", "--no-config", "--no-execute"}},
		{"fish", perch, []string{"fish", "--no-config", "--no-execute"}},
		{"bash", opts, []string{"bash", "-n"}},
		{"zsh", opts, []string{"zsh", "-n"}},
	}
	for _, c := range cases {
		if _, err := exec.LookPath(c.check[0]); err != nil {
			t.Logf("%s not installed; skipping its syntax check", c.check[0])
			continue
		}
		script, err := Script(c.shell, c.opt)
		if err != nil {
			t.Fatal(err)
		}
		cmd := exec.Command(c.check[0], c.check[1:]...)
		cmd.Stdin = strings.NewReader(script)
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Errorf("%s (perch=%v) hook does not parse: %v\n%s", c.shell, c.opt.Perch, err, out)
		}
	}
}
