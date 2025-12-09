package canvas

import (
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
)

// TestNewCanvas tests canvas creation with various dimensions
func TestNewCanvas(t *testing.T) {
	tests := []struct {
		name          string
		width, height int
		wantWidth     int
		wantHeight    int
	}{
		{"valid dimensions", 10, 5, 10, 5},
		{"zero width", 0, 5, 1, 5},
		{"zero height", 10, 0, 10, 1},
		{"negative width", -5, 10, 1, 10},
		{"negative height", 10, -5, 10, 1},
		{"both zero", 0, 0, 1, 1},
		{"both negative", -1, -1, 1, 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := NewCanvas(tt.width, tt.height)
			if c.Width != tt.wantWidth {
				t.Errorf("Width = %d, want %d", c.Width, tt.wantWidth)
			}
			if c.Height != tt.wantHeight {
				t.Errorf("Height = %d, want %d", c.Height, tt.wantHeight)
			}
			// Verify all cells are transparent spaces
			for y := 0; y < c.Height; y++ {
				for x := 0; x < c.Width; x++ {
					cell := c.Cells[y][x]
					if cell.Rune != ' ' || !cell.Transparent {
						t.Errorf("Cell at (%d, %d) not properly initialized", x, y)
					}
				}
			}
		})
	}
}

// TestSetGet tests setting and getting cells
func TestSetGet(t *testing.T) {
	c := NewCanvas(10, 10)
	style := lipgloss.NewStyle().Foreground(lipgloss.Color("#FF0000"))

	// Test valid set
	c.Set(5, 5, 'A', style)
	cell := c.Get(5, 5)
	if cell.Rune != 'A' {
		t.Errorf("Got rune %c, want 'A'", cell.Rune)
	}
	if cell.Transparent {
		t.Error("Cell should not be transparent after Set")
	}

	// Test out of bounds (should be ignored)
	c.Set(-1, 0, 'B', style)
	c.Set(0, -1, 'B', style)
	c.Set(100, 0, 'B', style)
	c.Set(0, 100, 'B', style)

	// Test out of bounds Get
	cell = c.Get(-1, 0)
	if !cell.Transparent || cell.Rune != ' ' {
		t.Error("Out of bounds Get should return transparent space")
	}
}

// TestDrawString tests string drawing including wide characters
func TestDrawString(t *testing.T) {
	c := NewCanvas(20, 5)
	style := lipgloss.NewStyle()

	// Test ASCII string
	c.DrawString(0, 0, "Hello", style)
	if c.Get(0, 0).Rune != 'H' {
		t.Error("First character not set correctly")
	}
	if c.Get(4, 0).Rune != 'o' {
		t.Error("Last character not set correctly")
	}

	// Test empty string
	c.DrawString(0, 1, "", style)

	// Test string with wide characters (emoji)
	c.DrawString(0, 2, "😀", style)
	// Wide character should take multiple cells
}

// TestDrawLines tests multi-line drawing
func TestDrawLines(t *testing.T) {
	c := NewCanvas(20, 10)
	style := lipgloss.NewStyle()
	lines := []string{"Line1", "Line2", "Line3"}

	c.DrawLines(0, 0, lines, style)

	// Check first line
	if c.Get(0, 0).Rune != 'L' {
		t.Error("First line not drawn correctly")
	}

	// Check second line
	if c.Get(0, 1).Rune != 'L' {
		t.Error("Second line not drawn correctly")
	}

	// Check third line
	if c.Get(0, 2).Rune != 'L' {
		t.Error("Third line not drawn correctly")
	}

	// Test empty lines
	c.DrawLines(0, 5, []string{}, style)
}

