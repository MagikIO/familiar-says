//go:build !unix

package live

import (
	"errors"
	"syscall"

	"github.com/MagikIO/familiar-says/internal/ipc"
)

// PerchOptions configures the headless prompt companion.
type PerchOptions struct {
	World     *World
	Server    *ipc.Server
	Hints     *Hints
	FramePath string
	ShellPID  int
	Signal    syscall.Signal
	Cols      int
	Format    string
}

var errNoPerch = errors.New("the prompt perch needs a Unix shell; use `familiar-says live` instead")

// ParseSignal is unavailable on this platform.
func ParseSignal(string) (syscall.Signal, error) { return 0, errNoPerch }

// RunPerch is unavailable on this platform.
func RunPerch(PerchOptions) error { return errNoPerch }
