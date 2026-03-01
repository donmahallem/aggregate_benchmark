package benchagg

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type Measurement struct {
	Language    string `json:"language"`
	GroupKey    string `json:"group_key,omitempty"`
	Duration    string `json:"duration"`
	Iterations  int    `json:"iterations,omitempty"`
	Day         int    `json:"day"`
	Year        int    `json:"year"`
	Part        int    `json:"part"`
	Description string `json:"description,omitempty"`
}

type HistoryEntry struct {
	Hash         string        `json:"hash"`
	Timestamp    string        `json:"timestamp"`
	Measurements []Measurement `json:"measurements"`
}

func MeasurementKey(m Measurement) string {
	type identity struct {
		Language    string `json:"l"`
		GroupKey    string `json:"g"`
		Day         int    `json:"d"`
		Year        int    `json:"y"`
		Part        int    `json:"p"`
		Description string `json:"desc"`
	}
	b, _ := json.Marshal(identity{m.Language, m.GroupKey, m.Day, m.Year, m.Part, m.Description})
	return string(b)
}

func DeduplicateMeasurements(measurements []Measurement) []Measurement {
	seen := make(map[string]struct{}, len(measurements))
	out := make([]Measurement, 0, len(measurements))
	for _, m := range measurements {
		k := MeasurementKey(m)
		if _, dup := seen[k]; dup {
			continue
		}
		seen[k] = struct{}{}
		out = append(out, m)
	}
	return out
}

func MergeWithHistory(history []HistoryEntry, entry HistoryEntry) []HistoryEntry {
	updated := make([]HistoryEntry, 0, len(history)+1)
	for _, h := range history {
		if h.Hash != entry.Hash {
			updated = append(updated, h)
		}
	}
	updated = append(updated, entry)
	sort.Slice(updated, func(i, j int) bool {
		return updated[i].Timestamp < updated[j].Timestamp
	})
	return updated
}

func PruneHistory(history []HistoryEntry, max int) []HistoryEntry {
	if max <= 0 || len(history) <= max {
		return history
	}
	return history[len(history)-max:]
}

func LoadInputFiles(dir string, logf func(format string, args ...any)) ([]Measurement, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			logf("warning: input-dir %q does not exist — no measurements loaded", dir)
			return nil, nil
		}
		return nil, fmt.Errorf("readdir %s: %w", dir, err)
	}

	var all []Measurement
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".json") || e.Name() == "data.json" {
			continue
		}
		path := filepath.Join(dir, e.Name())
		raw, err := os.ReadFile(path)
		if err != nil {
			logf("read %s: %v", path, err)
			continue
		}
		var doc BenchmarkFile
		if err := ValidateAndParse(raw, &doc); err != nil {
			logf("validation failed for %s: %v", e.Name(), err)
			continue
		}
		all = append(all, doc.Measurements...)
	}
	return DeduplicateMeasurements(all), nil
}

func LoadHistory(path string) ([]HistoryEntry, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("read %s: %w", path, err)
	}
	var history []HistoryEntry
	if err := json.Unmarshal(raw, &history); err != nil {
		return nil, fmt.Errorf("parse %s: %w", path, err)
	}
	return history, nil
}

func WriteHistory(path string, history []HistoryEntry) error {
	blob, err := json.MarshalIndent(history, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("mkdirall: %w", err)
	}
	if err := os.WriteFile(path, blob, 0o644); err != nil {
		return fmt.Errorf("write %s: %w", path, err)
	}
	return nil
}

func CopyStaticFiles(fsys fs.FS, dir string) error {
	copied := 0
	err := fs.WalkDir(fsys, "static", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		data, err := fs.ReadFile(fsys, path)
		if err != nil {
			return fmt.Errorf("read embedded %s: %w", path, err)
		}
		dest := filepath.Join(dir, d.Name())
		if err := os.WriteFile(dest, data, 0o644); err != nil {
			return fmt.Errorf("write %s: %w", dest, err)
		}
		copied++
		return nil
	})
	if err != nil {
		return err
	}
	if copied == 0 {
		return fmt.Errorf("no static files found in embedded FS — binary may have been built without static assets")
	}
	return nil
}
