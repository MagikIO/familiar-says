package canvas

import (
	"embed"
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
)

// embeddedCharacters holds the embedded filesystem set by SetEmbeddedFS.
var embeddedCharacters embed.FS

// embeddedFSSet tracks whether the embedded FS has been set.
var embeddedFSSet bool

// characterDirPrefix is the directory prefix for character files in the embedded FS.
var characterDirPrefix = "characters"

// SetEmbeddedFS sets the embedded filesystem containing character JSON files.
// This must be called during initialization from the main package.
func SetEmbeddedFS(fs embed.FS) {
	SetEmbeddedFSWithPrefix(fs, "characters")
}

// SetEmbeddedFSWithPrefix sets the embedded filesystem with a custom directory prefix.
// This is useful for tests that have a different directory structure.
func SetEmbeddedFSWithPrefix(fs embed.FS, prefix string) {
	embeddedCharacters = fs
	embeddedFSSet = true
	characterDirPrefix = prefix
	// Clear any cached data when FS changes
	ClearCharacterCache()
	characterListOnce = sync.Once{}
}

// characterCache caches loaded characters to avoid re-parsing JSON
var (
	characterCache     = make(map[string]*Character)
	characterCacheMu   sync.RWMutex
	characterListCache []string
	characterListOnce  sync.Once
)

// findCharactersDir searches for the characters directory by walking up from cwd.
// This is used as a fallback when embedded FS is not available (e.g., during tests).
func findCharactersDir() (string, bool) {
	dir, err := os.Getwd()
	if err != nil {
		return "", false
	}

	for {
		charDir := filepath.Join(dir, "characters")
		if info, err := os.Stat(charDir); err == nil && info.IsDir() {
			return charDir, true
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", false
		}
		dir = parent
	}
}

// loadEmbeddedCharacter loads a character from the embedded filesystem,
// or falls back to file-based loading if embedded FS is not set.
func loadEmbeddedCharacter(name string) (*Character, error) {
	// Check cache first
	characterCacheMu.RLock()
	if char, ok := characterCache[name]; ok {
		characterCacheMu.RUnlock()
		return char, nil
	}
	characterCacheMu.RUnlock()

	var data []byte
	var err error

	if embeddedFSSet {
		// Load from embedded FS
		filename := characterDirPrefix + "/" + name + ".json"
		data, err = embeddedCharacters.ReadFile(filename)
	} else {
		// Fallback: load from filesystem (useful for tests)
		charDir, found := findCharactersDir()
		if !found {
			return nil, os.ErrNotExist
		}
		filename := filepath.Join(charDir, name+".json")
		data, err = os.ReadFile(filename)
	}

	if err != nil {
		return nil, err
	}

	var char Character
	if err := json.Unmarshal(data, &char); err != nil {
		return nil, err
	}

	// Set name from filename if not specified
	if char.Name == "" {
		char.Name = name
	}

	// Cache the character
	characterCacheMu.Lock()
	characterCache[name] = &char
	characterCacheMu.Unlock()

	return &char, nil
}

// listEmbeddedCharacters returns a sorted list of all embedded character names.
func listEmbeddedCharacters() []string {
	characterListOnce.Do(func() {
		var entries []os.DirEntry
		var err error

		if embeddedFSSet {
			entries, err = embeddedCharacters.ReadDir(characterDirPrefix)
		} else {
			// Fallback: read from filesystem
			charDir, found := findCharactersDir()
			if !found {
				return
			}
			entries, err = os.ReadDir(charDir)
		}

		if err != nil {
			return
		}

		for _, entry := range entries {
			if entry.IsDir() {
				continue
			}
			name := entry.Name()
			if strings.HasSuffix(name, ".json") {
				charName := strings.TrimSuffix(name, ".json")
				characterListCache = append(characterListCache, charName)
			}
		}
		sort.Strings(characterListCache)
	})

	return characterListCache
}

// GetEmbeddedCharacter returns an embedded character by name.
func GetEmbeddedCharacter(name string) (*Character, bool) {
	name = strings.ToLower(strings.TrimSpace(name))
	char, err := loadEmbeddedCharacter(name)
	if err != nil {
		return nil, false
	}
	return char, true
}

// ListEmbeddedCharacters returns the names of all embedded characters.
func ListEmbeddedCharacters() []string {
	return listEmbeddedCharacters()
}

// ClearCharacterCache clears the character cache (useful for testing).
func ClearCharacterCache() {
	characterCacheMu.Lock()
	characterCache = make(map[string]*Character)
	characterCacheMu.Unlock()
}

// ResetCharacterList resets the character list cache (useful for testing).
func ResetCharacterList() {
	characterListCache = nil
	characterListOnce = sync.Once{}
}
