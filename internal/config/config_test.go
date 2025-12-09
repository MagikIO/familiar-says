package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadFromPath(t *testing.T) {
	tests := []struct {
		name       string
		configJSON string
		wantErr    bool
		validate   func(*testing.T, *Config)
	}{
		{
			name: "valid config with default only",
			configJSON: `{
				"default": {
					"character": "cat",
					"theme": "cyber",
					"mood": "happy"
				}
			}`,
			wantErr: false,
			validate: func(t *testing.T, cfg *Config) {
				if cfg.Default.Character == nil || *cfg.Default.Character != "cat" {
					t.Errorf("Character = %v, want cat", cfg.Default.Character)
				}
				if cfg.Default.Theme == nil || *cfg.Default.Theme != "cyber" {
					t.Errorf("Theme = %v, want cyber", cfg.Default.Theme)
				}
				if cfg.Default.Mood == nil || *cfg.Default.Mood != "happy" {
					t.Errorf("Mood = %v, want happy", cfg.Default.Mood)
				}
			},
		},
		{
			name: "valid config with profiles",
			configJSON: `{
				"default": {
					"character": "cat",
					"width": 40
				},
				"profiles": {
					"work": {
						"character": "owl",
						"theme": "cyber"
					}
				}
			}`,
			wantErr: false,
			validate: func(t *testing.T, cfg *Config) {
				if cfg.Default.Character == nil || *cfg.Default.Character != "cat" {
					t.Errorf("Default character = %v, want cat", cfg.Default.Character)
				}
				if cfg.Profiles == nil {
					t.Fatal("Profiles is nil")
				}
				work, exists := cfg.Profiles["work"]
				if !exists {
					t.Fatal("Profile 'work' not found")
				}
				if work.Character == nil || *work.Character != "owl" {
					t.Errorf("Work profile character = %v, want owl", work.Character)
				}
			},
		},
		{
			name: "valid config with all fields",
			configJSON: `{
				"default": {
					"character": "cat",
					"theme": "rainbow",
					"mood": "excited",
					"width": 60,
					"animate": true,
					"speed": 30,
					"effect": "sparkle",
					"think": false,
					"multipanel": true,
					"outlineColor": "#FF6B6B",
					"eyeColor": "blue",
					"mouthColor": "pink"
				}
			}`,
			wantErr: false,
			validate: func(t *testing.T, cfg *Config) {
				if cfg.Default.Character == nil || *cfg.Default.Character != "cat" {
					t.Errorf("Character = %v, want cat", cfg.Default.Character)
				}
				if cfg.Default.Width == nil || *cfg.Default.Width != 60 {
					t.Errorf("Width = %v, want 60", cfg.Default.Width)
				}
				if cfg.Default.Animate == nil || *cfg.Default.Animate != true {
					t.Errorf("Animate = %v, want true", cfg.Default.Animate)
				}
				if cfg.Default.Speed == nil || *cfg.Default.Speed != 30 {
					t.Errorf("Speed = %v, want 30", cfg.Default.Speed)
				}
				if cfg.Default.OutlineColor == nil || *cfg.Default.OutlineColor != "#FF6B6B" {
					t.Errorf("OutlineColor = %v, want #FF6B6B", cfg.Default.OutlineColor)
				}
			},
		},
		{
			name:       "invalid JSON",
			configJSON: `{"default": {invalid json}`,
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			configPath := filepath.Join(tmpDir, "config.json")

			if tt.configJSON != "" {
				if err := os.WriteFile(configPath, []byte(tt.configJSON), 0644); err != nil {
					t.Fatalf("Failed to write test config: %v", err)
				}
			}

			cfg, err := LoadFromPath(configPath)
			if (err != nil) != tt.wantErr {
				t.Errorf("LoadFromPath() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && tt.validate != nil {
				tt.validate(t, cfg)
			}
		})
	}
}

func TestLoadFromPath_NonExistentFile(t *testing.T) {
	tmpDir := t.TempDir()
	nonExistentPath := filepath.Join(tmpDir, "nonexistent.json")

	cfg, err := LoadFromPath(nonExistentPath)
	if err != nil {
		t.Errorf("LoadFromPath() should not error on non-existent file, got %v", err)
	}
	if cfg != nil {
		t.Errorf("LoadFromPath() should return nil for non-existent file, got %v", cfg)
	}
}

