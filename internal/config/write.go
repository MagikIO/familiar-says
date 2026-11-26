package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// SetCharacter records name as the user's familiar in the config file at
// path, creating the file if need be. It sets default.character, and also
// live.character when the file already pins one, so the choice takes effect
// everywhere. Settings it does not understand are carried over untouched.
func SetCharacter(path, name string) error {
	root := map[string]json.RawMessage{}
	data, err := os.ReadFile(path)
	switch {
	case err == nil:
		if err := json.Unmarshal(data, &root); err != nil {
			return fmt.Errorf("%s is not valid JSON, so it was left alone: %w", path, err)
		}
	case !os.IsNotExist(err):
		return err
	}

	value, _ := json.Marshal(name)
	set := func(section string, onlyIfPinned bool) error {
		fields := map[string]json.RawMessage{}
		if raw, ok := root[section]; ok {
			if err := json.Unmarshal(raw, &fields); err != nil {
				return fmt.Errorf("%q in %s is not an object, so it was left alone: %w", section, path, err)
			}
		}
		if _, pinned := fields["character"]; onlyIfPinned && !pinned {
			return nil
		}
		fields["character"] = value
		raw, err := json.Marshal(fields)
		if err != nil {
			return err
		}
		root[section] = raw
		return nil
	}
	if err := set("default", false); err != nil {
		return err
	}
	if err := set("live", true); err != nil {
		return err
	}

	out, err := json.MarshalIndent(root, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, append(out, '\n'), 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}
