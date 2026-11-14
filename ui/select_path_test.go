package ui_test

import (
	"strings"
	"testing"

	"github.com/lantervale/ember"
	"github.com/lantervale/ember/automation"
	"github.com/lantervale/ember/platform"
	"github.com/lantervale/ember/ui"
	"github.com/stretchr/testify/assert"
)

func recordedIDs(t *testing.T, nodes ...*ui.Node) []string {
	t.Helper()
	d := ui.NewDocument()
	d.Body().Append(nodes...)
	win := platform.NewWindow(platform.Size(400, 200))
	win.Add(d)
	auto := automation.New(automation.Recording())
	ember.New(ember.Automation(auto)).AddEntities(win).RunFrames(3)
	_, els := auto.Snapshot("", "")
	var ids []string
	for _, e := range els {
		ids = append(ids, e.ID)
	}
	return ids
}

func options() []ui.Arg {
	return []ui.Arg{ui.Option(ui.Text("windowed")), ui.Option(ui.Text("borderless")), ui.Option(ui.Text("fullscreen"))}
}

func TestASelectIsNamedByItsLabelNeverItsOptions(t *testing.T) {
	ids := strings.Join(recordedIDs(t, ui.Label(ui.Text("Display mode"), ui.Select(options()...))), " ")
	assert.Contains(t, ids, "display-mode")
	assert.NotContains(t, ids, "windowedborderlessfullscreen")
}

func TestASelectIsNamedByItsIDThenItsAriaLabel(t *testing.T) {
	ids := strings.Join(recordedIDs(t,
		ui.Select(append(options(), ui.ID("mode"))...),
		ui.Select(append(options(), ui.Attr("aria-label", "Quality"))...)), " ")
	assert.Contains(t, ids, "mode")
	assert.Contains(t, ids, "quality")
	assert.NotContains(t, ids, "windowed")
}
