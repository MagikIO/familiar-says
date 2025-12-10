package animation

import (
	"testing"
	"time"

	"github.com/MagikIO/familiar-says/internal/canvas"
	"github.com/charmbracelet/lipgloss"
)

func TestNewFramePlayer(t *testing.T) {
	char := &canvas.Character{
		Name: "test",
		Art:  []string{"test"},
	}
	anim := &canvas.AnimationSequence{
		Frames: []canvas.AnimationFrame{
			{DurationMs: 100, Eyes: "@@"},
			{DurationMs: 100, Eyes: "--"},
		},
		Loop: true,
	}
	styles := canvas.CharacterStyles{
		Outline: lipgloss.NewStyle(),
		Eyes:    lipgloss.NewStyle(),
		Mouth:   lipgloss.NewStyle(),
	}

	player := NewFramePlayer(char, anim, styles, "oo", "Y")

	if player == nil {
		t.Fatal("NewFramePlayer returned nil")
	}
	if player.CurrentFrameIndex() != 0 {
		t.Errorf("expected initial frame 0, got %d", player.CurrentFrameIndex())
	}
	if player.TotalFrames() != 2 {
		t.Errorf("expected 2 frames, got %d", player.TotalFrames())
	}
	if player.IsComplete() {
		t.Error("player should not be complete initially")
	}
}

func TestFramePlayerTick(t *testing.T) {
	char := &canvas.Character{
		Name: "test",
		Art:  []string{"( @@ )"},
		Eyes: &canvas.Slot{Line: 0, Col: 2, Width: 2, Placeholder: "@@"},
	}
	anim := &canvas.AnimationSequence{
		Frames: []canvas.AnimationFrame{
			{DurationMs: 100, Eyes: "@@"},
			{DurationMs: 100, Eyes: "--"},
		},
		Loop: false,
	}
	styles := canvas.CharacterStyles{
		Outline: lipgloss.NewStyle(),
		Eyes:    lipgloss.NewStyle(),
		Mouth:   lipgloss.NewStyle(),
	}

	player := NewFramePlayer(char, anim, styles, "oo", "")

	// First tick shouldn't advance (not enough time)
	player.Tick(50 * time.Millisecond)
	if player.CurrentFrameIndex() != 0 {
		t.Errorf("expected frame 0 after 50ms, got %d", player.CurrentFrameIndex())
	}

	// This tick should advance to frame 1
	player.Tick(50 * time.Millisecond)
	if player.CurrentFrameIndex() != 1 {
		t.Errorf("expected frame 1 after 100ms, got %d", player.CurrentFrameIndex())
	}

	// Another 100ms should complete the animation
	player.Tick(100 * time.Millisecond)
	if !player.IsComplete() {
		t.Error("player should be complete after playing all frames")
	}
}

func TestFramePlayerLoop(t *testing.T) {
	char := &canvas.Character{
		Name: "test",
		Art:  []string{"test"},
	}
	anim := &canvas.AnimationSequence{
		Frames: []canvas.AnimationFrame{
			{DurationMs: 100},
			{DurationMs: 100},
		},
		Loop: true,
	}
	styles := canvas.CharacterStyles{}

	player := NewFramePlayer(char, anim, styles, "", "")

	// Advance through both frames
	player.Tick(100 * time.Millisecond)
	player.Tick(100 * time.Millisecond)

	// Should loop back to frame 0, not be complete
	if player.IsComplete() {
		t.Error("looping animation should not be complete")
	}
	if player.CurrentFrameIndex() != 0 {
		t.Errorf("expected frame 0 after loop, got %d", player.CurrentFrameIndex())
	}
}

func TestFramePlayerReset(t *testing.T) {
	char := &canvas.Character{
		Name: "test",
		Art:  []string{"test"},
	}
	anim := &canvas.AnimationSequence{
		Frames: []canvas.AnimationFrame{
			{DurationMs: 100},
			{DurationMs: 100},
		},
		Loop: false,
	}
	styles := canvas.CharacterStyles{}

	player := NewFramePlayer(char, anim, styles, "", "")

	// Advance and complete
	player.Tick(200 * time.Millisecond)
	if !player.IsComplete() {
		t.Error("should be complete")
	}

	// Reset
	player.Reset()

	if player.IsComplete() {
		t.Error("should not be complete after reset")
	}
	if player.CurrentFrameIndex() != 0 {
		t.Errorf("expected frame 0 after reset, got %d", player.CurrentFrameIndex())
	}
}

func TestFramePlayerNilAnimation(t *testing.T) {
	char := &canvas.Character{
		Name: "test",
		Art:  []string{"test"},
	}
	styles := canvas.CharacterStyles{}

	player := NewFramePlayer(char, nil, styles, "oo", "")

	// Should not panic
	result := player.Tick(100 * time.Millisecond)
	if result == nil {
		t.Error("should return a canvas even with nil animation")
	}
	if player.TotalFrames() != 0 {
		t.Errorf("expected 0 frames with nil animation, got %d", player.TotalFrames())
	}
}

func TestFramePlayerSetAnimation(t *testing.T) {
	char := &canvas.Character{
		Name: "test",
		Art:  []string{"test"},
	}
	anim1 := &canvas.AnimationSequence{
		Frames: []canvas.AnimationFrame{{DurationMs: 100}},
		Loop:   false,
	}
	anim2 := &canvas.AnimationSequence{
		Frames: []canvas.AnimationFrame{{DurationMs: 100}, {DurationMs: 100}},
		Loop:   false,
	}
	styles := canvas.CharacterStyles{}

	player := NewFramePlayer(char, anim1, styles, "", "")

	if player.TotalFrames() != 1 {
		t.Errorf("expected 1 frame, got %d", player.TotalFrames())
	}

	player.SetAnimation(anim2)

	if player.TotalFrames() != 2 {
		t.Errorf("expected 2 frames after SetAnimation, got %d", player.TotalFrames())
	}
	if player.CurrentFrameIndex() != 0 {
		t.Error("SetAnimation should reset frame index")
	}
}

func TestFrameWithArtOverride(t *testing.T) {
	char := &canvas.Character{
		Name: "test",
		Art:  []string{"frame0"},
	}
	anim := &canvas.AnimationSequence{
		Frames: []canvas.AnimationFrame{
			{DurationMs: 100, Art: []string{"frame1"}},
		},
		Loop: false,
	}
	styles := canvas.CharacterStyles{}

	player := NewFramePlayer(char, anim, styles, "", "")
	result := player.Tick(0)

	// The canvas should contain the overridden art
	lines := result.Render()
	if len(lines) == 0 {
		t.Fatal("expected at least one line")
	}
	if lines[0] != "frame1" {
		t.Errorf("expected 'frame1', got '%s'", lines[0])
	}
}

func TestAbs(t *testing.T) {
	tests := []struct {
		input    int
		expected int
	}{
		{5, 5},
		{-5, 5},
		{0, 0},
		{-1, 1},
	}

	for _, tc := range tests {
		result := abs(tc.input)
		if result != tc.expected {
			t.Errorf("abs(%d) = %d, expected %d", tc.input, result, tc.expected)
		}
	}
}
