package bubble

import (
	"strings"
	"testing"
)

func TestNew(t *testing.T) {
	b := New("test", 40, StyleSay)
	
	if b.Text != "test" {
		t.Errorf("Expected text 'test', got '%s'", b.Text)
	}
	
	if b.Width != 40 {
		t.Errorf("Expected width 40, got %d", b.Width)
	}
	
	if b.Style != StyleSay {
		t.Errorf("Expected StyleSay, got %v", b.Style)
	}
	
	if b.ThinkChar != "\\" {
		t.Errorf("Expected think char '\\', got '%s'", b.ThinkChar)
	}
}

func TestNewThinkStyle(t *testing.T) {
	b := New("test", 40, StyleThink)
	
	if b.ThinkChar != "o" {
		t.Errorf("Expected think char 'o', got '%s'", b.ThinkChar)
	}
}

func TestRender(t *testing.T) {
	b := New("Hello", 40, StyleSay)
	lines := b.Render()
	
	if len(lines) == 0 {
		t.Error("Expected non-empty output")
	}
	
	// Check for borders
	if !strings.Contains(lines[0], "_") {
		t.Error("Expected top border with underscores")
	}
	
	if !strings.Contains(lines[len(lines)-1], "-") {
		t.Error("Expected bottom border with dashes")
	}
}

func TestWrapText(t *testing.T) {
	b := New("", 10, StyleSay)
	lines := b.wrapText("This is a long line that should wrap", 10)
	
	if len(lines) <= 1 {
		t.Error("Expected text to be wrapped into multiple lines")
	}
	
	for _, line := range lines {
		if len(line) > 10 {
			t.Errorf("Line exceeds width: '%s' (length %d)", line, len(line))
		}
	}
}

func TestPadRight(t *testing.T) {
	b := New("", 10, StyleSay)
	
	result := b.padRight("test", 10)
	if len(result) != 10 {
		t.Errorf("Expected padded string length 10, got %d", len(result))
	}
	
	if !strings.HasPrefix(result, "test") {
		t.Error("Expected string to start with 'test'")
	}
}

func TestRenderEmptyText(t *testing.T) {
	b := New("", 40, StyleSay)
	lines := b.Render()
	
	if len(lines) != 0 {
		t.Errorf("Expected empty output for empty text, got %d lines", len(lines))
	}
}

func TestRenderSingleLine(t *testing.T) {
	b := New("Hi", 40, StyleSay)
	lines := b.Render()
	
	// Should have: top border, content, bottom border
	if len(lines) != 3 {
		t.Errorf("Expected 3 lines for single word, got %d", len(lines))
	}
	
	// Single line should use < > brackets
	if !strings.Contains(lines[1], "<") || !strings.Contains(lines[1], ">") {
		t.Error("Expected single line to use < > brackets")
	}
}

func TestRenderMultiLine(t *testing.T) {
	b := New("Hello World this is a test", 10, StyleSay)
	lines := b.Render()
	
	// Should have multiple content lines
	if len(lines) < 4 {
		t.Errorf("Expected at least 4 lines, got %d", len(lines))
	}
	
	// Multi-line should use / \ | characters
	hasSlash := false
	for _, line := range lines {
		if strings.Contains(line, "/") || strings.Contains(line, "\\") {
			hasSlash = true
		}
	}
	
	if !hasSlash {
		t.Error("Expected multi-line bubble to use / or \\ characters")
	}
}
