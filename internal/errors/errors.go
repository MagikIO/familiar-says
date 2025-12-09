package errors

import (
	"errors"
	"fmt"
)

// CharacterLoadError represents an error that occurred while loading a character
type CharacterLoadError struct {
	CharacterName string
	AttemptedPaths []string
	Err error
}

func (e *CharacterLoadError) Error() string {
	if len(e.AttemptedPaths) > 0 {
		return fmt.Sprintf("failed to load character %q: tried paths %v: %v", e.CharacterName, e.AttemptedPaths, e.Err)
	}
	return fmt.Sprintf("failed to load character %q: %v", e.CharacterName, e.Err)
}

func (e *CharacterLoadError) Unwrap() error {
	return e.Err
}

// NewCharacterLoadError creates a new CharacterLoadError
func NewCharacterLoadError(name string, err error, paths ...string) *CharacterLoadError {
	return &CharacterLoadError{
		CharacterName: name,
		AttemptedPaths: paths,
		Err: err,
	}
}

// ColorParseError represents an error that occurred while parsing a color
type ColorParseError struct {
	Input string
	Err error
}

func (e *ColorParseError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("failed to parse color %q: %v", e.Input, e.Err)
	}
	return fmt.Sprintf("invalid color format: %q", e.Input)
}

func (e *ColorParseError) Unwrap() error {
	return e.Err
}

// NewColorParseError creates a new ColorParseError
func NewColorParseError(input string, err error) *ColorParseError {
	return &ColorParseError{
		Input: input,
		Err: err,
	}
}

// ValidationError represents a validation failure
type ValidationError struct {
	Field string
	Value interface{}
	Reason string
}

func (e *ValidationError) Error() string {
	return fmt.Sprintf("validation failed for %s (value: %v): %s", e.Field, e.Value, e.Reason)
}

// NewValidationError creates a new ValidationError
func NewValidationError(field string, value interface{}, reason string) *ValidationError {
	return &ValidationError{
		Field: field,
		Value: value,
		Reason: reason,
	}
}

// TerminalError represents an error related to terminal operations
type TerminalError struct {
	Operation string
	Err error
}

func (e *TerminalError) Error() string {
	return fmt.Sprintf("terminal operation %q failed: %v", e.Operation, e.Err)
}

func (e *TerminalError) Unwrap() error {
	return e.Err
}

// NewTerminalError creates a new TerminalError
func NewTerminalError(operation string, err error) *TerminalError {
	return &TerminalError{
		Operation: operation,
		Err: err,
	}
}

// Common error types for specific scenarios
var (
	ErrCharacterNotFound = errors.New("character not found")
	ErrInvalidColorFormat = errors.New("invalid color format")
	ErrEmptyArt = errors.New("character has no art defined")
	ErrInvalidWidth = errors.New("width must be greater than 0")
	ErrInvalidHeight = errors.New("height must be greater than 0")
)
