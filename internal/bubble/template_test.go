package bubble

import (
	"strings"
	"testing"
)

func TestGetTemplate(t *testing.T) {
	tests := []struct {
		name     string
		template string
		wantNil  bool
	}{
		{"say template", "say", false},
		{"think template", "think", false},
		{"shout template", "shout", false},
		{"whisper template", "whisper", false},
		{"song template", "song", false},
		{"code template", "code", false},
		{"unknown defaults to say", "unknown", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpl := GetTemplate(tt.template)
			if (tmpl == nil) != tt.wantNil {
				t.Errorf("GetTemplate(%q) returned nil = %v, want nil = %v", tt.template, tmpl == nil, tt.wantNil)
			}
		})
	}
}

func TestGetTemplateForStyle(t *testing.T) {
	tests := []struct {
		style    Style
		wantName string
	}{
		{StyleSay, "say"},
		{StyleThink, "think"},
		{StyleShout, "shout"},
		{StyleWhisper, "whisper"},
		{StyleSong, "song"},
		{StyleCode, "code"},
	}

	for _, tt := range tests {
		t.Run(tt.wantName, func(t *testing.T) {
			tmpl := GetTemplateForStyle(tt.style)
			if tmpl == nil {
				t.Fatalf("GetTemplateForStyle(%v) returned nil", tt.style)
			}
			if tmpl.Name != tt.wantName {
				t.Errorf("GetTemplateForStyle(%v) = %q, want %q", tt.style, tmpl.Name, tt.wantName)
			}
		})
	}
}

func TestBubbleTemplateFields(t *testing.T) {
	tmpl := GetTemplate("shout")

	if tmpl.TopBorder != "^" {
		t.Errorf("Shout template TopBorder = %q, want %q", tmpl.TopBorder, "^")
	}
	if tmpl.BottomBorder != "v" {
		t.Errorf("Shout template BottomBorder = %q, want %q", tmpl.BottomBorder, "v")
	}
	if tmpl.Connector != "!" {
		t.Errorf("Shout template Connector = %q, want %q", tmpl.Connector, "!")
	}
	if tmpl.Suffix != "!!!" {
		t.Errorf("Shout template Suffix = %q, want %q", tmpl.Suffix, "!!!")
	}
}

func TestSongTemplateHasDecorators(t *testing.T) {
	tmpl := GetTemplate("song")

	if tmpl.Prefix == "" {
		t.Error("Song template should have a prefix")
	}
	if tmpl.Suffix == "" {
		t.Error("Song template should have a suffix")
	}
	if tmpl.BorderDecorator == "" {
		t.Error("Song template should have a border decorator")
	}
	if !strings.Contains(tmpl.Prefix, "♪") && !strings.Contains(tmpl.Suffix, "♪") {
		t.Error("Song template should contain musical notes")
	}
}

func TestCodeTemplateIsCodeBlock(t *testing.T) {
	tmpl := GetTemplate("code")

	if !tmpl.IsCodeBlock {
		t.Error("Code template should have IsCodeBlock = true")
	}
	if !tmpl.SyntaxHighlight {
		t.Error("Code template should have SyntaxHighlight = true")
	}
}

func TestAllTemplates(t *testing.T) {
	templates := AllTemplates()

	expected := []string{"say", "think", "shout", "whisper", "song", "code", "dream", "angry", "box"}

	for _, name := range expected {
		found := false
		for _, tmpl := range templates {
			if tmpl == name {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("AllTemplates() missing %q", name)
		}
	}
}

func TestValidateTemplate(t *testing.T) {
	tests := []struct {
		name  string
		valid bool
	}{
		{"say", true},
		{"think", true},
		{"shout", true},
		{"whisper", true},
		{"song", true},
		{"code", true},
		{"nonexistent", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ValidateTemplate(tt.name)
			if got != tt.valid {
				t.Errorf("ValidateTemplate(%q) = %v, want %v", tt.name, got, tt.valid)
			}
		})
	}
}

