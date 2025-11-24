package cowparser

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

// CowFile represents a parsed .cow character file
type CowFile struct {
	Eyes      string
	Tongue    string
	Thoughts  string
	Body      []string
	Variables map[string]string
}

// Parse reads and parses a .cow file
func Parse(filename string) (*CowFile, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	cow := &CowFile{
		Eyes:      "oo",
		Tongue:    "  ",
		Thoughts:  "\\",
		Body:      []string{},
		Variables: make(map[string]string),
	}

	scanner := bufio.NewScanner(file)
	inBody := false
	bodyBuilder := strings.Builder{}

	for scanner.Scan() {
		line := scanner.Text()

		// Start of heredoc
		if strings.Contains(line, "<<") {
			inBody = true
			continue
		}

		// End of heredoc
		if inBody && strings.HasPrefix(strings.TrimSpace(line), "EOC") {
			inBody = false
			cow.Body = strings.Split(bodyBuilder.String(), "\n")
			continue
		}

		// Collect body lines
		if inBody {
			bodyBuilder.WriteString(line + "\n")
			continue
		}

		// Parse variable assignments
		if strings.Contains(line, "=") && !strings.HasPrefix(strings.TrimSpace(line), "#") {
			parts := strings.SplitN(line, "=", 2)
			if len(parts) == 2 {
				key := strings.TrimSpace(parts[0])
				value := strings.Trim(strings.TrimSpace(parts[1]), `"'`)
				cow.Variables[key] = value

				// Handle special variables
				switch key {
				case "$eyes":
					cow.Eyes = value
				case "$tongue":
					cow.Tongue = value
				case "$thoughts":
					cow.Thoughts = value
				}
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading file: %w", err)
	}

	return cow, nil
}

// ReplaceVariables replaces special variables in the body
func (c *CowFile) ReplaceVariables(eyes, tongue string) []string {
	result := make([]string, len(c.Body))
	
	for i, line := range c.Body {
		replaced := line
		replaced = strings.ReplaceAll(replaced, "$eyes", eyes)
		replaced = strings.ReplaceAll(replaced, "$thoughts", c.Thoughts)
		replaced = strings.ReplaceAll(replaced, "$tongue", tongue)
		
		// Replace any custom variables
		for key, value := range c.Variables {
			replaced = strings.ReplaceAll(replaced, key, value)
		}
		
		result[i] = replaced
	}
	
	return result
}
