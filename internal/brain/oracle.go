package brain

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"
)

// Oracle answers free-form questions by running an external command — any
// program that reads a prompt on stdin and writes an answer to stdout, such as
// `claude -p` or `llm`. Nothing is ever sent to it unless the user asks.
type Oracle struct {
	Command []string
	Timeout time.Duration
}

// Configured reports whether an oracle command has been set.
func (o Oracle) Configured() bool { return len(o.Command) > 0 }

// Answer is an oracle's reply to a question.
type Answer struct {
	Question string
	Text     string
	Err      error
}

// Ask runs the oracle. It blocks until the command exits or times out.
//
// dir is the directory the user is working in. The oracle is started there,
// so one that can read files (an assistant CLI given read-only tools) sees
// the project being asked about. An empty dir means somewhere neutral.
func (o Oracle) Ask(question, persona, background, dir string) Answer {
	ans := Answer{Question: question}
	if !o.Configured() {
		ans.Err = errors.New("no oracle configured")
		return ans
	}
	timeout := o.Timeout
	if timeout <= 0 {
		timeout = 60 * time.Second
	}
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	var prompt strings.Builder
	fmt.Fprintf(&prompt, "You are %s, a small familiar who lives in the user's terminal and helps them.\n", persona)
	prompt.WriteString("Answer the question below in plain text, at most three short sentences (under 400 characters), no markdown or code fences. ")
	prompt.WriteString("If a command would help, give the exact command inline.\n")
	if background != "" {
		prompt.WriteString("\nContext:\n" + background + "\n")
	}
	prompt.WriteString("\nQuestion: " + question + "\n")

	cmd := exec.CommandContext(ctx, o.Command[0], o.Command[1:]...)
	cmd.Stdin = strings.NewReader(prompt.String())
	cmd.Dir = os.TempDir()
	if info, err := os.Stat(dir); dir != "" && err == nil && info.IsDir() {
		cmd.Dir = dir
	}
	var out, errOut bytes.Buffer
	cmd.Stdout, cmd.Stderr = &out, &errOut
	if err := cmd.Run(); err != nil {
		if ctx.Err() != nil {
			ans.Err = errors.New("the oracle took too long")
		} else if msg := strings.TrimSpace(errOut.String()); msg != "" {
			ans.Err = errors.New(clip(msg, 120))
		} else {
			ans.Err = err
		}
		return ans
	}
	ans.Text = clipSentences(strings.Join(strings.Fields(out.String()), " "), 1200)
	if ans.Text == "" {
		ans.Err = errors.New("the oracle had nothing to say")
	}
	return ans
}

// clipSentences shortens text to at most n runes without stopping in the
// middle of a sentence where it can be helped.
func clipSentences(text string, n int) string {
	r := []rune(text)
	if len(r) <= n {
		return text
	}
	cut := string(r[:n])
	if i := strings.LastIndexAny(cut, ".!?"); i > n/2 {
		return cut[:i+1]
	}
	return strings.TrimSpace(cut) + "…"
}
