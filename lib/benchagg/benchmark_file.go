package benchagg

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"os"

	"github.com/google/jsonschema-go/jsonschema"
)

var inputFileSchema *jsonschema.Resolved

//go:embed json_schema.json
var schemaData []byte

func init() {
	var s jsonschema.Schema
	if err := json.Unmarshal(schemaData, &s); err != nil {
		panic("benchagg: invalid embedded JSON schema: " + err.Error())
	}
	rs, err := s.Resolve(nil)
	if err != nil {
		panic("benchagg: failed to resolve JSON schema: " + err.Error())
	}
	inputFileSchema = rs
}

type BenchmarkFile struct {
	Name         string        `json:"name"`
	Hash         string        `json:"hash"`
	Timestamp    string        `json:"timestamp"`
	Measurements []Measurement `json:"measurements"`
}

func ValidateAndParse(raw []byte, doc *BenchmarkFile) error {
	var instance any
	if err := json.Unmarshal(raw, &instance); err != nil {
		return fmt.Errorf("invalid JSON: %w", err)
	}

	if err := inputFileSchema.Validate(instance); err != nil {
		return fmt.Errorf("schema validation: %w", err)
	}

	if err := json.Unmarshal(raw, doc); err != nil {
		return fmt.Errorf("unmarshal: %w", err)
	}
	return nil
}

func (f *BenchmarkFile) LoadFromBytes(raw []byte) error {
	return ValidateAndParse(raw, f)
}

func (f *BenchmarkFile) LoadFromFile(path string) error {
	b, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read file: %w", err)
	}
	return f.LoadFromBytes(b)
}

func (f *BenchmarkFile) WriteToFile(path string) error {
	b, err := json.MarshalIndent(f, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal json: %w", err)
	}
	if err := os.WriteFile(path, b, 0o644); err != nil {
		return fmt.Errorf("write file: %w", err)
	}
	return nil
}
