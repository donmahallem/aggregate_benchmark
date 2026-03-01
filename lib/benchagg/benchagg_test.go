package benchagg_test

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/donmahallem/aggregate_benchmark/lib/benchagg"
)

// ── helpers ───────────────────────────────────────────────────────────────────

func entry(hash, ts string, ms []benchagg.Measurement) benchagg.HistoryEntry {
	return benchagg.HistoryEntry{Hash: hash, Timestamp: ts, Measurements: ms}
}

func meas(lang string, day, year, part int, dur string) benchagg.Measurement {
	return benchagg.Measurement{Language: lang, Day: day, Year: year, Part: part, Duration: dur}
}

func validDoc() []byte {
	doc := benchagg.BenchmarkFile{
		Name:      "test",
		Hash:      "abc123",
		Timestamp: "2024-12-01T12:00:00Z",
		Measurements: []benchagg.Measurement{
			meas("go", 1, 2024, 1, "500us"),
		},
	}
	b, err := json.Marshal(doc)
	if err != nil {
		panic(fmt.Sprintf("failed to marshal valid doc: %v", err))
	}
	return b
}

// ── MeasurementKey ────────────────────────────────────────────────────────────

func TestMeasurementKey_SameIdentity(t *testing.T) {
	a := meas("go", 1, 2024, 1, "1ms")
	b := meas("go", 1, 2024, 1, "999s") // different duration — same identity
	if benchagg.MeasurementKey(a) != benchagg.MeasurementKey(b) {
		t.Error("expected same key when only duration differs")
	}
}

func TestMeasurementKey_DifferentLanguage(t *testing.T) {
	a := meas("go", 1, 2024, 1, "1ms")
	b := meas("python", 1, 2024, 1, "1ms")
	if benchagg.MeasurementKey(a) == benchagg.MeasurementKey(b) {
		t.Error("expected different keys for different languages")
	}
}

func TestMeasurementKey_DifferentDay(t *testing.T) {
	a := meas("go", 1, 2024, 1, "1ms")
	b := meas("go", 2, 2024, 1, "1ms")
	if benchagg.MeasurementKey(a) == benchagg.MeasurementKey(b) {
		t.Error("expected different keys for different days")
	}
}

func TestMeasurementKey_DifferentPart(t *testing.T) {
	a := meas("go", 1, 2024, 1, "1ms")
	b := meas("go", 1, 2024, 2, "1ms")
	if benchagg.MeasurementKey(a) == benchagg.MeasurementKey(b) {
		t.Error("expected different keys for different parts")
	}
}

func TestMeasurementKey_GroupKeyDistinguishes(t *testing.T) {
	a := meas("go", 1, 2024, 1, "1ms")
	b := meas("go", 1, 2024, 1, "1ms")
	b.GroupKey = "sample"
	if benchagg.MeasurementKey(a) == benchagg.MeasurementKey(b) {
		t.Error("expected different keys for different group_key values")
	}
}

func TestMeasurementKey_EmptyGroupKeyEquivalent(t *testing.T) {
	a := meas("go", 1, 2024, 1, "1ms")
	b := meas("go", 1, 2024, 1, "1ms")
	b.GroupKey = ""
	if benchagg.MeasurementKey(a) != benchagg.MeasurementKey(b) {
		t.Error("expected same key when group_key is empty and omitted")
	}
}

// ── ValidateAndParse ──────────────────────────────────────────────────────────

