package effects

import (
	"testing"
)

func TestAllEffects(t *testing.T) {
	effects := AllEffects()
	
	if len(effects) == 0 {
		t.Error("Expected at least one effect")
	}
	
	// Check that EffectNone is included
	found := false
	for _, e := range effects {
		if e == EffectNone {
			found = true
			break
		}
	}
	
	if !found {
		t.Error("Expected EffectNone to be in the list")
	}
}

func TestGetEffectDescription(t *testing.T) {
	tests := []struct {
		effect      Effect
		shouldExist bool
	}{
		{EffectNone, true},
		{EffectConfetti, true},
		{EffectFireworks, true},
		{EffectSparkle, true},
		{EffectRainbow, true},
		{Effect("unknown"), false},
	}
	
	for _, tt := range tests {
		desc := GetEffectDescription(tt.effect)
		if tt.shouldExist && desc == "Unknown effect" {
			t.Errorf("Expected description for effect %s", tt.effect)
		}
		if !tt.shouldExist && desc != "Unknown effect" {
			t.Errorf("Expected 'Unknown effect' for %s, got '%s'", tt.effect, desc)
		}
	}
}

func TestApplyNone(t *testing.T) {
	content := []string{"line1", "line2"}
	result := Apply(content, EffectNone)
	
	if len(result) != len(content) {
		t.Error("EffectNone should not change the number of lines")
	}
	
	for i, line := range result {
		if line != content[i] {
			t.Errorf("EffectNone should not modify content, line %d changed", i)
		}
	}
}

func TestApplyConfetti(t *testing.T) {
	content := []string{"test"}
	result := Apply(content, EffectConfetti)
	
	// Confetti should add header and footer
	if len(result) <= len(content) {
		t.Error("Expected confetti to add lines")
	}
}

func TestApplyFireworks(t *testing.T) {
	content := []string{"test"}
	result := Apply(content, EffectFireworks)
	
	// Fireworks should add lines above and below
	if len(result) <= len(content) {
		t.Error("Expected fireworks to add lines")
	}
}

func TestApplySparkle(t *testing.T) {
	content := []string{"test"}
	result := Apply(content, EffectSparkle)
	
	// Sparkle should return at least the same number of lines
	if len(result) < len(content) {
		t.Error("Expected sparkle to maintain or increase line count")
	}
}

func TestApplyRainbow(t *testing.T) {
	content := []string{"test"}
	result := Apply(content, EffectRainbow)
	
	// Rainbow should maintain the same number of lines
	if len(result) != len(content) {
		t.Error("Expected rainbow to maintain line count")
	}
	
	// Result should exist
	if len(result) == 0 {
		t.Error("Expected non-empty result from rainbow effect")
	}
}
