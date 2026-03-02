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
	SeriesKey   string `json:"series_key"`
	GroupKey    string `json:"group_key"`
	Duration    string `json:"duration"`
	Iterations  int    `json:"iterations,omitempty"`
	Description string `json:"description,omitempty"`
}

type HistoryEntry struct {
	Hash         string        `json:"hash"`
	Timestamp    string        `json:"timestamp"`
	Measurements []Measurement `json:"measurements"`
}

// SeriesMeta holds display metadata for a single series (a line on a graph).
type SeriesMeta struct {
	Label string `json:"label,omitempty"`
	Color string `json:"color,omitempty"`
}

// GroupMeta holds display metadata for a single group (a graph card).
type GroupMeta struct {
	Title string `json:"title,omitempty"`
}

// Meta holds all display-name mappings. It is carried at the top level of both
// input files and the output data.json — never repeated per measurement.
type Meta struct {
	Series map[string]SeriesMeta `json:"series,omitempty"`
	Groups map[string]GroupMeta  `json:"groups,omitempty"`
}

// OutputData is the shape of the written data.json.
type OutputData struct {
	Meta    Meta           `json:"meta"`
	History []HistoryEntry `json:"history"`
}

// MergeMeta merges override into base, with override values taking precedence.
// A new Meta is returned; neither argument is mutated.
func MergeMeta(base, override Meta) Meta {
	out := Meta{
		Series: make(map[string]SeriesMeta),
		Groups: make(map[string]GroupMeta),
	}
	for k, v := range base.Series {
		out.Series[k] = v
	}
	for k, v := range override.Series {
		out.Series[k] = v
	}
	for k, v := range base.Groups {
		out.Groups[k] = v
	}
	for k, v := range override.Groups {
		out.Groups[k] = v
	}
	return out
}

// MeasurementKey returns a stable identity key based solely on the two stable
// fields: group_key (which graph) and series_key (which line). Display-only
// fields (description) are intentionally excluded so they can be changed
// without affecting deduplication or history continuity.
func MeasurementKey(m Measurement) string {
	type identity struct {
		GroupKey  string `json:"g"`
		SeriesKey string `json:"s"`
	}
	b, _ := json.Marshal(identity{m.GroupKey, m.SeriesKey})
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

func LoadInputFiles(dir string, logf func(format string, args ...any)) ([]Measurement, Meta, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			logf("warning: input-dir %q does not exist — no measurements loaded", dir)
			return nil, Meta{}, nil
		}
		return nil, Meta{}, fmt.Errorf("readdir %s: %w", dir, err)
	}

	var all []Measurement
	merged := Meta{}
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
		merged = MergeMeta(merged, doc.Meta)
	}
	return DeduplicateMeasurements(all), merged, nil
}

func LoadHistory(path string) (OutputData, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return OutputData{}, nil
		}
		return OutputData{}, fmt.Errorf("read %s: %w", path, err)
	}
	var data OutputData
	if err := json.Unmarshal(raw, &data); err != nil {
		return OutputData{}, fmt.Errorf("parse %s: %w", path, err)
	}
	return data, nil
}

func WriteHistory(path string, data OutputData) error {
	blob, err := json.MarshalIndent(data, "", "  ")
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
