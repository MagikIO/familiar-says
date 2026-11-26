//go:build unix

package wrap

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/creack/pty"
	"golang.org/x/term"
)

// Environment variables set for the wrapped shell.
const (
	EnvWrapped    = "FAMILIAR_SAYS_WRAPPED"    // "1" inside the wrapper
	EnvTranscript = "FAMILIAR_SAYS_TRANSCRIPT" // file the last command's output is written to
	EnvWrapPID    = "FAMILIAR_SAYS_WRAP_PID"   // the wrapper's process id
)

// Run starts argv inside a pseudo-terminal, relays everything between it and
// the real terminal, and writes each command's output to a transcript file
// named in the child's environment. It returns the child's exit status.
//
// If the wrapper cannot be set up, the caller should simply run the shell
// directly: a familiar that cannot listen is no reason to leave someone
// without a shell.
func Run(argv []string, dir string) (int, error) {
	if os.Getenv(EnvWrapped) == "1" {
		return 0, errors.New("this shell is already wrapped")
	}
	if !term.IsTerminal(int(os.Stdin.Fd())) || !term.IsTerminal(int(os.Stdout.Fd())) {
		return 0, errors.New("wrap needs a terminal")
	}
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return 0, err
	}
	transcript := filepath.Join(dir, fmt.Sprintf("wrap-%d.out", os.Getpid()))
	defer os.Remove(transcript)

	cmd := exec.Command(argv[0], argv[1:]...)
	// The shell hooks only mark output when their parent is this wrapper, so
	// that these variables leaking into some other shell (one started inside
	// tmux from here, say) cannot make it print markers nobody will remove.
	cmd.Env = append(os.Environ(), EnvWrapped+"=1", EnvTranscript+"="+transcript, fmt.Sprintf("%s=%d", EnvWrapPID, os.Getpid()))
	ptmx, err := pty.Start(cmd)
	if err != nil {
		return 0, err
	}
	defer ptmx.Close()

	// Keep the inner terminal the same size as the outer one.
	_ = pty.InheritSize(os.Stdin, ptmx)
	sigs := make(chan os.Signal, 4)
	signal.Notify(sigs, syscall.SIGWINCH, syscall.SIGTERM, syscall.SIGHUP)
	defer signal.Stop(sigs)
	go func() {
		for sig := range sigs {
			if sig == syscall.SIGWINCH {
				_ = pty.InheritSize(os.Stdin, ptmx)
			} else if cmd.Process != nil {
				_ = cmd.Process.Signal(syscall.SIGHUP) // our terminal is going away; so is theirs
			}
		}
	}()

	state, err := term.MakeRaw(int(os.Stdin.Fd()))
	if err != nil {
		_ = cmd.Process.Kill()
		return 0, err
	}
	defer term.Restore(int(os.Stdin.Fd()), state)

	// Ask the terminal to say when its window gains or loses focus, so the
	// familiar knows whether you are looking when a long command finishes.
	// Those reports arrive as input; they are ours, so they are taken out of
	// the stream unless the program inside asked for them too.
	focus := &focusTracker{path: transcript + ".focus"}
	focus.set(true)
	defer os.Remove(focus.path)
	_, _ = os.Stdout.WriteString(focusOn)
	defer os.Stdout.WriteString(focusOff)

	go func() {
		buf := make([]byte, 4096)
		for {
			n, err := os.Stdin.Read(buf)
			if n > 0 {
				if _, werr := ptmx.Write(focus.input(buf[:n])); werr != nil {
					return
				}
			}
			if err != nil {
				return
			}
		}
	}()

	filter := &Filter{OnCommand: func(seq string, raw []byte) {
		// The sequence number goes on the first line so a reader can tell
		// whether this is the command it is asking about.
		tmp := transcript + ".tmp"
		if os.WriteFile(tmp, append([]byte(seq+"\n"), Clean(raw)...), 0o600) == nil {
			_ = os.Rename(tmp, transcript)
		}
	}}
	buf := make([]byte, 32*1024)
	for {
		n, err := ptmx.Read(buf)
		if n > 0 {
			_, _ = os.Stdout.Write(filter.Write(buf[:n]))
			if focus.output(buf[:n]) {
				// The program inside turned focus reporting off for itself;
				// that reached the real terminal, so turn ours back on.
				_, _ = os.Stdout.WriteString(focusOn)
			}
		}
		if err != nil {
			break // EIO once the child side closes
		}
	}
	_, _ = os.Stdout.Write(filter.Flush())

	err = cmd.Wait()
	var exit *exec.ExitError
	if errors.As(err, &exit) {
		return exit.ExitCode(), nil
	}
	return 0, err
}
