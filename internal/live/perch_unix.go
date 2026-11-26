//go:build unix

package live

import (
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/MagikIO/familiar-says/internal/ipc"
)

// ParseSignal resolves the name of a signal the perch may use to nudge the
// shell. Only the two user-defined signals make sense for that.
func ParseSignal(name string) (syscall.Signal, error) {
	switch strings.TrimPrefix(strings.ToUpper(name), "SIG") {
	case "USR1":
		return syscall.SIGUSR1, nil
	case "USR2":
		return syscall.SIGUSR2, nil
	}
	return 0, fmt.Errorf("unsupported signal %q (use USR1 or USR2)", name)
}

// PerchOptions configures the headless prompt companion.
type PerchOptions struct {
	World     *World
	Server    *ipc.Server
	Hints     *Hints
	FramePath string         // where rendered frames are written
	ShellPID  int            // the shell to nudge when a new frame is ready
	Signal    syscall.Signal // the nudge; the shell hook must be trapping it
	Cols      int            // the shell's terminal width
	Format    string         // "" for raw escape codes (fish), "zsh" for zsh prompt syntax
}

// RunPerch runs the familiar with no terminal of its own. Each time the
// picture changes it writes the frame to a file and signals the shell, whose
// prompt function prints the frame above the prompt. The shell's own redraw
// machinery does all the cursor work, which is what makes this safe to run
// underneath an interactive session.
//
// It returns when the shell goes away or a quit event arrives.
func RunPerch(opt PerchOptions) error {
	w := opt.World
	w.Resize(perchWidth(opt.Cols), w.WantHeight())

	results := make(chan any, 16)
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGTERM, syscall.SIGINT, syscall.SIGHUP)

	defer os.Remove(opt.FramePath)
	defer opt.Hints.Close()

	const (
		fast = 110 * time.Millisecond // while something is moving
		slow = 400 * time.Millisecond // while nothing is
	)
	var (
		last    string
		busy    bool  // a command is running; the prompt is not on screen
		done    int64 // highest command sequence number seen to finish
		nudge   bool  // something happened; don't wait out the idle timer
		prev    = time.Now()
		timer   = time.NewTimer(0)
		publish = func() {
			frame := strings.Join(w.Render().Lines(), "\n")
			if frame == last {
				return
			}
			last = frame
			if opt.Format == "zsh" {
				frame = zshPrompt(frame)
			}
			if writeAtomic(opt.FramePath, []byte(frame)) == nil && !busy {
				_ = syscall.Kill(opt.ShellPID, opt.Signal)
			}
		}
	)
	defer timer.Stop()

	for {
		select {
		case <-stop:
			return nil

		case v := <-results:
			w.Deliver(v)
			nudge = true

		case ev := <-opt.Server.Events:
			nudge = true
			switch ev.Type {
			case ipc.TypeQuit:
				return nil
			case ipc.TypeResize:
				w.Resize(perchWidth(ev.Cols), w.WantHeight())
			case ipc.TypePreexec:
				// The prompt is about to scroll away. Hold the world still
				// so that whatever the familiar makes of this command plays
				// out where the user can see it: at the next prompt.
				//
				// Hooks send events without waiting, so a quick command's
				// preexec can arrive after its completion; the sequence
				// number tells us that one is already over.
				busy = ev.Seq == 0 || ev.Seq > done
			case ipc.TypeCommand:
				busy = false
				done = max(done, ev.Seq)
				prev = time.Now()
				if ev.Cols > 0 {
					w.Resize(perchWidth(ev.Cols), w.WantHeight())
				}
				w.Handle(ev)
			default:
				w.Handle(ev)
			}
			// A change of shape changes how tall the perch needs to be.
			if w.H != w.WantHeight() {
				w.Resize(w.W, w.WantHeight())
			}

		case now := <-timer.C:
			// A shell that has exited leaves us re-parented.
			if os.Getppid() != opt.ShellPID {
				return nil
			}
			dt := now.Sub(prev)
			prev = now
			if dt > 500*time.Millisecond {
				dt = 500 * time.Millisecond
			}
			if !busy {
				w.Step(dt)
				publish()
			}
			if w.Active() && !busy {
				timer.Reset(fast)
			} else {
				timer.Reset(slow)
			}
		}

		for _, task := range w.Tasks() {
			task := task
			go func() { results <- task() }()
		}
		opt.Hints.Update(w.Suggestion())
		if nudge {
			nudge = false
			timer.Reset(15 * time.Millisecond)
		}
	}
}
