// Package ipc carries events from the outside world (shell hooks, the
// `familiar-says tell` command, scripts) to a running familiar over a Unix
// socket. The protocol is one JSON object per line; senders never wait for a
// reply, so a hook costs a shell only the time it takes to start the binary.
package ipc

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"time"
)

// Event types.
const (
	TypePreexec = "preexec" // a command line is about to run
	TypeCommand = "command" // a command line finished
	TypeSay     = "say"     // make the familiar say Text
	TypeAsk     = "ask"     // put Text to the oracle
	TypePet     = "pet"
	TypeNap     = "nap"
	TypeWake    = "wake"
	TypeWear    = "wear"   // put on the accessory named by Text ("none" takes it off)
	TypeBecome  = "become" // turn into the character named by Text
	TypeBond    = "bond"   // say how the two of you are doing
	TypeRescan  = "rescan" // look at the current directory again (its rules may have changed)
	TypeResize  = "resize" // the host terminal is now Cols wide
	TypeQuit    = "quit"
)

// Event is one message to the familiar. Only Type is mandatory.
type Event struct {
	Type       string `json:"type"`
	Cmd        string `json:"cmd,omitempty"`
	Exit       int    `json:"exit,omitempty"`
	DurationMs int64  `json:"duration_ms,omitempty"`
	Cwd        string `json:"cwd,omitempty"`
	Shell      string `json:"shell,omitempty"`
	Cols       int    `json:"cols,omitempty"`
	Seq        int64  `json:"seq,omitempty"`        // per-shell command counter; pairs a preexec with its command
	Transcript string `json:"transcript,omitempty"` // file holding the command's output, when the shell is wrapped
	Text       string `json:"text,omitempty"`
	Mood       string `json:"mood,omitempty"`
	Style      string `json:"style,omitempty"`
}

// EnvSocket names the environment variable that overrides the socket path.
const EnvSocket = "FAMILIAR_SAYS_SOCKET"

// RuntimeDir is the per-user directory for sockets and frame files.
func RuntimeDir() string {
	if d := os.Getenv("XDG_RUNTIME_DIR"); d != "" {
		return filepath.Join(d, "familiar-says")
	}
	return filepath.Join(os.TempDir(), fmt.Sprintf("familiar-says-%d", os.Getuid()))
}

// SocketPath resolves the socket to use: $FAMILIAR_SAYS_SOCKET if set,
// otherwise the default socket of the live companion.
func SocketPath() string {
	if p := os.Getenv(EnvSocket); p != "" {
		return p
	}
	return filepath.Join(RuntimeDir(), "live.sock")
}

// ErrNotRunning means nothing is listening on the socket.
var ErrNotRunning = errors.New("no familiar is listening (start one with `familiar-says live`)")

// Send delivers one event. It fails fast with ErrNotRunning when no familiar
// is listening, which is the normal case for a hook in a shell without one.
func Send(path string, ev Event) error {
	if _, err := os.Stat(path); err != nil {
		return ErrNotRunning
	}
	conn, err := net.DialTimeout("unix", path, 300*time.Millisecond)
	if err != nil {
		return ErrNotRunning
	}
	defer conn.Close()
	_ = conn.SetWriteDeadline(time.Now().Add(500 * time.Millisecond))
	data, err := json.Marshal(ev)
	if err != nil {
		return err
	}
	_, err = conn.Write(append(data, '\n'))
	return err
}

// Server receives events on a Unix socket.
type Server struct {
	Events chan Event
	path   string
	ln     net.Listener
}

// Listen claims the socket at path. A socket left behind by a crashed process
// is replaced; one with a live listener is an error.
func Listen(path string) (*Server, error) {
	// sockaddr_un holds about a hundred bytes; the kernel's own error for an
	// overlong path ("invalid argument") explains nothing.
	if len(path) > 100 {
		return nil, fmt.Errorf("socket path is too long (%d bytes, limit 100): %s\nset XDG_RUNTIME_DIR or %s to somewhere shorter", len(path), path, EnvSocket)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return nil, err
	}
	if _, err := os.Stat(path); err == nil {
		if conn, err := net.DialTimeout("unix", path, 200*time.Millisecond); err == nil {
			conn.Close()
			return nil, fmt.Errorf("a familiar is already listening on %s", path)
		}
		_ = os.Remove(path)
	}
	ln, err := net.Listen("unix", path)
	if err != nil {
		return nil, err
	}
	_ = os.Chmod(path, 0o600)
	s := &Server{Events: make(chan Event, 64), path: path, ln: ln}
	go s.accept()
	return s, nil
}

func (s *Server) accept() {
	for {
		conn, err := s.ln.Accept()
		if err != nil {
			return
		}
		go s.read(conn)
	}
}

func (s *Server) read(conn net.Conn) {
	defer conn.Close()
	_ = conn.SetReadDeadline(time.Now().Add(5 * time.Second))
	sc := bufio.NewScanner(conn)
	sc.Buffer(make([]byte, 0, 4096), 1<<20)
	for sc.Scan() {
		var ev Event
		if json.Unmarshal(sc.Bytes(), &ev) != nil || ev.Type == "" {
			continue
		}
		select {
		case s.Events <- ev:
		default: // drop rather than block a shell hook
		}
	}
}

// Path returns the socket path being served.
func (s *Server) Path() string { return s.path }

// Close stops listening and removes the socket.
func (s *Server) Close() error {
	err := s.ln.Close()
	_ = os.Remove(s.path)
	return err
}

// Broadcast delivers an event to every familiar running for this user: the
// companion pane and each shell's perch. It returns how many received it.
func Broadcast(ev Event) int {
	paths, _ := filepath.Glob(filepath.Join(RuntimeDir(), "*.sock"))
	if p := os.Getenv(EnvSocket); p != "" {
		paths = append(paths, p)
	}
	seen := map[string]bool{}
	n := 0
	for _, p := range paths {
		if seen[p] {
			continue
		}
		seen[p] = true
		if Send(p, ev) == nil {
			n++
		}
	}
	return n
}
