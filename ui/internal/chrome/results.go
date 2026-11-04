//go:build chrome

package chrome

import (
	"embed"
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"
)

// platformThresholds are the thresholds of another system than macOS, whose
// Chrome rasterises a little differently: thresholds/<GOOS>.json maps a
// result's name to the threshold it has there, in place of the one its test
// gives (macOS's).
//
//go:embed thresholds
var platformThresholds embed.FS

//nolint:gochecknoglobals // read once from the embedded thresholds
var overrides = sync.OnceValue(func() map[string]float64 {
	m := map[string]float64{}
	if data, err := platformThresholds.ReadFile("thresholds/" + runtime.GOOS + ".json"); err == nil {
		if err = json.Unmarshal(data, &m); err != nil {
			panic("chrome: thresholds/" + runtime.GOOS + ".json: " + err.Error())
		}
	}
	return m
})

// Result is one page's comparison with Chrome, as the parity report lists
// it.
type Result struct {
	// Kind is "card" for a kit component's review card, "page" for a pixel page.
	Kind      string  `json:"kind"`
	Name      string  `json:"name"`
	Mean      float64 `json:"mean"`
	Threshold float64 `json:"threshold"`
	Pixels    int     `json:"pixels"`
	Total     int     `json:"total"`
	OK        bool    `json:"ok"`
}

// ResultsFile is the file in EMBER_UI_RESULTS that each comparison adds a
// line of JSON to.
const ResultsFile = "results.jsonl"

// Within compares the page at pkg with Chrome, as Pixels does, and fails t
// when its mean difference passes threshold.
func Within(t testing.TB, pkg string, w, h int, threshold float64) Diff {
	t.Helper()
	d := Pixels(t, pkg, w, h)
	check(t, "page", filepath.Base(pkg), d, threshold)
	return d
}

// check fails t when d passes threshold (or this system's, from
// thresholds/<GOOS>.json), and with EMBER_UI_RESULTS set adds the result there.
func check(t testing.TB, kind, name string, d Diff, threshold float64) {
	t.Helper()
	name = strings.TrimPrefix(name, "kit-")
	threshold = thresholdFor(name, threshold, overrides())
	ok := d.Mean <= threshold
	if !ok {
		t.Errorf("mean difference %.4f, more than %.4f", d.Mean, threshold)
	}
	dir := os.Getenv("EMBER_UI_RESULTS")
	if dir == "" {
		return
	}
	line, err := json.Marshal(
		Result{Kind: kind, Name: name, Mean: d.Mean, Threshold: threshold, Pixels: d.Pixels, Total: d.Total, OK: ok},
	)
	if err != nil {
		t.Fatal(err)
	}
	if err = os.MkdirAll(dir, 0o755); err != nil { //nolint:gosec // the folder the gate names in EMBER_UI_RESULTS
		t.Fatal(err)
	}
	f, err := os.OpenFile( //nolint:gosec // the folder the gate names in EMBER_UI_RESULTS
		filepath.Join(dir, ResultsFile),
		os.O_APPEND|os.O_CREATE|os.O_WRONLY,
		0o644,
	)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	if _, err = f.Write(append(line, '\n')); err != nil {
		t.Fatal(err)
	}
}

// thresholdFor is name's threshold on this system: the override when there
// is one, else the test's own.
func thresholdFor(name string, given float64, overrides map[string]float64) float64 {
	if v, ok := overrides[name]; ok {
		return v
	}
	return given
}
