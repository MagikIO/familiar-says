package wrap

import (
	"os"
	"strings"
	"testing"
)

type captured struct{ seq, out string }

func feed(chunks ...string) (passed string, got []captured) {
	f := &Filter{OnCommand: func(seq string, raw []byte) { got = append(got, captured{seq, string(raw)}) }}
	var sb strings.Builder
	for _, c := range chunks {
		sb.Write(f.Write([]byte(c)))
	}
	sb.Write(f.Flush())
	return sb.String(), got
}

func TestMarkersAreRemovedAndOutputCaptured(t *testing.T) {
	passed, got := feed("prompt> ls\r\n\x1b]7797;S;4\x1b\\file-a\r\nfile-b\r\n\x1b]7797;E;4\x1b\\prompt> ")
	if passed != "prompt> ls\r\nfile-a\r\nfile-b\r\nprompt> " {
		t.Errorf("the terminal should see everything but the markers, got %q", passed)
	}
	if len(got) != 1 || got[0].seq != "4" || got[0].out != "file-a\r\nfile-b\r\n" {
		t.Errorf("captured %+v", got)
	}
}

func TestMarkersSplitAcrossReads(t *testing.T) {
	whole := "a\x1b]7797;S;12\x07hello\x1b]7797;E;12\x07b"
	for cut := 1; cut < len(whole); cut++ {
		passed, got := feed(whole[:cut], whole[cut:])
		if passed != "ahellob" {
			t.Fatalf("cut at %d: terminal saw %q", cut, passed)
		}
		if len(got) != 1 || got[0].seq != "12" || got[0].out != "hello" {
			t.Fatalf("cut at %d: captured %+v", cut, got)
		}
	}
	// One byte at a time is the worst case.
	var chunks []string
	for _, b := range []byte(whole) {
		chunks = append(chunks, string(b))
	}
	if passed, got := feed(chunks...); passed != "ahellob" || len(got) != 1 {
		t.Errorf("byte-at-a-time: %q %+v", passed, got)
	}
}

func TestOtherEscapeSequencesPassUntouched(t *testing.T) {
	in := "\x1b[31mred\x1b[0m \x1b]0;title\x07 \x1b]133;C\x1b\\ \x1b]7;file://x\x07 \x1b]779 lone \x1b"
	if passed, got := feed(in); passed != in || len(got) != 0 {
		t.Errorf("non-marker sequences must pass through byte for byte:\n got %q\nwant %q", passed, in)
	}
	// Something that starts like a marker but never ends must not be held forever.
	in = "\x1b]7797;S;" + strings.Repeat("9", 200) + " and on"
	if passed, _ := feed(in); passed != in {
		t.Errorf("a runaway marker should be released, got %q", passed)
	}
}

func TestLongOutputKeepsHeadAndTail(t *testing.T) {
	big := "FIRST LINE\n" + strings.Repeat("noise noise noise\n", 20000) + "LAST LINE\n"
	_, got := feed("\x1b]7797;S;1\x07", big, "\x1b]7797;E;1\x07")
	if len(got) != 1 {
		t.Fatal("nothing captured")
	}
	out := got[0].out
	if len(out) > keepHead+keepTail+64 {
		t.Errorf("capture should be bounded, got %d bytes", len(out))
	}
	if !strings.HasPrefix(out, "FIRST LINE") || !strings.HasSuffix(out, "LAST LINE\n") || !strings.Contains(out, "[...]") {
		t.Error("capture should keep the start and the end, and mark the gap")
	}
}

func TestOutputOutsideACommandIsNotRecorded(t *testing.T) {
	_, got := feed("typing at the prompt, secrets and all\x1b]7797;E;9\x07")
	if len(got) != 0 {
		t.Errorf("an end marker with no start should capture nothing, got %+v", got)
	}
}

func TestClean(t *testing.T) {
	raw := "\x1b[1;31merror\x1b[0m: \x1b]8;;http://x\x1b\\link\x1b]8;;\x1b\\ here\r\n" +
		"10%\r50%\r100% done\r\n" +
		"typo\b\b\b\bfixed\r\n" +
		"\ttabbed\x07\r\n"
	want := "error: link here\n100% done\nfixed\n    tabbed"
	if got := Clean([]byte(raw)); got != want {
		t.Errorf("got %q\nwant %q", got, want)
	}
}

func TestFocusReportsAreConsumedUnlessTheProgramWantsThem(t *testing.T) {
	path := t.TempDir() + "/focus"
	f := &focusTracker{path: path}
	state := func() string {
		b, _ := os.ReadFile(path)
		return strings.TrimSpace(string(b))
	}
	f.set(true)

	if got := string(f.input([]byte("ls\x1b[O -la"))); got != "ls -la" || state() != "0" {
		t.Errorf("focus-out should be removed and recorded: %q, state %q", got, state())
	}
	if got := string(f.input([]byte("\x1b[I"))); got != "" || state() != "1" {
		t.Errorf("focus-in should be removed and recorded: %q, state %q", got, state())
	}
	if got := string(f.input([]byte("\x1b"))); got != "\x1b" {
		t.Errorf("a lone Escape must pass straight through, got %q", got)
	}
	if got := string(f.input([]byte("\x1b[A\x1bOP"))); got != "\x1b[A\x1bOP" {
		t.Errorf("other keys must be untouched, got %q", got)
	}

	// vim (say) turns focus reporting on: it should now receive the reports.
	if f.output([]byte("\x1b[?1049h\x1b[?1004h")) {
		t.Error("turning reporting on is not turning it off")
	}
	if got := string(f.input([]byte("\x1b[O"))); got != "\x1b[O" || state() != "0" {
		t.Errorf("a program that asked for focus reports should get them: %q, state %q", got, state())
	}
	// ... and when it turns reporting off on exit, we need to turn ours back on.
	if !f.output([]byte("\x1b[?1004l\x1b[?1049l")) {
		t.Error("the caller should be told to re-enable focus reporting")
	}
	if got := string(f.input([]byte("\x1b[I"))); got != "" {
		t.Errorf("reports should be consumed again, got %q", got)
	}
}
