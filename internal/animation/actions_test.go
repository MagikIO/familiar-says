package animation

import (
	"testing"
)

func TestAllActions(t *testing.T) {
	actions := AllActions()

	if len(actions) == 0 {
		t.Error("AllActions should return at least one action")
	}

	// Check that ActionNone is included
	found := false
	for _, a := range actions {
		if a == ActionNone {
			found = true
			break
		}
	}
	if !found {
		t.Error("AllActions should include ActionNone")
	}
}

func TestGetActionDescription(t *testing.T) {
	tests := []struct {
		action      Action
		shouldExist bool
	}{
		{ActionNone, true},
		{ActionIdle, true},
		{ActionBlink, true},
		{ActionWave, true},
		{ActionJump, true},
		{ActionTailWag, true},
		{Action("nonexistent"), false},
	}

	for _, tc := range tests {
		desc := GetActionDescription(tc.action)
		if tc.shouldExist && desc == "Unknown action" {
			t.Errorf("expected description for %s, got 'Unknown action'", tc.action)
		}
		if !tc.shouldExist && desc != "Unknown action" {
			t.Errorf("expected 'Unknown action' for %s, got '%s'", tc.action, desc)
		}
	}
}

func TestValidateAction(t *testing.T) {
	tests := []struct {
		name     string
		expected bool
	}{
		{"none", true},
		{"idle", true},
		{"blink", true},
		{"wave", true},
		{"WAVE", true},       // Case insensitive
		{"Wave", true},       // Mixed case
		{"invalid", false},
		{"", false},
		{"foo", false},
	}

	for _, tc := range tests {
		result := ValidateAction(tc.name)
		if result != tc.expected {
			t.Errorf("ValidateAction(%q) = %v, expected %v", tc.name, result, tc.expected)
		}
	}
}

func TestParseAction(t *testing.T) {
	tests := []struct {
		name     string
		expected Action
	}{
		{"none", ActionNone},
		{"idle", ActionIdle},
		{"blink", ActionBlink},
		{"wave", ActionWave},
		{"WAVE", ActionWave},     // Case insensitive
		{"invalid", ActionNone},  // Falls back to None
		{"", ActionNone},
	}

	for _, tc := range tests {
		result := ParseAction(tc.name)
		if result != tc.expected {
			t.Errorf("ParseAction(%q) = %s, expected %s", tc.name, result, tc.expected)
		}
	}
}

func TestIsIdleAction(t *testing.T) {
	tests := []struct {
		action   Action
		expected bool
	}{
		{ActionIdle, true},
		{ActionBlink, true},
		{ActionBreathe, true},
		{ActionTailWag, true},
		{ActionWave, false},
		{ActionJump, false},
		{ActionNone, false},
	}

	for _, tc := range tests {
		result := IsIdleAction(tc.action)
		if result != tc.expected {
			t.Errorf("IsIdleAction(%s) = %v, expected %v", tc.action, result, tc.expected)
		}
	}
}

func TestIsTriggeredAction(t *testing.T) {
	tests := []struct {
		action   Action
		expected bool
	}{
		{ActionWave, true},
		{ActionJump, true},
		{ActionNod, true},
		{ActionHop, true},
		{ActionIdle, false},
		{ActionBlink, false},
		{ActionNone, false},
	}

	for _, tc := range tests {
		result := IsTriggeredAction(tc.action)
		if result != tc.expected {
			t.Errorf("IsTriggeredAction(%s) = %v, expected %v", tc.action, result, tc.expected)
		}
	}
}

func TestActionConstants(t *testing.T) {
	// Verify that action constants have expected string values
	tests := []struct {
		action   Action
		expected string
	}{
		{ActionNone, "none"},
		{ActionIdle, "idle"},
		{ActionBlink, "blink"},
		{ActionWave, "wave"},
		{ActionJump, "jump"},
		{ActionTailWag, "tail_wag"},
		{ActionNod, "nod"},
		{ActionBreathe, "breathe"},
	}

	for _, tc := range tests {
		if string(tc.action) != tc.expected {
			t.Errorf("Action constant %v should be %q, got %q", tc.action, tc.expected, string(tc.action))
		}
	}
}
