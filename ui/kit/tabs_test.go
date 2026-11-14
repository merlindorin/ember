package kit_test

import (
	"testing"

	"github.com/lantervale/ember/automation"
	"github.com/lantervale/ember/input"
	"github.com/lantervale/ember/ui"
	"github.com/lantervale/ember/ui/kit"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTabsShowOnePageAtATime(t *testing.T) {
	video, audio := kit.Label("Video page"), kit.Label("Audio page")
	tabs := kit.Tabs(kit.Tab("Video", video), kit.Tab("Audio", audio), kit.Tab("Controls", kit.Label("Controls page")))
	s := show(t, tabs)
	changes := watch[ui.Changed](s)
	assert.Greater(t, video.Box().W, float32(0), "the first tab's page shows")
	assert.Zero(t, audio.Box().W)
	assert.Equal(t, automation.RoleSelect, s.element(t, "Audio").Role)
	assert.Equal(t, 0, kit.SelectedTab(s.doc, tabs))

	s.click(kit.TabOf(tabs, 1))
	assert.Zero(t, video.Box().W)
	assert.Greater(t, audio.Box().W, float32(0))
	require.Len(t, *changes, 1)
	assert.Equal(t, 1, kit.SelectedTab(s.doc, tabs))

	s.key(input.KeyRight)
	assert.Equal(t, 2, kit.SelectedTab(s.doc, tabs), "the arrows walk the tabs")
	assert.True(t, s.doc.Focused(kit.TabOf(tabs, 2)))
}

func TestTabPagesAreNamedByTheirTabs(t *testing.T) {
	tabs := kit.Tabs(kit.Tab("Floor tiles", ui.Div()), kit.Tab("Walls", ui.Div(), ui.ID("w")))
	assert.Equal(t, "floor-tiles-panel", tabs.Children()[1].ID())
	assert.Equal(t, "w-panel", tabs.Children()[2].ID())
	v, _ := kit.TabOf(tabs, 0).Attr("aria-controls")
	assert.Equal(t, "floor-tiles-panel", v)
}
