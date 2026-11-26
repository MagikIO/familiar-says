package brain

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"time"
)

// Memory is what the familiar remembers between sessions.
//
// Several familiars can be alive at once (one per shell, plus a companion
// pane), all sharing one memory file. So Save never overwrites the file with
// its own view: it re-reads what is on disk and adds only what this process
// has learned since it last looked.
type Memory struct {
	Counts     map[string]int    `json:"counts,omitempty"`     // command line -> times seen
	Suggested  map[string]bool   `json:"suggested,omitempty"`  // command lines already given a shorthand tip
	Introduced map[string]int64  `json:"introduced,omitempty"` // project root -> unix time of the last briefing
	Trusted    map[string]string `json:"trusted,omitempty"`    // project rule file -> hash the user approved
	Born       int64             `json:"born,omitempty"`
	Commands   int               `json:"commands,omitempty"`
	Failures   int               `json:"failures,omitempty"`
	Fixes      int               `json:"fixes,omitempty"` // failures it saw turned into successes
	Pets       int               `json:"pets,omitempty"`
	Days       int               `json:"days,omitempty"`     // distinct days spent together
	LastDay    string            `json:"last_day,omitempty"` // the most recent of them, YYYY-MM-DD
	Level      int               `json:"level,omitempty"`    // highest bond level already celebrated
	Wearing    string            `json:"wearing,omitempty"`  // accessory it has on
	Tip        int               `json:"tip,omitempty"`      // next tip to hand out
	Hinted     int               `json:"hinted,omitempty"`   // times the accept-key hint has been shown

	path string
	base *Memory // the state this process last saw on disk
}

// MemoryPath is where memory lives on disk.
func MemoryPath() string {
	dir := os.Getenv("XDG_STATE_HOME")
	if dir == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return ""
		}
		dir = filepath.Join(home, ".local", "state")
	}
	return filepath.Join(dir, "familiar-says", "memory.json")
}

func readMemory(path string) *Memory {
	m := &Memory{}
	if path != "" {
		if data, err := os.ReadFile(path); err == nil {
			_ = json.Unmarshal(data, m)
		}
	}
	m.init()
	return m
}

func (m *Memory) init() {
	if m.Counts == nil {
		m.Counts = map[string]int{}
	}
	if m.Suggested == nil {
		m.Suggested = map[string]bool{}
	}
	if m.Introduced == nil {
		m.Introduced = map[string]int64{}
	}
	if m.Trusted == nil {
		m.Trusted = map[string]string{}
	}
}

// LoadMemory reads memory from path. A missing or unreadable file yields a
// blank memory; pass "" for one that is never written to disk.
func LoadMemory(path string) *Memory {
	m := readMemory(path)
	m.path = path
	if m.Born == 0 {
		m.Born = time.Now().Unix()
	}
	m.base = m.clone()
	return m
}

func (m *Memory) clone() *Memory {
	c := *m
	c.base = nil
	c.Counts = make(map[string]int, len(m.Counts))
	for k, v := range m.Counts {
		c.Counts[k] = v
	}
	c.Suggested = make(map[string]bool, len(m.Suggested))
	for k, v := range m.Suggested {
		c.Suggested[k] = v
	}
	c.Introduced = make(map[string]int64, len(m.Introduced))
	for k, v := range m.Introduced {
		c.Introduced[k] = v
	}
	c.Trusted = make(map[string]string, len(m.Trusted))
	for k, v := range m.Trusted {
		c.Trusted[k] = v
	}
	return &c
}

// absorb folds in whatever other familiars have written to disk since this
// process last looked, keeping everything this process has learned itself.
func (m *Memory) absorb(disk *Memory) {
	base := m.base
	if base == nil {
		base = &Memory{}
		base.init()
	}

	// Tallies: what is on disk, plus what we added.
	m.Commands = disk.Commands + (m.Commands - base.Commands)
	m.Failures = disk.Failures + (m.Failures - base.Failures)
	m.Fixes = disk.Fixes + (m.Fixes - base.Fixes)
	m.Pets = disk.Pets + (m.Pets - base.Pets)
	counts := make(map[string]int, len(disk.Counts))
	for k, v := range disk.Counts {
		counts[k] = v
	}
	for k, v := range m.Counts {
		if d := v - base.Counts[k]; d > 0 {
			counts[k] += d
		}
	}
	m.Counts = counts

	// Facts that only ever accumulate.
	for k, v := range disk.Suggested {
		m.Suggested[k] = m.Suggested[k] || v
	}
	for k, v := range disk.Introduced {
		if v > m.Introduced[k] {
			m.Introduced[k] = v
		}
	}
	m.Level = max(m.Level, disk.Level)
	m.Tip = max(m.Tip, disk.Tip)
	m.Hinted = max(m.Hinted, disk.Hinted)
	if disk.Born != 0 && (m.Born == 0 || disk.Born < m.Born) {
		m.Born = disk.Born
	}
	// Two familiars that both notice the day has changed must not count it twice.
	if disk.LastDay > m.LastDay || (disk.LastDay == m.LastDay && disk.Days > m.Days) {
		m.Days, m.LastDay = disk.Days, disk.LastDay
	}

	// Choices: ours if we changed it, otherwise whatever is current.
	if m.Wearing == base.Wearing {
		m.Wearing = disk.Wearing
	}
	trusted := make(map[string]string, len(disk.Trusted))
	for k, v := range disk.Trusted {
		trusted[k] = v
	}
	for k, v := range m.Trusted {
		if base.Trusted[k] != v {
			trusted[k] = v
		}
	}
	for k := range base.Trusted {
		if _, still := m.Trusted[k]; !still {
			delete(trusted, k)
		}
	}
	m.Trusted = trusted
}

// Save merges this familiar's memories into the file on disk.
func (m *Memory) Save() error {
	if m.path == "" {
		return nil
	}
	m.absorb(readMemory(m.path))
	m.prune()
	data, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(m.path), 0o700); err != nil {
		return err
	}
	tmp := m.path + "." + filepath.Base(os.Args[0]) + ".tmp"
	if err := os.WriteFile(tmp, data, 0o600); err != nil {
		return err
	}
	if err := os.Rename(tmp, m.path); err != nil {
		return err
	}
	m.base = m.clone()
	return nil
}

// Today notes that the familiar and its person spent time together on the
// day of now, and reports whether that is news.
func (m *Memory) Today(now time.Time) bool {
	day := now.Format("2006-01-02")
	if day == m.LastDay {
		return false
	}
	m.LastDay = day
	m.Days++
	return true
}

// prune keeps the command tally from growing without bound by forgetting the
// least-used entries.
func (m *Memory) prune() {
	const limit = 300
	if len(m.Counts) <= limit {
		return
	}
	type kv struct {
		k string
		v int
	}
	all := make([]kv, 0, len(m.Counts))
	for k, v := range m.Counts {
		all = append(all, kv{k, v})
	}
	sort.Slice(all, func(i, j int) bool {
		if all[i].v != all[j].v {
			return all[i].v > all[j].v
		}
		return all[i].k < all[j].k
	})
	m.Counts = make(map[string]int, limit)
	for _, e := range all[:limit] {
		m.Counts[e.k] = e.v
	}
}
