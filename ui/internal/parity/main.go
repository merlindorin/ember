// Command parity reads the results the Chrome pixel tests wrote
// (EMBER_UI_RESULTS, chrome.Result per line) and prints them as a table: each
// page and kit card, its mean difference from Chrome, its threshold, and
// whether it passed. -docs fills the guide: it writes the summary page
// 08-chrome.md whole, and the sentence between each kit page's
// <!-- chrome --> markers (<!-- chrome NAME --> for a page's extra
// cards). -json writes every result. It exits 1 when a result passed its
// threshold, when there are none, or when a card has no markers in the
// guide or markers have no card.
package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
)

// result mirrors chrome.Result, which only builds with the chrome tag.
type result struct {
	Kind      string  `json:"kind"`
	Name      string  `json:"name"`
	Mean      float64 `json:"mean"`
	Threshold float64 `json:"threshold"`
	Pixels    int     `json:"pixels"`
	Total     int     `json:"total"`
	OK        bool    `json:"ok"`
}

func main() {
	dir := flag.String("dir", os.Getenv("EMBER_UI_RESULTS"), "the folder the tests wrote results.jsonl into")
	docs := flag.String("docs", "", "fill the guide's ui section in this folder (docs/05-ui)")
	js := flag.String("json", "", "also write every result as a JSON array to this file")
	summary := flag.String("summary", "", "also write every result as a Markdown table to this file (a CI job summary)")
	reset := flag.Bool("reset", false, "make the folder and empty its results before a run, then stop")
	flag.Parse()
	if *reset {
		if err := resetDir(*dir); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		return
	}
	rs, err := read(filepath.Join(*dir, "results.jsonl"))
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	failed := table(os.Stdout, rs)
	if *docs != "" {
		if err = fill(*docs, rs); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	}
	if *summary != "" {
		if err = writeFile(*summary, func(w io.Writer) error { return summaryTable(w, rs) }); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	}
	if *js != "" {
		if err = writeFile(*js, func(w io.Writer) error {
			e := json.NewEncoder(w)
			e.SetIndent("", "  ")
			return e.Encode(rs)
		}); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	}
	if failed > 0 || len(rs) == 0 {
		os.Exit(1)
	}
}

// read loads the results, the last one kept when a page was run twice,
// sorted by kind then name.
func read(path string) ([]result, error) {
	f, err := os.Open(path) //nolint:gosec // a developer tool reading the report folder it was given
	if err != nil {
		return nil, err
	}
	defer f.Close()
	byName := map[string]result{}
	s := bufio.NewScanner(f)
	for s.Scan() {
		var r result
		if err = json.Unmarshal(s.Bytes(), &r); err != nil {
			return nil, fmt.Errorf("%s: %w", path, err)
		}
		byName[r.Kind+"/"+r.Name] = r
	}
	if err = s.Err(); err != nil {
		return nil, err
	}
	rs := make([]result, 0, len(byName))
	for _, r := range byName {
		rs = append(rs, r)
	}
	sort.Slice(rs, func(i, j int) bool {
		if rs[i].Kind != rs[j].Kind {
			return rs[i].Kind < rs[j].Kind
		}
		return rs[i].Name < rs[j].Name
	})
	return rs, nil
}

// table prints rs and returns how many failed.
func table(w io.Writer, rs []result) int {
	failed := 0
	fmt.Fprintf(w, "%-5s %-28s %8s %9s  %s\n", "kind", "page", "mean", "threshold", "ok")
	for _, r := range rs {
		ok := "ok"
		if !r.OK {
			ok, failed = "FAIL", failed+1
		}
		fmt.Fprintf(w, "%-5s %-28s %7.2f%% %8.2f%%  %s\n", r.Kind, r.Name, 100*r.Mean, 100*r.Threshold, ok)
	}
	fmt.Fprintf(w, "%d results, %d over their threshold\n", len(rs), failed)
	return failed
}

// resetDir makes dir and removes the results a run left in it.
func resetDir(dir string) error {
	if err := os.MkdirAll(dir, 0o755); err != nil { //nolint:gosec // a developer tool's own report folder
		return err
	}
	results := filepath.Join(dir, "results.jsonl")
	if err := os.Remove(results); err != nil && !os.IsNotExist(err) { //nolint:gosec // the tool's own report folder
		return err
	}
	return nil
}

// summaryTable writes every result as Markdown, failures marked.
func summaryTable(w io.Writer, rs []result) error {
	failed := 0
	fmt.Fprintf(w, "| Kind | Page | Mean difference | Threshold | OK |\n|---|---|---|---|---|\n")
	for _, r := range rs {
		ok := "ok"
		if !r.OK {
			ok, failed = "**FAIL**", failed+1
		}
		fmt.Fprintf(w, "| %s | %s | %.2f%% | %.2f%% | %s |\n", r.Kind, r.Name, 100*r.Mean, 100*r.Threshold, ok)
	}
	_, err := fmt.Fprintf(w, "\n%d results, %d over their threshold\n", len(rs), failed)
	return err
}

func writeFile(path string, write func(io.Writer) error) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	if err = write(f); err != nil {
		f.Close()
		return err
	}
	return f.Close()
}
