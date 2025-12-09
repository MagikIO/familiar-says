package errors

import (
	"errors"
	"strings"
	"testing"
)

func TestCharacterLoadError(t *testing.T) {
	tests := []struct {
		name           string
		characterName  string
		err            error
		paths          []string
		expectedSubstr []string
	}{
		{
			name:          "simple error without paths",
			characterName: "missing",
			err:           errors.New("file not found"),
			paths:         nil,
			expectedSubstr: []string{
				"failed to load character",
				"missing",
				"file not found",
			},
		},
		{
			name:          "error with attempted paths",
			characterName: "custom",
			err:           errors.New("no such file"),
			paths:         []string{"custom.json", "characters/custom.json"},
			expectedSubstr: []string{
				"failed to load character",
				"custom",
				"tried paths",
				"custom.json",
				"characters/custom.json",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := NewCharacterLoadError(tt.characterName, tt.err, tt.paths...)
			errStr := err.Error()

			for _, substr := range tt.expectedSubstr {
				if !strings.Contains(errStr, substr) {
					t.Errorf("error message should contain %q, got: %s", substr, errStr)
				}
			}

			// Test Unwrap
			if unwrapped := errors.Unwrap(err); unwrapped != tt.err {
				t.Errorf("Unwrap() should return original error, got: %v", unwrapped)
			}
		})
	}
}

func TestColorParseError(t *testing.T) {
	tests := []struct {
		name           string
		input          string
		err            error
		expectedSubstr []string
	}{
		{
			name:  "invalid color format without underlying error",
			input: "notacolor",
			err:   nil,
			expectedSubstr: []string{
				"invalid color format",
				"notacolor",
			},
		},
		{
			name:  "color parse error with underlying error",
			input: "#GGGGGG",
			err:   errors.New("invalid hex"),
			expectedSubstr: []string{
				"failed to parse color",
				"#GGGGGG",
				"invalid hex",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := NewColorParseError(tt.input, tt.err)
			errStr := err.Error()

			for _, substr := range tt.expectedSubstr {
				if !strings.Contains(errStr, substr) {
					t.Errorf("error message should contain %q, got: %s", substr, errStr)
				}
			}

			// Test Unwrap
			if unwrapped := errors.Unwrap(err); unwrapped != tt.err {
				t.Errorf("Unwrap() should return original error, got: %v", unwrapped)
			}
		})
	}
}

func TestValidationError(t *testing.T) {
	tests := []struct {
		name           string
		field          string
		value          interface{}
		reason         string
		expectedSubstr []string
	}{
		{
			name:   "width validation",
			field:  "width",
			value:  -5,
			reason: "must be greater than 0",
			expectedSubstr: []string{
				"validation failed",
				"width",
				"-5",
				"must be greater than 0",
			},
		},
		{
			name:   "string validation",
			field:  "character name",
			value:  "",
			reason: "cannot be empty",
			expectedSubstr: []string{
				"validation failed",
				"character name",
				"cannot be empty",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := NewValidationError(tt.field, tt.value, tt.reason)
			errStr := err.Error()

			for _, substr := range tt.expectedSubstr {
				if !strings.Contains(errStr, substr) {
					t.Errorf("error message should contain %q, got: %s", substr, errStr)
				}
			}
		})
	}
}

func TestTerminalError(t *testing.T) {
	tests := []struct {
		name           string
		operation      string
		err            error
		expectedSubstr []string
	}{
		{
			name:      "width detection failure",
			operation: "get terminal width",
			err:       errors.New("not a terminal"),
			expectedSubstr: []string{
				"terminal operation",
				"get terminal width",
				"not a terminal",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := NewTerminalError(tt.operation, tt.err)
			errStr := err.Error()

			for _, substr := range tt.expectedSubstr {
				if !strings.Contains(errStr, substr) {
					t.Errorf("error message should contain %q, got: %s", substr, errStr)
				}
			}

			// Test Unwrap
			if unwrapped := errors.Unwrap(err); unwrapped != tt.err {
				t.Errorf("Unwrap() should return original error, got: %v", unwrapped)
			}
		})
	}
}

func TestSentinelErrors(t *testing.T) {
	tests := []struct {
		name string
		err  error
	}{
		{"ErrCharacterNotFound", ErrCharacterNotFound},
		{"ErrInvalidColorFormat", ErrInvalidColorFormat},
		{"ErrEmptyArt", ErrEmptyArt},
		{"ErrInvalidWidth", ErrInvalidWidth},
		{"ErrInvalidHeight", ErrInvalidHeight},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.err == nil {
				t.Errorf("sentinel error %s should not be nil", tt.name)
			}
			if tt.err.Error() == "" {
				t.Errorf("sentinel error %s should have a message", tt.name)
			}
		})
	}
}
