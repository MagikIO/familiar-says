package ipc

import (
	"errors"
	"net"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// sockPath returns a short socket path; t.TempDir can exceed the ~100 bytes a
// Unix socket address may hold.
func sockPath(t *testing.T) string {
	t.Helper()
	dir, err := os.MkdirTemp("", "fs")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.RemoveAll(dir) })
	return filepath.Join(dir, "sub", "t.sock")
}

func receive(t *testing.T, s *Server) Event {
	t.Helper()
	select {
	case ev := <-s.Events:
		return ev
	case <-time.After(2 * time.Second):
		t.Fatal("no event arrived")
		return Event{}
	}
}

func TestSendAndReceive(t *testing.T) {
	path := sockPath(t)
	s, err := Listen(path)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()

	want := Event{Type: TypeCommand, Cmd: `echo "héllo" | wc -c`, Exit: 3, DurationMs: 1234, Cwd: "/tmp", Shell: "fish", Cols: 120, Seq: 7}
	if err := Send(path, want); err != nil {
		t.Fatal(err)
	}
	if got := receive(t, s); got != want {
		t.Errorf("got %+v, want %+v", got, want)
	}

	if info, err := os.Stat(path); err != nil || info.Mode().Perm() != 0o600 {
		t.Errorf("socket should be private to the user: %v %v", info, err)
	}
}

func TestSendWithNobodyListening(t *testing.T) {
	path := sockPath(t)
	if err := Send(path, Event{Type: TypePet}); !errors.Is(err, ErrNotRunning) {
		t.Errorf("want ErrNotRunning when the socket does not exist, got %v", err)
	}
	// A socket file left behind by a process that died.
	os.MkdirAll(filepath.Dir(path), 0o700)
	ln, err := net.Listen("unix", path)
	if err != nil {
		t.Fatal(err)
	}
	ln.(*net.UnixListener).SetUnlinkOnClose(false)
	ln.Close()
	if err := Send(path, Event{Type: TypePet}); !errors.Is(err, ErrNotRunning) {
		t.Errorf("want ErrNotRunning for a dead socket, got %v", err)
	}

	s, err := Listen(path)
	if err != nil {
		t.Fatalf("a dead socket should be replaced, got %v", err)
	}
	defer s.Close()
	if err := Send(path, Event{Type: TypePet}); err != nil {
		t.Fatal(err)
	}
	receive(t, s)
}

func TestOnlyOneListenerPerSocket(t *testing.T) {
	path := sockPath(t)
	s, err := Listen(path)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Listen(path); err == nil || !strings.Contains(err.Error(), "already listening") {
		t.Errorf("a second listener should be refused, got %v", err)
	}
	s.Close()
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Error("Close should remove the socket")
	}
}

func TestGarbageIsIgnored(t *testing.T) {
	path := sockPath(t)
	s, err := Listen(path)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	conn, err := net.Dial("unix", path)
	if err != nil {
		t.Fatal(err)
	}
	conn.Write([]byte("not json\n{}\n{\"type\":\"pet\"}\n"))
	conn.Close()
	if got := receive(t, s); got.Type != TypePet {
		t.Errorf("only the well-formed event should come through, got %+v", got)
	}
}

func TestOverlongPathIsExplained(t *testing.T) {
	_, err := Listen("/tmp/" + strings.Repeat("x", 120) + "/a.sock")
	if err == nil || !strings.Contains(err.Error(), "too long") {
		t.Errorf("want a clear error for an overlong socket path, got %v", err)
	}
}

func TestSocketPathHonoursEnvironment(t *testing.T) {
	t.Setenv(EnvSocket, "/somewhere/else.sock")
	if got := SocketPath(); got != "/somewhere/else.sock" {
		t.Errorf("got %q", got)
	}
	t.Setenv(EnvSocket, "")
	t.Setenv("XDG_RUNTIME_DIR", "/run/user/42")
	if got := SocketPath(); got != "/run/user/42/familiar-says/live.sock" {
		t.Errorf("got %q", got)
	}
}
