package main

import (
	"embed"
	"flag"
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/donmahallem/aggregate_benchmark/lib/benchagg"
)

//go:embed static
var staticFiles embed.FS

func main() {
	inputDir := flag.String("input-dir", envOr("INPUT_INPUT_DIR", "benchmarks"),
		"directory containing benchmark JSON files (relative to workspace root)")
	pagesDir := flag.String("pages-dir", envOr("INPUT_PAGES_DIR", "./output"),
		"directory to read/write data.json and static site files")
	maxHistoryStr := flag.String("max-history", envOr("INPUT_MAX_HISTORY", "100"),
		"maximum number of history entries to keep (0 = unlimited)")
	serve := flag.Bool("serve", false,
		"serve the static site with data.json over HTTP instead of aggregating")
	addr := flag.String("addr", ":8080",
		"address to listen on when --serve is set (e.g. :8080 or 127.0.0.1:3000)")
	flag.Parse()

	if *serve {
		runServe(*pagesDir, *addr)
		return
	}

	maxHistory, err := strconv.Atoi(*maxHistoryStr)
	if err != nil || maxHistory < 0 {
		log.Printf("warning: invalid max-history %q, using 100", *maxHistoryStr)
		maxHistory = 100
	}

	hash := envOr("GITHUB_SHA", "unknown")

	measurements, meta, err := benchagg.LoadInputFiles(*inputDir, func(format string, args ...any) {
		fmt.Fprintf(os.Stderr, format+"\n", args...)
	})
	if err != nil {
		log.Fatalf("load input files: %v", err)
	}
	fmt.Fprintf(os.Stderr, "loaded %d measurements from %s\n", len(measurements), *inputDir)

	historyPath := filepath.Join(*pagesDir, "data.json")
	outputData, err := benchagg.LoadHistory(historyPath)
	if err != nil {
		log.Printf("warning: could not load history: %v — starting fresh", err)
		outputData = benchagg.OutputData{}
	}
	fmt.Fprintf(os.Stderr, "existing history: %d entries\n", len(outputData.History))

	entry := benchagg.HistoryEntry{
		Hash:         hash,
		Timestamp:    time.Now().UTC().Format(time.RFC3339),
		Measurements: measurements,
	}

	merged := benchagg.MergeWithHistory(outputData.History, entry)
	updated := benchagg.PruneHistory(merged, maxHistory)
	if removed := len(merged) - len(updated); removed > 0 {
		fmt.Fprintf(os.Stderr, "pruned %d old entries (max-history=%d)\n", removed, maxHistory)
	}

	outputData.History = updated
	outputData.Meta = benchagg.MergeMeta(outputData.Meta, meta)

	if err := os.MkdirAll(*pagesDir, 0o755); err != nil {
		log.Fatalf("mkdirall %s: %v", *pagesDir, err)
	}

	if err := benchagg.WriteHistory(historyPath, outputData); err != nil {
		log.Fatalf("write history: %v", err)
	}
	fmt.Fprintf(os.Stderr, "wrote %s (%d entries)\n", historyPath, len(updated))

	if err := benchagg.CopyStaticFiles(staticFiles, *pagesDir); err != nil {
		log.Fatalf("copy static files: %v", err)
	}
	fmt.Fprintln(os.Stderr, "copied static site files")
}

func runServe(pagesDir, addr string) {
	dataPath := filepath.Join(pagesDir, "data.json")

	staticFS, err := fs.Sub(staticFiles, "static")
	if err != nil {
		log.Fatalf("serve: sub fs: %v", err)
	}

	mux := http.NewServeMux()

	mux.HandleFunc("/data.json", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		http.ServeFile(w, r, dataPath)
	})

	mux.Handle("/", http.FileServer(http.FS(staticFS)))

	log.Printf("serving at http://%s  (data.json from %s)", addr, dataPath)
	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Fatalf("serve: %v", err)
	}
}

func envOr(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