// TestOverlay tests canvas composition
func TestOverlay(t *testing.T) {
	base := NewCanvas(10, 10)
	overlay := NewCanvas(5, 5)
	style := lipgloss.NewStyle()

	// Fill base with 'A'
	base.Fill('A', style)

	// Fill overlay with 'B'
	overlay.Fill('B', style)

	// Overlay at position (2, 2)
	base.Overlay(overlay, 2, 2)

	// Check that overlay area has 'B'
	if base.Get(2, 2).Rune != 'B' {
		t.Error("Overlay not applied correctly")
	}
	if base.Get(6, 6).Rune != 'B' {
		t.Error("Overlay boundary not correct")
	}

	// Check that area outside overlay still has 'A'
	if base.Get(0, 0).Rune != 'A' {
		t.Error("Base canvas modified outside overlay area")
	}

	// Test nil overlay
	base.Overlay(nil, 0, 0)
}

// TestClone tests canvas cloning
func TestClone(t *testing.T) {
	original := NewCanvas(5, 5)
	style := lipgloss.NewStyle().Foreground(lipgloss.Color("#FF0000"))
	original.Set(2, 2, 'X', style)

	clone := original.Clone()

	// Check dimensions
	if clone.Width != original.Width || clone.Height != original.Height {
		t.Error("Clone dimensions don't match original")
	}

	// Check cell content
	if clone.Get(2, 2).Rune != 'X' {
		t.Error("Clone doesn't have same cell content")
	}

	// Modify clone and ensure original is unchanged
	clone.Set(2, 2, 'Y', style)
	if original.Get(2, 2).Rune != 'X' {
		t.Error("Modifying clone affected original")
	}
}

// TestMerge tests horizontal merging
func TestMerge(t *testing.T) {
	left := NewCanvas(5, 5)
	right := NewCanvas(5, 5)
	style := lipgloss.NewStyle()

	left.Fill('L', style)
	right.Fill('R', style)

	// Merge with gap of 2
	result := Merge(left, right, 2)

	expectedWidth := 5 + 2 + 5 // left + gap + right
	if result.Width != expectedWidth {
		t.Errorf("Merged width = %d, want %d", result.Width, expectedWidth)
	}

	// Check left side
	if result.Get(0, 0).Rune != 'L' {
		t.Error("Left canvas not merged correctly")
	}

	// Check right side
	if result.Get(7, 0).Rune != 'R' {
		t.Error("Right canvas not merged correctly")
	}

	// Test with nil canvases
	result = Merge(nil, nil, 0)
	if result.Width != 1 || result.Height != 1 {
		t.Error("Merge(nil, nil) should return 1x1 canvas")
	}

	result = Merge(left, nil, 0)
	if result.Width != left.Width {
		t.Error("Merge with nil right should return clone of left")
	}

	result = Merge(nil, right, 0)
	if result.Width != right.Width {
		t.Error("Merge with nil left should return clone of right")
	}
}

// TestStack tests vertical stacking
func TestStack(t *testing.T) {
	top := NewCanvas(5, 3)
	bottom := NewCanvas(5, 3)
	style := lipgloss.NewStyle()

	top.Fill('T', style)
	bottom.Fill('B', style)

	// Stack with gap of 1
	result := Stack(top, bottom, 1)

	expectedHeight := 3 + 1 + 3 // top + gap + bottom
	if result.Height != expectedHeight {
		t.Errorf("Stacked height = %d, want %d", result.Height, expectedHeight)
	}

	// Check top part
	if result.Get(0, 0).Rune != 'T' {
		t.Error("Top canvas not stacked correctly")
	}

	// Check bottom part
	if result.Get(0, 4).Rune != 'B' {
		t.Error("Bottom canvas not stacked correctly")
	}

	// Test with nil canvases
	result = Stack(nil, nil, 0)
	if result.Width != 1 || result.Height != 1 {
		t.Error("Stack(nil, nil) should return 1x1 canvas")
	}

	result = Stack(top, nil, 0)
	if result.Height != top.Height {
		t.Error("Stack with nil bottom should return clone of top")
	}

	result = Stack(nil, bottom, 0)
	if result.Height != bottom.Height {
		t.Error("Stack with nil top should return clone of bottom")
	}
}

