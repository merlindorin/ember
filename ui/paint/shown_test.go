package paint //nolint:testpackage // white-box: builds box trees by hand and reads the painter's own state

import (
	"testing"

	"github.com/lantervale/ember/color"
	"github.com/lantervale/ember/ui/internal/box"
	"github.com/lantervale/ember/ui/internal/dom"
	"github.com/lantervale/ember/ui/internal/style"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPaintDrawsWhatATransitionShowsNotTheCascadesStyle(t *testing.T) {
	div, txt := dom.NewElement(dom.Div), dom.NewText("hi")
	div.Append(txt)
	b := el(0, 0, 100, 20, func(s *style.Computed) { s.Background, s.Color = red, red })
	b.Node = div
	run := text(0, 0, "hi", b.Style)
	run.Node = txt
	anon := &box.Box{Kind: box.AnonymousBlock, Style: b.Style}
	anon.Append(run)
	b.Append(anon)

	mid := *b.Style
	mid.Background, mid.Color = color.RGB(0.5, 0, 0.5), color.RGB(0, 0.5, 0)
	dom.Data(div).Style, dom.Data(div).Shown = b.Style, &mid
	dom.Data(txt).Style, dom.Data(txt).Shown = b.Style, &mid

	r := newRecorder()
	Paint(r, tree(b))
	require.Len(t, r.of("rect"), 1)
	assert.Equal(t, mid.Background, r.of("rect")[0].Color)
	require.Len(t, r.of("text"), 1)
	assert.Equal(t, mid.Color, r.of("text")[0].Color, "a run inside an anonymous box takes its element's shown style")
}

func TestABoxWithoutAShownStyleIsPaintedWithItsStyle(t *testing.T) {
	b := el(0, 0, 10, 10, bg(red))
	b.Node = dom.NewElement(dom.Div)
	r := newRecorder()
	Paint(r, tree(b))
	assert.Equal(t, red, r.of("rect")[0].Color)
}