func TestTailDirection(t *testing.T) {
	tests := []struct {
		dir  TailDirection
		want string
	}{
		{TailDown, "down"},
		{TailUp, "up"},
		{TailLeft, "left"},
		{TailRight, "right"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			if got := tt.dir.String(); got != tt.want {
				t.Errorf("TailDirection.String() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestParseTailDirection(t *testing.T) {
	tests := []struct {
		input string
		want  TailDirection
	}{
		{"down", TailDown},
		{"up", TailUp},
		{"left", TailLeft},
		{"right", TailRight},
		{"DOWN", TailDown},
		{"UP", TailUp},
		{"invalid", TailDown}, // default
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			if got := ParseTailDirection(tt.input); got != tt.want {
				t.Errorf("ParseTailDirection(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestStyleString(t *testing.T) {
	tests := []struct {
		style Style
		want  string
	}{
		{StyleSay, "say"},
		{StyleThink, "think"},
		{StyleShout, "shout"},
		{StyleWhisper, "whisper"},
		{StyleSong, "song"},
		{StyleCode, "code"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			if got := tt.style.String(); got != tt.want {
				t.Errorf("Style.String() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestParseStyle(t *testing.T) {
	tests := []struct {
		input string
		want  Style
	}{
		{"say", StyleSay},
		{"think", StyleThink},
		{"shout", StyleShout},
		{"whisper", StyleWhisper},
		{"song", StyleSong},
		{"code", StyleCode},
		{"SHOUT", StyleShout},
		{"invalid", StyleSay}, // default
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			if got := ParseStyle(tt.input); got != tt.want {
				t.Errorf("ParseStyle(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestAllStyles(t *testing.T) {
	styles := AllStyles()
	expected := []string{"say", "think", "shout", "whisper", "song", "code"}

	if len(styles) != len(expected) {
		t.Errorf("AllStyles() returned %d styles, want %d", len(styles), len(expected))
	}

	for i, want := range expected {
		if styles[i] != want {
			t.Errorf("AllStyles()[%d] = %q, want %q", i, styles[i], want)
		}
	}
}

func TestRenderLines(t *testing.T) {
	lines := RenderLines("Hello", 40, StyleSay)

	if len(lines) == 0 {
		t.Error("RenderLines should return non-empty output")
	}

	// Check for top border
	if !strings.Contains(lines[0], "_") {
		t.Error("Expected top border with underscores")
	}
}

func TestRenderLinesShout(t *testing.T) {
	lines := RenderLines("Hey", 40, StyleShout)

	if len(lines) == 0 {
		t.Error("RenderLines should return non-empty output")
	}

	// Check for shout border characters
	if !strings.Contains(lines[0], "^") {
		t.Error("Expected shout bubble to have ^ in top border")
	}

	// Check for !!! suffix
	foundSuffix := false
	for _, line := range lines {
		if strings.Contains(line, "!!!") {
			foundSuffix = true
			break
		}
	}
	if !foundSuffix {
		t.Error("Expected shout bubble to have !!! suffix")
	}
}

func TestRenderLinesSong(t *testing.T) {
	lines := RenderLines("La la la", 40, StyleSong)

	if len(lines) == 0 {
		t.Error("RenderLines should return non-empty output")
	}

	// Check for musical notes
	foundNote := false
	for _, line := range lines {
		if strings.Contains(line, "♪") || strings.Contains(line, "♫") {
			foundNote = true
			break
		}
	}
	if !foundNote {
		t.Error("Expected song bubble to contain musical notes")
	}
}

func TestRenderWithTemplate(t *testing.T) {
	tmpl := GetTemplate("whisper")
	lines := RenderWithTemplate("Secret", 40, tmpl)

	if len(lines) == 0 {
		t.Error("RenderWithTemplate should return non-empty output")
	}

	// Check for whisper border
	if !strings.Contains(lines[0], ".") {
		t.Error("Expected whisper bubble to have dotted top border")
	}

	// Check for prefix/suffix
	foundParens := false
	for _, line := range lines {
		if strings.Contains(line, "(") && strings.Contains(line, ")") {
			foundParens = true
			break
		}
	}
	if !foundParens {
		t.Error("Expected whisper bubble to have parentheses around text")
	}
}

func TestGenerateConnectorLines(t *testing.T) {
	lines := GenerateConnectorLines(StyleSay, 2, 4, TailDown)

	if len(lines) != 2 {
		t.Errorf("Expected 2 connector lines, got %d", len(lines))
	}

	// Check that lines contain the connector character
	if !strings.Contains(lines[0], "\\") {
		t.Error("Expected connector line to contain backslash")
	}
}

func TestGenerateConnectorLinesThink(t *testing.T) {
	lines := GenerateConnectorLines(StyleThink, 2, 4, TailDown)

	if !strings.Contains(lines[0], "o") {
		t.Error("Expected think connector to contain 'o'")
	}
}

func TestGetConnectorCharForStyle(t *testing.T) {
	tests := []struct {
		style Style
		want  string
	}{
		{StyleSay, "\\"},
		{StyleThink, "o"},
		{StyleShout, "!"},
		{StyleWhisper, "."},
		{StyleSong, "♪"},
	}

	for _, tt := range tests {
		t.Run(tt.style.String(), func(t *testing.T) {
			if got := GetConnectorCharForStyle(tt.style); got != tt.want {
				t.Errorf("GetConnectorCharForStyle(%v) = %q, want %q", tt.style, got, tt.want)
			}
		})
	}
}
