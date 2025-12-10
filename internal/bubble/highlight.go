package bubble

import (
	"bytes"
	"strings"

	"github.com/alecthomas/chroma/v2"
	"github.com/alecthomas/chroma/v2/formatters"
	"github.com/alecthomas/chroma/v2/lexers"
	"github.com/alecthomas/chroma/v2/styles"
)

// HighlightConfig configures syntax highlighting for code bubbles
type HighlightConfig struct {
	Language string // Language to highlight (e.g., "go", "python", "javascript")
	Style    string // Chroma style name (e.g., "monokai", "dracula", "github")
}

// DefaultHighlightConfig returns default syntax highlighting settings
func DefaultHighlightConfig() HighlightConfig {
	return HighlightConfig{
		Language: "",         // Auto-detect
		Style:    "monokai",  // Popular terminal-friendly style
	}
}

// HighlightCode applies syntax highlighting to code text.
// It returns ANSI-colored text suitable for terminal output.
func HighlightCode(code string, config HighlightConfig) string {
	// Get lexer
	var lexer chroma.Lexer
	if config.Language != "" {
		lexer = lexers.Get(config.Language)
	}
	if lexer == nil {
		lexer = lexers.Analyse(code)
	}
	if lexer == nil {
		lexer = lexers.Fallback
	}
	lexer = chroma.Coalesce(lexer)

	// Get style
	style := styles.Get(config.Style)
	if style == nil {
		style = styles.Fallback
	}

	// Use terminal256 formatter for ANSI output
	formatter := formatters.Get("terminal256")
	if formatter == nil {
		formatter = formatters.Fallback
	}

	// Tokenize and format
	iterator, err := lexer.Tokenise(nil, code)
	if err != nil {
		return code // Return unhighlighted on error
	}

	var buf bytes.Buffer
	err = formatter.Format(&buf, style, iterator)
	if err != nil {
		return code // Return unhighlighted on error
	}

	return buf.String()
}

// HighlightCodeLines applies syntax highlighting and returns lines.
// This is useful for bubble rendering where we need individual lines.
func HighlightCodeLines(code string, config HighlightConfig) []string {
	highlighted := HighlightCode(code, config)
	return strings.Split(highlighted, "\n")
}

// DetectLanguage attempts to detect the programming language from code content.
func DetectLanguage(code string) string {
	lexer := lexers.Analyse(code)
	if lexer != nil {
		config := lexer.Config()
		if config != nil && len(config.Aliases) > 0 {
			return config.Aliases[0]
		}
		if config != nil {
			return config.Name
		}
	}
	return ""
}

// AvailableStyles returns a list of available syntax highlighting styles.
func AvailableStyles() []string {
	return styles.Names()
}

// AvailableLanguages returns a list of available language lexers.
func AvailableLanguages() []string {
	names := lexers.Names(false)
	return names
}

// IsValidLanguage checks if a language name is valid.
func IsValidLanguage(lang string) bool {
	return lexers.Get(lang) != nil
}

// IsValidStyle checks if a style name is valid.
func IsValidStyle(style string) bool {
	return styles.Get(style) != nil
}

// RenderCodeBubble renders a code block with optional syntax highlighting.
// It uses the code template and applies highlighting if enabled.
func RenderCodeBubble(code string, width int, config HighlightConfig) []string {
	tmpl := GetTemplate("code")
	
	// Apply syntax highlighting if configured
	displayCode := code
	if tmpl.SyntaxHighlight {
		displayCode = HighlightCode(code, config)
	}
	
	// Use the template-based rendering
	return RenderWithTemplate(displayCode, width, tmpl)
}

// CodeBubbleWithLanguage creates a code bubble with language header.
func CodeBubbleWithLanguage(code, language string, width int, config HighlightConfig) []string {
	if language == "" {
		language = DetectLanguage(code)
	}
	if config.Language == "" {
		config.Language = language
	}
	
	tmpl := GetTemplate("code")
	
	// Apply syntax highlighting
	displayCode := code
	if tmpl.SyntaxHighlight && language != "" {
		displayCode = HighlightCode(code, config)
	}
	
	// Calculate max line length
	lines := strings.Split(displayCode, "\n")
	maxLen := 0
	for _, line := range lines {
		// Strip ANSI codes for width calculation
		stripped := stripANSI(line)
		if len(stripped) > maxLen {
			maxLen = len(stripped)
		}
	}
	if maxLen > width {
		maxLen = width
	}
	
	result := []string{}
	
	// Top border with language indicator
	langHeader := ""
	if language != "" {
		langHeader = " " + language + " "
	}
	topBorderLen := maxLen + 2 - len(langHeader)
	if topBorderLen < 0 {
		topBorderLen = 0
	}
	topBorder := tmpl.TopLeftCorner + strings.Repeat(tmpl.TopBorder, topBorderLen/2) + langHeader + strings.Repeat(tmpl.TopBorder, (topBorderLen+1)/2) + tmpl.TopRightCorner
	result = append(result, topBorder)
	
	// Content lines
	for _, line := range lines {
		// Pad to width (accounting for ANSI codes)
		stripped := stripANSI(line)
		padding := maxLen - len(stripped)
		if padding < 0 {
			padding = 0
		}
		paddedLine := line + strings.Repeat(" ", padding)
		result = append(result, tmpl.SingleLeft+" "+paddedLine+" "+tmpl.SingleRight)
	}
	
	// Bottom border
	bottomBorder := tmpl.BottomLeftCorner + strings.Repeat(tmpl.BottomBorder, maxLen+2) + tmpl.BottomRightCorner
	result = append(result, bottomBorder)
	
	return result
}

// stripANSI removes ANSI escape codes from a string for width calculation.
func stripANSI(s string) string {
	var result strings.Builder
	inEscape := false
	
	for _, r := range s {
		if r == '\x1b' {
			inEscape = true
			continue
		}
		if inEscape {
			if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') {
				inEscape = false
			}
			continue
		}
		result.WriteRune(r)
	}
	
	return result.String()
}
