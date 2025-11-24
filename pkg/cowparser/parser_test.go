package cowparser

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParse(t *testing.T) {
	// Create a temporary .cow file for testing
	tmpDir := t.TempDir()
	cowFile := filepath.Join(tmpDir, "test.cow")

	content := `## Test cow
$the_cow = <<EOC;
        $thoughts   ^__^
         $thoughts  ($eyes)\_______
            (__)\\       )/\\
             $tongue ||----w |
                ||     ||
EOC
`

	if err := os.WriteFile(cowFile, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	cow, err := Parse(cowFile)
	if err != nil {
		t.Fatalf("Failed to parse cow file: %v", err)
	}

	if len(cow.Body) == 0 {
		t.Error("Expected non-empty body")
	}

	if cow.Thoughts != "\\" {
		t.Errorf("Expected thoughts to be '\\', got '%s'", cow.Thoughts)
	}
}

func TestReplaceVariables(t *testing.T) {
	cow := &CowFile{
		Eyes:     "oo",
		Tongue:   "  ",
		Thoughts: "\\",
		Body: []string{
			"$thoughts test",
			"$eyes test",
			"$tongue test",
		},
		Variables: make(map[string]string),
	}

	result := cow.ReplaceVariables("^^", "U ")

	if len(result) != 3 {
		t.Errorf("Expected 3 lines, got %d", len(result))
	}

	if result[0] != "\\ test" {
		t.Errorf("Expected '\\ test', got '%s'", result[0])
	}

	if result[1] != "^^ test" {
		t.Errorf("Expected '^^ test', got '%s'", result[1])
	}

	if result[2] != "U  test" {
		t.Errorf("Expected 'U  test', got '%s'", result[2])
	}
}

func TestParseNonExistentFile(t *testing.T) {
	_, err := Parse("nonexistent.cow")
	if err == nil {
		t.Error("Expected error for non-existent file")
	}
}
