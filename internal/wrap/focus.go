package wrap

import (
	"bytes"
	"os"
	"sync/atomic"
)

const (
	focusOn  = "\x1b[?1004h"
	focusOff = "\x1b[?1004l"
)

var (
	focusIn  = []byte("\x1b[I")
	focusOut = []byte("\x1b[O")
)

// focusTracker follows the terminal's focus reports and records the current
// state in a file for the familiar to read.
type focusTracker struct {
	path      string
	childAsks atomic.Bool // the wrapped program enabled focus reporting itself
}

func (f *focusTracker) set(focused bool) {
	state := "0\n"
	if focused {
		state = "1\n"
	}
	if f.path != "" {
		_ = os.WriteFile(f.path, []byte(state), 0o600)
	}
}

// input processes bytes typed at (or reported by) the terminal and returns
// what should be passed on to the wrapped program.
//
// Only whole reports within one read are recognised. Terminals send them in
// one piece, and waiting on a lone ESC for more would delay the Escape key.
func (f *focusTracker) input(p []byte) []byte {
	in, out := bytes.LastIndex(p, focusIn), bytes.LastIndex(p, focusOut)
	if in < 0 && out < 0 {
		return p
	}
	f.set(in > out)
	if f.childAsks.Load() {
		return p
	}
	p = bytes.ReplaceAll(p, focusIn, nil)
	return bytes.ReplaceAll(p, focusOut, nil)
}

// output watches what the wrapped program writes, to learn whether it wants
// focus reports for itself. It reports true when the program has just turned
// reporting off, which the caller must undo for its own sake.
func (f *focusTracker) output(p []byte) (turnedOff bool) {
	on, off := bytes.LastIndex(p, []byte(focusOn)), bytes.LastIndex(p, []byte(focusOff))
	if on < 0 && off < 0 {
		return false
	}
	f.childAsks.Store(on > off)
	return off > on
}
