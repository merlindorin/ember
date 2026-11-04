package main

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTheTableKeepsTheLastRunOfAPageAndCountsFailures(t *testing.T) {
	dir := t.TempDir()
	lines := `{"kind":"card","name":"09-button","mean":0.003,"threshold":0.004,"ok":true}
{"kind":"page","name":"m0","mean":0.0003,"threshold":0.0002,"ok":false}
{"kind":"card","name":"01-label","mean":0.001,"threshold":0.002,"ok":true}
{"kind":"page","name":"m0","mean":0.0001,"threshold":0.0002,"ok":true}
`
	require.NoError(t, os.WriteFile(filepath.Join(dir, "results.jsonl"), []byte(lines), 0o644))
	rs, err := read(filepath.Join(dir, "results.jsonl"))
	require.NoError(t, err)
	require.Len(t, rs, 3)
	assert.Equal(t, "01-label", rs[0].Name, "cards first, by name")

	var out bytes.Buffer
	assert.Zero(t, table(&out, rs), "m0's last run passed")
	assert.Contains(t, out.String(), "09-button")
}

func TestTheGuideGetsASentencePerCardAndASummary(t *testing.T) {
	dir := t.TempDir()
	kit := filepath.Join(dir, "07-kit")
	require.NoError(t, os.MkdirAll(kit, 0o755))
	write := func(name, body string) {
		require.NoError(t, os.WriteFile(filepath.Join(kit, name), []byte(body), 0o644))
	}
	write("09-button.md", "# Button\n\n![](b.png)\n<!-- chrome -->\n<!-- /chrome -->\n\nText.\n")
	write("06-row-and-column.md", "# Row and column\n\n<!-- chrome 06-row-column -->\n<!-- /chrome -->\n")
	write("14-textinput.md",
		"# Text input\n\n<!-- chrome -->\nold\n<!-- /chrome -->\n<!-- chrome 14b-search -->\n<!-- /chrome -->\n")
	rs := []result{
		{Kind: "card", Name: "09-button", Mean: 0.0043, Threshold: 0.0066},
		{Kind: "card", Name: "06-row-column", Mean: 0.0011, Threshold: 0.0026},
		{Kind: "card", Name: "14-textinput", Mean: 0.003, Threshold: 0.004},
		{Kind: "card", Name: "14b-search", Mean: 0.0029, Threshold: 0.0038},
		{Kind: "page", Name: "m0", Mean: 0.0001, Threshold: 0.0002},
	}
	require.NoError(t, fill(dir, rs))

	b, _ := os.ReadFile(filepath.Join(kit, "09-button.md"))
	assert.Contains(t, string(b),
		"<!-- chrome -->\nMatches Chrome within 0.43% of its pixels (threshold 0.66%).\n<!-- /chrome -->\n\nText.")
	ti, _ := os.ReadFile(filepath.Join(kit, "14-textinput.md"))
	assert.NotContains(t, string(ti), "\nold\n")
	assert.Contains(t, string(ti), "<!-- chrome 14b-search -->\nMatches Chrome within 0.29%")
	sum, _ := os.ReadFile(filepath.Join(dir, "08-chrome.md"))
	assert.Contains(t, string(sum), "# Matches Chrome\n")
	assert.Contains(t, string(sum), "| [Button](07-kit/09-button.md) | 0.43% | 0.66% |")
	assert.Contains(t, string(sum), "| [Text input: search](07-kit/14-textinput.md) | 0.29% | 0.38% |")
	assert.Contains(t, string(sum), "## Pixel pages\n", "the CSS pages follow the components")
	assert.Contains(t, string(sum), "| m0 | 0.01% | 0.02% |")
	assert.Contains(t, string(sum), "| [Row and column](07-kit/06-row-and-column.md) | 0.11% |",
		"a page's own card, named in its marker")
	rc, _ := os.ReadFile(filepath.Join(kit, "06-row-and-column.md"))
	assert.Contains(t, string(rc), "<!-- chrome 06-row-column -->\nMatches Chrome", "a named marker keeps its name")

	again, _ := os.ReadFile(filepath.Join(kit, "09-button.md"))
	require.NoError(t, fill(dir, rs))
	b, _ = os.ReadFile(filepath.Join(kit, "09-button.md"))
	assert.Equal(t, string(again), string(b), "filling twice changes nothing")
}

func TestACardWithoutMarkersFailsLoudly(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "07-kit"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "07-kit", "09-button.md"), []byte("# Button\n"), 0o644))
	err := fill(dir, []result{{Kind: "card", Name: "09-button"}})
	assert.ErrorContains(t, err, "09-button")
}

func TestTheSummaryMarksFailures(t *testing.T) {
	var b bytes.Buffer
	require.NoError(t,
		summaryTable(&b,
			[]result{{Kind: "page", Name: "m0", Mean: 0.0003, Threshold: 0.0002},
				{Kind: "card", Name: "09-button", Mean: 0.004, Threshold: 0.0066, OK: true}}))
	assert.Contains(t, b.String(), "| page | m0 | 0.03% | 0.02% | **FAIL** |")
	assert.Contains(t, b.String(), "2 results, 1 over their threshold")
}

func TestResetEmptiesOnlyTheResults(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "r")
	require.NoError(t, resetDir(dir))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "results.jsonl"), []byte("{}\n"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "m0.png"), nil, 0o644))
	require.NoError(t, resetDir(dir))
	_, err := os.Stat(filepath.Join(dir, "results.jsonl"))
	assert.True(t, os.IsNotExist(err))
	_, err = os.Stat(filepath.Join(dir, "m0.png"))
	assert.NoError(t, err)
}

func TestAnExtraCardIsNamedByWhatSetsItApart(t *testing.T) {
	assert.Equal(t, "save", variant("32b-file-dialog-save", "32-file-dialog"))
	assert.Equal(t, "scrolled", variant("24b-table-scrolled", "24-table"))
	assert.Equal(t, "search", variant("14b-search", "14-text-input"))
}
