# Benchmark Aggregator

A GitHub Action that collects benchmark JSON files, merges them into a history, and publishes a Chart.js dashboard to GitHub Pages.

## Usage

```yaml
- uses: donmahallem/aggregate_benchmark@main
  with:
    input-dir: benchmarks # folder containing benchmark JSON files
    pages-dir: _pages # checkout of your gh-pages branch
    max-history: 100 # keep only the N most-recent commits (0 = unlimited)
    json-format: inline # inline, pretty, or compact
```

The `json-format` option controls the generated `data.json`:

- `inline` (default): indent the document, with each measurement on one line.
- `pretty`: fully indent the document.
- `compact`: omit formatting whitespace.

The equivalent CLI option is `--json-format=inline` (or `pretty` / `compact`).

## Input file format

Each file in `input-dir` MUST conform to [`json_schema.json`](lib/benchagg/json_schema.json):

```json
{
  "name": "go",
  "hash": "<commit-sha>",
  "timestamp": "2024-12-01T10:00:00Z",
  "measurements": [
    { "language": "go", "year": 2024, "day": 1, "part": 1, "duration": "500us" }
  ]
}
```

Required fields: `name`, `hash`, `timestamp`.  
Each measurement requires: `language`, `year`, `day`, `part`, `duration` (in go format, e.g. `100ns`, `5ms`, `1s`).

## Local preview

```sh
go run . --serve --pages-dir _pages
# open http://localhost:8080
```