func TestValidateAndParse_Valid(t *testing.T) {
	var doc benchagg.BenchmarkFile
	if err := benchagg.ValidateAndParse(validDoc(), &doc); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidateAndParse_MissingName(t *testing.T) {
	raw := []byte(`{"hash":"abc","timestamp":"2024-01-01T00:00:00Z","measurements":[]}`)
	var doc benchagg.BenchmarkFile
	if err := benchagg.ValidateAndParse(raw, &doc); err == nil {
		t.Error("expected error for missing name")
	}
}

func TestValidateAndParse_InvalidDuration(t *testing.T) {
	raw := []byte(`{"name":"x","hash":"abc","timestamp":"2024-01-01T00:00:00Z","measurements":[{"language":"go","day":1,"year":2024,"part":1,"duration":"bad"}]}`)
	var doc benchagg.BenchmarkFile
	if err := benchagg.ValidateAndParse(raw, &doc); err == nil {
		t.Error("expected error for invalid duration")
	}
}

func TestValidateAndParse_ValidDurationUnits(t *testing.T) {
	units := []string{"1ns", "1us", "1µs", "1ms", "1s", "1m", "1h"}
	for _, u := range units {
		raw := []byte(`{"name":"x","hash":"abc","timestamp":"2024-01-01T00:00:00Z","measurements":[{"language":"go","day":1,"year":2024,"part":1,"duration":"` + u + `"}]}`)
		var doc benchagg.BenchmarkFile
		if err := benchagg.ValidateAndParse(raw, &doc); err != nil {
			t.Errorf("unit %q should be valid, got: %v", u, err)
		}
	}
}

func TestValidateAndParse_MissingMeasurementDay(t *testing.T) {
	raw := []byte(`{"name":"x","hash":"abc","timestamp":"2024-01-01T00:00:00Z","measurements":[{"language":"go","year":2024,"part":1,"duration":"1ms"}]}`)
	var doc benchagg.BenchmarkFile
	if err := benchagg.ValidateAndParse(raw, &doc); err == nil {
		t.Error("expected error for missing day")
	}
}

func TestValidateAndParse_InvalidJSON(t *testing.T) {
	var doc benchagg.BenchmarkFile
	if err := benchagg.ValidateAndParse([]byte("not json"), &doc); err == nil {
		t.Error("expected error for invalid JSON")
	}
}

// ── DeduplicateMeasurements ───────────────────────────────────────────────────

func TestDeduplicateMeasurements_UniquePreserved(t *testing.T) {
	m1 := meas("go", 1, 2024, 1, "1ms")
	m2 := meas("python", 1, 2024, 1, "5ms")
	m3 := meas("go", 1, 2024, 2, "2ms")
	out := benchagg.DeduplicateMeasurements([]benchagg.Measurement{m1, m2, m3})
	if len(out) != 3 {
		t.Errorf("expected 3, got %d", len(out))
	}
}

func TestDeduplicateMeasurements_DuplicateRemoved(t *testing.T) {
	m := meas("go", 1, 2024, 1, "1ms")
	out := benchagg.DeduplicateMeasurements([]benchagg.Measurement{m, m})
	if len(out) != 1 {
		t.Errorf("expected 1, got %d", len(out))
	}
}

func TestDeduplicateMeasurements_FirstOccurrenceKept(t *testing.T) {
	a := meas("go", 1, 2024, 1, "1ms")
	b := meas("go", 1, 2024, 1, "9s") // same identity, different duration
	out := benchagg.DeduplicateMeasurements([]benchagg.Measurement{a, b})
	if len(out) != 1 {
		t.Fatalf("expected 1 result, got %d", len(out))
	}
	if out[0].Duration != "1ms" {
		t.Errorf("expected first occurrence to win, got duration %q", out[0].Duration)
	}
}

func TestDeduplicateMeasurements_Empty(t *testing.T) {
	if out := benchagg.DeduplicateMeasurements(nil); len(out) != 0 {
		t.Errorf("expected empty, got %d", len(out))
	}
}

// ── MergeWithHistory ──────────────────────────────────────────────────────────

func TestMergeWithHistory_AppendNewEntry(t *testing.T) {
	h := []benchagg.HistoryEntry{
		entry("aaa", "2024-12-01T10:00:00Z", nil),
		entry("bbb", "2024-12-08T10:00:00Z", nil),
	}
	result := benchagg.MergeWithHistory(h, entry("ccc", "2024-12-15T10:00:00Z", nil))
	if len(result) != 3 {
		t.Fatalf("expected 3, got %d", len(result))
	}
	if result[2].Hash != "ccc" {
		t.Errorf("expected last entry hash 'ccc', got %q", result[2].Hash)
	}
}

func TestMergeWithHistory_ReplaceExistingHash(t *testing.T) {
	orig := entry("aaa", "2024-12-01T10:00:00Z", []benchagg.Measurement{meas("go", 1, 2024, 1, "1ms")})
	h := []benchagg.HistoryEntry{orig, entry("bbb", "2024-12-08T10:00:00Z", nil)}

	rerun := entry("aaa", "2024-12-01T10:00:00Z", nil) // same hash, no measurements
	result := benchagg.MergeWithHistory(h, rerun)

	if len(result) != 2 {
		t.Fatalf("expected 2 entries after replace, got %d", len(result))
	}
	for _, e := range result {
		if e.Hash == "aaa" && len(e.Measurements) != 0 {
			t.Error("expected replaced entry to have 0 measurements")
		}
	}
}

func TestMergeWithHistory_ChronologicalSort(t *testing.T) {
	h := []benchagg.HistoryEntry{
		entry("bbb", "2024-12-08T10:00:00Z", nil),
		entry("aaa", "2024-12-01T10:00:00Z", nil),
	}
	early := entry("zzz", "2024-11-01T00:00:00Z", nil)
	result := benchagg.MergeWithHistory(h, early)
	if result[0].Hash != "zzz" || result[1].Hash != "aaa" || result[2].Hash != "bbb" {
		t.Errorf("unexpected order: %v", hashSlice(result))
	}
}

func TestMergeWithHistory_NilHistory(t *testing.T) {
	result := benchagg.MergeWithHistory(nil, entry("x", "2024-12-01T00:00:00Z", nil))
	if len(result) != 1 {
		t.Errorf("expected 1, got %d", len(result))
	}
}

// ── PruneHistory ──────────────────────────────────────────────────────────────

func makeHistory(n int) []benchagg.HistoryEntry {
	h := make([]benchagg.HistoryEntry, n)
	for i := range h {
		h[i] = entry(
			fmt.Sprintf("h%02d", i),
			fmt.Sprintf("2024-01-%02dT00:00:00Z", i+1),
			nil,
		)
	}
	return h
}

func TestPruneHistory_BelowLimit(t *testing.T) {
	if out := benchagg.PruneHistory(makeHistory(5), 10); len(out) != 5 {
		t.Errorf("expected 5, got %d", len(out))
	}
}

func TestPruneHistory_AtLimit(t *testing.T) {
	if out := benchagg.PruneHistory(makeHistory(10), 10); len(out) != 10 {
		t.Errorf("expected 10, got %d", len(out))
	}
}

func TestPruneHistory_ExceedsLimit(t *testing.T) {
	out := benchagg.PruneHistory(makeHistory(10), 3)
	if len(out) != 3 {
		t.Fatalf("expected 3, got %d", len(out))
	}
	if out[0].Hash != "h07" || out[2].Hash != "h09" {
		t.Errorf("expected most-recent 3, got %v", hashSlice(out))
	}
}

func TestPruneHistory_LimitOne(t *testing.T) {
	out := benchagg.PruneHistory(makeHistory(5), 1)
	if len(out) != 1 || out[0].Hash != "h04" {
		t.Errorf("expected only last entry h04, got %v", hashSlice(out))
	}
}

func TestPruneHistory_ZeroMeansNoLimit(t *testing.T) {
	if out := benchagg.PruneHistory(makeHistory(50), 0); len(out) != 50 {
		t.Errorf("expected 50, got %d", len(out))
	}
}

func TestPruneHistory_NegativeMeansNoLimit(t *testing.T) {
	if out := benchagg.PruneHistory(makeHistory(50), -1); len(out) != 50 {
		t.Errorf("expected 50, got %d", len(out))
	}
}

func TestPruneHistory_DoesNotMutateOriginal(t *testing.T) {
	h := makeHistory(5)
	benchagg.PruneHistory(h, 2)
	if len(h) != 5 {
		t.Error("PruneHistory must not mutate the original slice")
	}
}

func TestPruneHistory_Empty(t *testing.T) {
	if out := benchagg.PruneHistory([]benchagg.HistoryEntry{}, 10); len(out) != 0 {
		t.Errorf("expected empty, got %d", len(out))
	}
}

// ── utility ───────────────────────────────────────────────────────────────────

func hashSlice(h []benchagg.HistoryEntry) []string {
	s := make([]string, len(h))
	for i, e := range h {
		s[i] = e.Hash
	}
	return s
}
