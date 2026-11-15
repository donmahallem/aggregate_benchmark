package main

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/donmahallem/aggregate_benchmark/lib/benchagg"
)

func TestJSONFormatInput(t *testing.T) {
	binary := filepath.Join(t.TempDir(), "aggregator.exe")
	if err := exec.Command("go", "build", "-o", binary, ".").Run(); err != nil {
		t.Fatalf("build aggregator: %v", err)
	}
	t.Setenv("INPUT_JSON-FORMAT", "")
	t.Setenv("INPUT_JSON_FORMAT", "")
	t.Setenv("GITHUB_SHA", "format-test")

	for _, tc := range []struct {
		name    string
		env     string
		envName   string
		flag    string
		format  string
		invalid bool
	}{
		{name: "default", format: "inline"},
		{name: "action inline", env: "inline", format: "inline"},
		{name: "action compact", env: "compact", format: "compact"},
		{name: "action pretty", env: "pretty", format: "pretty"},
		{name: "underscore environment", envName: "INPUT_JSON_FORMAT", env: "compact", format: "compact"},
		{name: "flag overrides action", env: "compact", flag: "pretty", format: "pretty"},
		{name: "invalid input", env: "invalid", invalid: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			args := []string{"--input-dir=testdata", "--pages-dir=" + dir}
			if tc.flag != "" {
				args = append(args, "--json-format="+tc.flag)
			}
			cmd := exec.Command(binary, args...)
			envName := tc.envName
			if envName == "" {
				envName = "INPUT_JSON-FORMAT"
			}
			cmd.Env = append(os.Environ(), envName+"="+tc.env)
			output, err := cmd.CombinedOutput()
			if tc.invalid {
				if err == nil || !strings.Contains(string(output), "invalid json-format") {
					t.Fatalf("expected invalid input error, got %v: %s", err, output)
				}
				return
			}
			if err != nil {
				t.Fatalf("aggregator failed: %v: %s", err, output)
			}
			blob, err := os.ReadFile(filepath.Join(dir, "data.json"))
			if err != nil {
				t.Fatal(err)
			}
			var data benchagg.OutputData
			if err := json.Unmarshal(blob, &data); err != nil {
				t.Fatalf("invalid JSON: %v", err)
			}
			if len(data.History) != 1 || data.History[0].Hash != "format-test" || len(data.History[0].Measurements) == 0 {
				t.Fatal("generated history was not preserved")
			}
			var compact bytes.Buffer
			if err := json.Compact(&compact, blob); err != nil {
				t.Fatalf("invalid JSON: %v", err)
			}
			want := compact.Bytes()
			if tc.format == "pretty" {
				var indented bytes.Buffer
				if err := json.Indent(&indented, want, "", "  "); err != nil {
					t.Fatal(err)
				}
				want = indented.Bytes()
			}
			if tc.format == "inline" {
				if !bytes.Contains(blob, []byte("\n  \"history\": [\n")) ||
					!bytes.Contains(blob, []byte("\n        { \"series_key\":")) ||
					bytes.Contains(blob, []byte("\n          \"series_key\":")) {
					t.Fatalf("measurements are not inline in indented output: %s", blob)
				}
			} else if !bytes.Equal(blob, want) {
				t.Errorf("unexpected JSON formatting for %s:\ngot: %s\nwant: %s", tc.name, blob, want)
			}
		})
	}
}