// TestRender tests styled rendering
func TestRender(t *testing.T) {
	c := NewCanvas(10, 5)
	style := lipgloss.NewStyle()
	c.DrawString(0, 0, "Hello", style)
	c.DrawString(0, 1, "World", style)

	lines := c.Render()

	if len(lines) == 0 {
		t.Error("Render returned no lines")
	}

	// Should trim trailing empty lines
	if len(lines) > 2 {
		// Check if extra lines are actually empty
		for i := 2; i < len(lines); i++ {
			if strings.TrimSpace(stripAnsi(lines[i])) != "" {
				t.Errorf("Line %d should be trimmed but contains: %s", i, lines[i])
			}
		}
	}
}

// TestRenderPlain tests plain text rendering
func TestRenderPlain(t *testing.T) {
	c := NewCanvas(10, 3)
	style := lipgloss.NewStyle()
	c.DrawString(0, 0, "Test", style)

	lines := c.RenderPlain()

	if len(lines) != 3 {
		t.Errorf("RenderPlain returned %d lines, want 3", len(lines))
	}

	// First line should contain "Test"
	if !strings.Contains(lines[0], "Test") {
		t.Error("First line doesn't contain 'Test'")
	}

	// Should not contain ANSI codes
	for _, line := range lines {
		if strings.Contains(line, "\x1b") {
			t.Error("RenderPlain contains ANSI escape codes")
		}
	}
}

// TestContentHeightWidth tests content dimension calculation
func TestContentHeightWidth(t *testing.T) {
	c := NewCanvas(20, 20)
	style := lipgloss.NewStyle()

	// Empty canvas
	if c.ContentHeight() != 0 {
		t.Error("Empty canvas should have ContentHeight 0")
	}
	if c.ContentWidth() != 0 {
		t.Error("Empty canvas should have ContentWidth 0")
	}

	// Add content at specific position
	c.Set(5, 3, 'X', style)

	height := c.ContentHeight()
	if height != 4 { // 0-indexed, so line 3 means height 4
		t.Errorf("ContentHeight = %d, want 4", height)
	}

	width := c.ContentWidth()
	if width != 6 { // Column 5 means width 6
		t.Errorf("ContentWidth = %d, want 6", width)
	}
}

// TestResize tests canvas resizing
func TestResize(t *testing.T) {
	c := NewCanvas(10, 10)
	style := lipgloss.NewStyle()
	c.Set(5, 5, 'X', style)

	// Resize larger
	larger := c.Resize(20, 20)
	if larger.Width != 20 || larger.Height != 20 {
		t.Error("Resize to larger dimensions failed")
	}
	if larger.Get(5, 5).Rune != 'X' {
		t.Error("Content not preserved after resize")
	}

	// Resize smaller
	smaller := c.Resize(5, 5)
	if smaller.Width != 5 || smaller.Height != 5 {
		t.Error("Resize to smaller dimensions failed")
	}
}

// TestClear tests clearing the canvas
func TestClear(t *testing.T) {
	c := NewCanvas(5, 5)
	style := lipgloss.NewStyle()
	c.Fill('X', style)

	c.Clear()

	// All cells should be transparent spaces
	for y := 0; y < c.Height; y++ {
		for x := 0; x < c.Width; x++ {
			cell := c.Cells[y][x]
			if cell.Rune != ' ' || !cell.Transparent {
				t.Errorf("Cell at (%d, %d) not cleared properly", x, y)
			}
		}
	}
}

// TestFill tests filling the canvas
func TestFill(t *testing.T) {
	c := NewCanvas(5, 5)
	style := lipgloss.NewStyle().Foreground(lipgloss.Color("#FF0000"))

	c.Fill('*', style)

	// All cells should be '*' and non-transparent
	for y := 0; y < c.Height; y++ {
		for x := 0; x < c.Width; x++ {
			cell := c.Cells[y][x]
			if cell.Rune != '*' || cell.Transparent {
				t.Errorf("Cell at (%d, %d) not filled properly", x, y)
			}
		}
	}
}

