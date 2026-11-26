// Package wrap runs a shell inside a pseudo-terminal that the familiar can
// listen in on. Everything passes straight through in both directions, so the
// shell and the terminal behave exactly as before; on the way past, the output
// of the most recent command is kept so the familiar can read the error you
// just got instead of guessing from the exit status.
package wrap

import (
	"bytes"
	"regexp"
	"strings"
)

// The shell hooks bracket each command's output with these private OSC
// sequences (only when running under the wrapper, which removes them again so
// the terminal never sees them):
//
//	ESC ] 7797 ; S ; <seq> ESC \    output of command <seq> starts here
//	ESC ] 7797 ; E ; <seq> ESC \    ... and ends here
var markerStart = []byte("\x1b]7797;")

const (
	maxMarker = 48        // longest marker we will wait for before giving up on it
	keepHead  = 16 * 1024 // bytes of a command's output kept from its start
	keepTail  = 24 * 1024 // ... and from its end
)

// Marker is a parsed start or end marker.
type Marker struct {
	End bool
	Seq string
}

// Filter strips markers out of a byte stream and collects the output that
// appears between a start marker and the next end marker.
type Filter struct {
	// OnCommand is called with each finished command's sequence number and
	// raw output.
	OnCommand func(seq string, output []byte)

	pending   []byte // bytes held back because they might begin a marker
	recording bool
	head      []byte
	tail      []byte
	dropped   bool
}

// Write passes p through the filter and returns the bytes that should go on
// to the terminal.
func (f *Filter) Write(p []byte) []byte {
	data := p
	if len(f.pending) > 0 {
		data = append(f.pending, p...)
		f.pending = nil
	}
	var out []byte
	for len(data) > 0 {
		i := bytes.IndexByte(data, 0x1b)
		if i < 0 {
			out = f.pass(out, data)
			break
		}
		out = f.pass(out, data[:i])
		data = data[i:]

		// Could this ESC begin one of our markers?
		n := min(len(data), len(markerStart))
		if !bytes.Equal(data[:n], markerStart[:n]) {
			out = f.pass(out, data[:1])
			data = data[1:]
			continue
		}
		end, width := terminator(data)
		if end < 0 {
			if len(data) > maxMarker {
				// Not a marker after all; let it through.
				out = f.pass(out, data[:1])
				data = data[1:]
				continue
			}
			f.pending = append([]byte(nil), data...) // wait for the rest
			break
		}
		f.marker(string(data[len(markerStart):end]))
		data = data[end+width:]
	}
	return out
}

// terminator finds the end of an OSC sequence: BEL, or ESC backslash.
func terminator(data []byte) (index, width int) {
	for i := 1; i < len(data) && i <= maxMarker; i++ {
		switch {
		case data[i] == 0x07:
			return i, 1
		case data[i] == 0x1b && i+1 < len(data) && data[i+1] == '\\':
			return i, 2
		}
	}
	return -1, 0
}

func (f *Filter) pass(out, data []byte) []byte {
	if len(data) == 0 {
		return out
	}
	if f.recording {
		f.record(data)
	}
	return append(out, data...)
}

func (f *Filter) record(data []byte) {
	if room := keepHead - len(f.head); room > 0 {
		n := min(room, len(data))
		f.head = append(f.head, data[:n]...)
		data = data[n:]
	}
	if len(data) == 0 {
		return
	}
	f.tail = append(f.tail, data...)
	if over := len(f.tail) - keepTail; over > 0 {
		f.tail = f.tail[over:]
		f.dropped = true
	}
}

func (f *Filter) marker(body string) {
	kind, seq, _ := strings.Cut(body, ";")
	switch kind {
	case "S":
		f.recording = true
		f.head, f.tail, f.dropped = f.head[:0], f.tail[:0], false
	case "E":
		if !f.recording {
			return
		}
		f.recording = false
		if f.OnCommand != nil {
			raw := append([]byte(nil), f.head...)
			if f.dropped {
				raw = append(raw, "\n[...]\n"...)
			}
			raw = append(raw, f.tail...)
			f.OnCommand(seq, raw)
		}
	}
}

// Flush returns anything still held back, for when the stream ends.
func (f *Filter) Flush() []byte {
	p := f.pending
	f.pending = nil
	return p
}

var (
	// CSI and OSC sequences, and the odd two-byte escape.
	ansi = regexp.MustCompile(`\x1b\[[0-9;:?<=>]*[ -/]*[@-~]|\x1b\][^\x07\x1b]*(?:\x07|\x1b\\)|\x1b[()][0-9A-B]|\x1b[=>78DEHMNOc]`)
)

// Clean turns raw terminal output into the text a person would have read:
// escape sequences removed, progress bars collapsed to their final state,
// backspaces applied.
func Clean(raw []byte) string {
	text := ansi.ReplaceAllString(string(raw), "")
	var out strings.Builder
	for _, line := range strings.Split(text, "\n") {
		line = strings.TrimRight(line, "\r")
		// A carriage return restarts the line; what is visible is what was
		// written last.
		if i := strings.LastIndexByte(line, '\r'); i >= 0 {
			line = line[i+1:]
		}
		var b []rune
		for _, r := range line {
			switch {
			case r == '\b':
				if len(b) > 0 {
					b = b[:len(b)-1]
				}
			case r == '\t':
				b = append(b, ' ', ' ', ' ', ' ')
			case r < 0x20 || r == 0x7f:
			default:
				b = append(b, r)
			}
		}
		out.WriteString(strings.TrimRight(string(b), " "))
		out.WriteByte('\n')
	}
	return strings.Trim(out.String(), "\n")
}