func TestGetEffectiveConfig(t *testing.T) {
	tests := []struct {
		name        string
		config      Config
		profileName string
		validate    func(*testing.T, *FlagConfig)
	}{
		{
			name: "default only (no profile)",
			config: Config{
				Default: FlagConfig{
					Character: stringPtr("cat"),
					Theme:     stringPtr("default"),
				},
			},
			profileName: "",
			validate: func(t *testing.T, cfg *FlagConfig) {
				if cfg.Character == nil || *cfg.Character != "cat" {
					t.Errorf("Character = %v, want cat", cfg.Character)
				}
				if cfg.Theme == nil || *cfg.Theme != "default" {
					t.Errorf("Theme = %v, want default", cfg.Theme)
				}
			},
		},
		{
			name: "profile overrides default",
			config: Config{
				Default: FlagConfig{
					Character: stringPtr("cat"),
					Theme:     stringPtr("default"),
					Width:     intPtr(40),
				},
				Profiles: map[string]FlagConfig{
					"work": {
						Character: stringPtr("owl"),
						Theme:     stringPtr("cyber"),
					},
				},
			},
			profileName: "work",
			validate: func(t *testing.T, cfg *FlagConfig) {
				if cfg.Character == nil || *cfg.Character != "owl" {
					t.Errorf("Character = %v, want owl (from profile)", cfg.Character)
				}
				if cfg.Theme == nil || *cfg.Theme != "cyber" {
					t.Errorf("Theme = %v, want cyber (from profile)", cfg.Theme)
				}
				if cfg.Width == nil || *cfg.Width != 40 {
					t.Errorf("Width = %v, want 40 (from default)", cfg.Width)
				}
			},
		},
		{
			name: "non-existent profile uses default",
			config: Config{
				Default: FlagConfig{
					Character: stringPtr("cat"),
				},
				Profiles: map[string]FlagConfig{
					"work": {
						Character: stringPtr("owl"),
					},
				},
			},
			profileName: "nonexistent",
			validate: func(t *testing.T, cfg *FlagConfig) {
				if cfg.Character == nil || *cfg.Character != "cat" {
					t.Errorf("Character = %v, want cat (default for non-existent profile)", cfg.Character)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			effective := tt.config.GetEffectiveConfig(tt.profileName)
			if tt.validate != nil {
				tt.validate(t, effective)
			}
		})
	}
}

func TestMerge(t *testing.T) {
	tests := []struct {
		name     string
		base     FlagConfig
		override FlagConfig
		validate func(*testing.T, *FlagConfig)
	}{
		{
			name: "override non-nil values",
			base: FlagConfig{
				Character: stringPtr("cat"),
				Theme:     stringPtr("default"),
			},
			override: FlagConfig{
				Character: stringPtr("owl"),
				Mood:      stringPtr("happy"),
			},
			validate: func(t *testing.T, result *FlagConfig) {
				if result.Character == nil || *result.Character != "owl" {
					t.Errorf("Character = %v, want owl (overridden)", result.Character)
				}
				if result.Theme == nil || *result.Theme != "default" {
					t.Errorf("Theme = %v, want default (from base)", result.Theme)
				}
				if result.Mood == nil || *result.Mood != "happy" {
					t.Errorf("Mood = %v, want happy (from override)", result.Mood)
				}
			},
		},
		{
			name: "nil values don't override",
			base: FlagConfig{
				Character: stringPtr("cat"),
				Width:     intPtr(40),
			},
			override: FlagConfig{
				Theme: stringPtr("cyber"),
			},
			validate: func(t *testing.T, result *FlagConfig) {
				if result.Character == nil || *result.Character != "cat" {
					t.Errorf("Character = %v, want cat (unchanged)", result.Character)
				}
				if result.Width == nil || *result.Width != 40 {
					t.Errorf("Width = %v, want 40 (unchanged)", result.Width)
				}
				if result.Theme == nil || *result.Theme != "cyber" {
					t.Errorf("Theme = %v, want cyber (from override)", result.Theme)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			base := tt.base
			Merge(&base, &tt.override)
			if tt.validate != nil {
				tt.validate(t, &base)
			}
		})
	}
}

func TestLoadFromEnv(t *testing.T) {
	tests := []struct {
		name     string
		envVars  map[string]string
		validate func(*testing.T, *FlagConfig)
	}{
		{
			name: "string values",
			envVars: map[string]string{
				"FAMILIAR_SAYS_CHARACTER": "owl",
				"FAMILIAR_SAYS_THEME":     "cyber",
				"FAMILIAR_SAYS_MOOD":      "happy",
			},
			validate: func(t *testing.T, cfg *FlagConfig) {
				if cfg.Character == nil || *cfg.Character != "owl" {
					t.Errorf("Character = %v, want owl", cfg.Character)
				}
				if cfg.Theme == nil || *cfg.Theme != "cyber" {
					t.Errorf("Theme = %v, want cyber", cfg.Theme)
				}
				if cfg.Mood == nil || *cfg.Mood != "happy" {
					t.Errorf("Mood = %v, want happy", cfg.Mood)
				}
			},
		},
		{
			name: "integer values",
			envVars: map[string]string{
				"FAMILIAR_SAYS_WIDTH": "60",
				"FAMILIAR_SAYS_SPEED": "30",
			},
			validate: func(t *testing.T, cfg *FlagConfig) {
				if cfg.Width == nil || *cfg.Width != 60 {
					t.Errorf("Width = %v, want 60", cfg.Width)
				}
				if cfg.Speed == nil || *cfg.Speed != 30 {
					t.Errorf("Speed = %v, want 30", cfg.Speed)
				}
			},
		},
		{
			name: "boolean values - various formats",
			envVars: map[string]string{
				"FAMILIAR_SAYS_ANIMATE":    "true",
				"FAMILIAR_SAYS_THINK":      "1",
				"FAMILIAR_SAYS_MULTIPANEL": "yes",
			},
			validate: func(t *testing.T, cfg *FlagConfig) {
				if cfg.Animate == nil || *cfg.Animate != true {
					t.Errorf("Animate = %v, want true", cfg.Animate)
				}
				if cfg.Think == nil || *cfg.Think != true {
					t.Errorf("Think = %v, want true", cfg.Think)
				}
				if cfg.Multipanel == nil || *cfg.Multipanel != true {
					t.Errorf("Multipanel = %v, want true", cfg.Multipanel)
				}
			},
		},
		{
			name: "boolean false values",
			envVars: map[string]string{
				"FAMILIAR_SAYS_ANIMATE": "false",
				"FAMILIAR_SAYS_THINK":   "0",
			},
			validate: func(t *testing.T, cfg *FlagConfig) {
				if cfg.Animate == nil || *cfg.Animate != false {
					t.Errorf("Animate = %v, want false", cfg.Animate)
				}
				if cfg.Think == nil || *cfg.Think != false {
					t.Errorf("Think = %v, want false", cfg.Think)
				}
			},
		},
		{
			name: "color values",
			envVars: map[string]string{
				"FAMILIAR_SAYS_OUTLINE_COLOR": "#FF6B6B",
				"FAMILIAR_SAYS_EYE_COLOR":     "blue",
				"FAMILIAR_SAYS_MOUTH_COLOR":   "pink",
			},
			validate: func(t *testing.T, cfg *FlagConfig) {
				if cfg.OutlineColor == nil || *cfg.OutlineColor != "#FF6B6B" {
					t.Errorf("OutlineColor = %v, want #FF6B6B", cfg.OutlineColor)
				}
				if cfg.EyeColor == nil || *cfg.EyeColor != "blue" {
					t.Errorf("EyeColor = %v, want blue", cfg.EyeColor)
				}
				if cfg.MouthColor == nil || *cfg.MouthColor != "pink" {
					t.Errorf("MouthColor = %v, want pink", cfg.MouthColor)
				}
			},
		},
		{
			name: "invalid integer ignored",
			envVars: map[string]string{
				"FAMILIAR_SAYS_WIDTH": "invalid",
			},
			validate: func(t *testing.T, cfg *FlagConfig) {
				if cfg.Width != nil {
					t.Errorf("Width should be nil for invalid value, got %v", cfg.Width)
				}
			},
		},
		{
			name: "invalid boolean ignored",
			envVars: map[string]string{
				"FAMILIAR_SAYS_ANIMATE": "maybe",
			},
			validate: func(t *testing.T, cfg *FlagConfig) {
				if cfg.Animate != nil {
					t.Errorf("Animate should be nil for invalid value, got %v", cfg.Animate)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clear environment first
			clearEnv()

			// Set test environment variables
			for key, val := range tt.envVars {
				os.Setenv(key, val)
			}

			// Clean up after test
			defer clearEnv()

			cfg := LoadFromEnv()
			if tt.validate != nil {
				tt.validate(t, cfg)
			}
		})
	}
}

func TestParseBool(t *testing.T) {
	tests := []struct {
		input   string
		want    bool
		wantOk  bool
	}{
		{"true", true, true},
		{"True", true, true},
		{"TRUE", true, true},
		{"1", true, true},
		{"yes", true, true},
		{"Yes", true, true},
		{"false", false, true},
		{"False", false, true},
		{"FALSE", false, true},
		{"0", false, true},
		{"no", false, true},
		{"No", false, true},
		{"invalid", false, false},
		{"maybe", false, false},
		{"", false, false},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got, ok := parseBool(tt.input)
			if ok != tt.wantOk {
				t.Errorf("parseBool(%q) ok = %v, want %v", tt.input, ok, tt.wantOk)
			}
			if ok && got != tt.want {
				t.Errorf("parseBool(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

// Helper to clear all FAMILIAR_SAYS_* environment variables
func clearEnv() {
	envVars := []string{
		"FAMILIAR_SAYS_CHARACTER",
		"FAMILIAR_SAYS_THEME",
		"FAMILIAR_SAYS_MOOD",
		"FAMILIAR_SAYS_WIDTH",
		"FAMILIAR_SAYS_ANIMATE",
		"FAMILIAR_SAYS_SPEED",
		"FAMILIAR_SAYS_EFFECT",
		"FAMILIAR_SAYS_THINK",
		"FAMILIAR_SAYS_MULTIPANEL",
		"FAMILIAR_SAYS_OUTLINE_COLOR",
		"FAMILIAR_SAYS_EYE_COLOR",
		"FAMILIAR_SAYS_MOUTH_COLOR",
	}
	for _, v := range envVars {
		os.Unsetenv(v)
	}
}