// TestApplyStyle tests applying styles to canvas
func TestApplyStyle(t *testing.T) {
	c := NewCanvas(5, 5)
	originalStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#FF0000"))
	newStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#00FF00"))

	c.Fill('X', originalStyle)
	c.ApplyStyle(newStyle)

	// All non-transparent cells should have new style
	// We can't directly compare styles, so we just verify the method ran without error
	// and check that cells are still non-transparent
	for y := 0; y < c.Height; y++ {
		for x := 0; x < c.Width; x++ {
			cell := c.Cells[y][x]
			if !cell.Transparent && cell.Rune != 'X' {
				t.Errorf("Cell at (%d, %d) content changed unexpectedly", x, y)
			}
		}
	}

	// Test on canvas with transparent cells
	c2 := NewCanvas(5, 5)
	c2.Set(2, 2, 'Y', originalStyle)
	c2.ApplyStyle(newStyle)

	// Only the non-transparent cell should remain non-transparent
	if c2.Get(2, 2).Transparent {
		t.Error("Style application changed transparency incorrectly")
	}
	if c2.Get(2, 2).Rune != 'Y' {
		t.Error("Style application changed cell content")
	}
}

// TestFromLines tests creating canvas from lines
func TestFromLines(t *testing.T) {
	lines := []string{"Hello", "World", "Test"}
	style := lipgloss.NewStyle()

	c := FromLines(lines, style)

	if c.Height != 3 {
		t.Errorf("Canvas height = %d, want 3", c.Height)
	}

	// Check content
	plain := c.RenderPlain()
	if !strings.Contains(plain[0], "Hello") {
		t.Error("First line content not preserved")
	}

	// Test empty lines
	c2 := FromLines([]string{}, style)
	if c2.Width != 1 || c2.Height != 1 {
		t.Error("FromLines with empty slice should return 1x1 canvas")
	}

	// Test lines with different widths
	c3 := FromLines([]string{"Short", "A very long line"}, style)
	if c3.Width < len("A very long line") {
		t.Error("Canvas width should accommodate longest line")
	}
}

// TestStringWidth tests width calculation
func TestStringWidth(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  int
	}{
		{"ASCII", "Hello", 5},
		{"empty", "", 0},
		{"spaces", "   ", 3},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := StringWidth(tt.input)
			if got != tt.want {
				t.Errorf("StringWidth(%q) = %d, want %d", tt.input, got, tt.want)
			}
		})
	}
}

// TestStripAnsi tests ANSI code stripping
func TestStripAnsi(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"no codes", "Hello", "Hello"},
		{"with color", "\x1b[31mRed\x1b[0m", "Red"},
		{"empty", "", ""},
		{"only codes", "\x1b[31m\x1b[0m", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := stripAnsi(tt.input)
			if got != tt.want {
				t.Errorf("stripAnsi(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

// TestEdgeCases tests various edge cases
func TestEdgeCases(t *testing.T) {
	t.Run("overlay outside bounds", func(t *testing.T) {
		base := NewCanvas(10, 10)
		overlay := NewCanvas(5, 5)
		style := lipgloss.NewStyle()
		overlay.Fill('X', style)

		// Overlay partially outside
		base.Overlay(overlay, 8, 8)
		// Should handle gracefully
	})

	t.Run("negative gap in merge", func(t *testing.T) {
		left := NewCanvas(5, 5)
		right := NewCanvas(5, 5)
		// Negative gap should still work (overlap)
		result := Merge(left, right, -2)
		if result == nil {
			t.Error("Merge with negative gap failed")
		}
	})

	t.Run("very large dimensions", func(t *testing.T) {
		// Should handle large canvases
		c := NewCanvas(1000, 1000)
		if c.Width != 1000 || c.Height != 1000 {
			t.Error("Failed to create large canvas")
		}
	})
}
