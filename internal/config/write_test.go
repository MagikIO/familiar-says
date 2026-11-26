package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSetCharacterCreatesConfig(t *testing.T) {
	path := filepath.Join(t.TempDir(), "nested", "config.json")
	if err := SetCharacter(path, "owl"); err != nil {
		t.Fatal(err)
	}
	cfg, err := LoadFromPath(path)
	if err != nil || cfg == nil {
		t.Fatalf("written config should load: %v", err)
	}
	if cfg.Default.Character == nil || *cfg.Default.Character != "owl" {
		t.Errorf("default.character should be owl, got %v", cfg.Default.Character)
	}
	if cfg.Live != nil {
		t.Error("a live section should not be invented")
	}
}

func TestSetCharacterKeepsEverythingElse(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.json")
	os.WriteFile(path, []byte(`{
  "default": {"character": "cat", "theme": "cyber", "width": 50},
  "profiles": {"work": {"character": "owl"}},
  "live": {"character": "cat", "talkativeness": 0.2, "oracle": {"command": ["claude", "-p"]}},
  "somethingFromTheFuture": {"keep": true}
}`), 0o644)

	if err := SetCharacter(path, "dragon"); err != nil {
		t.Fatal(err)
	}
	cfg, err := LoadFromPath(path)
	if err != nil {
		t.Fatal(err)
	}
	if *cfg.Default.Character != "dragon" || *cfg.Live.Character != "dragon" {
		t.Errorf("both pinned characters should change: default=%q live=%q", *cfg.Default.Character, *cfg.Live.Character)
	}
	if *cfg.Default.Theme != "cyber" || *cfg.Default.Width != 50 || *cfg.Live.Talkativeness != 0.2 ||
		cfg.Live.Oracle.Command[0] != "claude" || *cfg.Profiles["work"].Character != "owl" {
		t.Errorf("other settings must survive: %+v", cfg)
	}
	raw, _ := os.ReadFile(path)
	if !strings.Contains(string(raw), "somethingFromTheFuture") {
		t.Error("unknown settings must be carried over")
	}
}

func TestSetCharacterLeavesUnpinnedLiveAlone(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.json")
	os.WriteFile(path, []byte(`{"live": {"tips": false}}`), 0o644)
	if err := SetCharacter(path, "fox"); err != nil {
		t.Fatal(err)
	}
	cfg, _ := LoadFromPath(path)
	if cfg.Live.Character != nil {
		t.Error("live.character should only be updated when it was already set")
	}
	if *cfg.Default.Character != "fox" {
		t.Error("default.character should be set")
	}
}

func TestSetCharacterRefusesToClobberBrokenConfig(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.json")
	broken := `{"default": {"character": "cat",}`
	os.WriteFile(path, []byte(broken), 0o644)
	if err := SetCharacter(path, "owl"); err == nil {
		t.Fatal("a config that does not parse must not be overwritten")
	}
	if raw, _ := os.ReadFile(path); string(raw) != broken {
		t.Error("the broken file should be untouched")
	}
}
