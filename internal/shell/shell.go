// Package shell generates the shell integration scripts: the hooks that tell
// a familiar what you are running, the key that accepts its suggestions and,
// for fish, the prompt wrapper it perches on.
package shell

import (
	"bytes"
	_ "embed"
	"fmt"
	"strings"
	"text/template"
)

//go:embed hook.fish.tmpl
var fishTmpl string

//go:embed hook.bash.tmpl
var bashTmpl string

//go:embed hook.zsh.tmpl
var zshTmpl string

// Options fills in a hook script.
type Options struct {
	Bin       string   // absolute path of the familiar-says binary
	Dir       string   // runtime directory holding sockets and hint files
	Perch     bool     // also put the familiar on the prompt (fish and zsh)
	PerchArgs []string // extra arguments for the perch process
	Signal    string   // signal the perch uses to request a repaint: USR1 or USR2
}

// Supported lists the shells Script can generate hooks for.
var Supported = []string{"fish", "bash", "zsh"}

// Script renders the integration script for the named shell.
func Script(name string, opt Options) (string, error) {
	var src string
	switch name {
	case "fish":
		src = fishTmpl
	case "bash":
		src = bashTmpl
	case "zsh":
		src = zshTmpl
	default:
		return "", fmt.Errorf("unsupported shell %q (supported: %s)", name, strings.Join(Supported, ", "))
	}
	if opt.Perch && name == "bash" {
		return "", fmt.Errorf("--perch needs fish or zsh: bash cannot redraw its prompt while you type. In bash, run `familiar-says live` in a split instead")
	}
	if opt.Signal == "" {
		opt.Signal = "USR2"
	}
	quote := posixQuote
	if name == "fish" {
		quote = fishQuote
	}
	tmpl, err := template.New(name).Funcs(template.FuncMap{
		"q": quote,
		"args": func(args []string) string {
			quoted := make([]string, len(args))
			for i, a := range args {
				quoted[i] = quote(a)
			}
			return strings.Join(quoted, " ")
		},
	}).Parse(src)
	if err != nil {
		return "", err
	}
	var out bytes.Buffer
	if err := tmpl.Execute(&out, opt); err != nil {
		return "", err
	}
	return out.String(), nil
}

// fishQuote quotes s as a single fish word.
func fishQuote(s string) string {
	return "'" + strings.NewReplacer(`\`, `\\`, `'`, `\'`).Replace(s) + "'"
}

// posixQuote quotes s as a single bash/zsh word.
func posixQuote(s string) string {
	return "'" + strings.ReplaceAll(s, `'`, `'\''`) + "'"
}
