package kit_test

import (
	"testing"

	"github.com/lantervale/ember/geom"
	"github.com/lantervale/ember/ui"
	"github.com/lantervale/ember/ui/internal/dom"
	"github.com/lantervale/ember/ui/kit"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// shape writes n's tree as tags and text, to compare structures.
func shape(n *ui.Node) string {
	if n.IsText() {
		return n.Text()
	}
	s := "<" + n.Tag().String()
	if st := n.Style(); st != "" {
		s += " " + st
	}
	s += ">"
	for _, c := range n.Children() {
		s += shape(c)
	}
	return s + "</" + n.Tag().String() + ">"
}

func TestRichTextReadsItsTags(t *testing.T) {
	cases := map[string]string{
		"plain":              "<span>plain</span>",
		"a [b]bold[/b] word": "<span>a <strong>bold</strong> word</span>",
		"[i]it[/i] and [color=#ffce76]gold[/color]": "<span><em>it</em> and <span color: #ffce76>gold</span></span>",
		"[b]a[i]b[/b]c[/i]":                         "<span><strong>a<em>b</em></strong><em>c</em></span>",
		"[color=rebeccapurple]x[/color]":            "<span><span color: rebeccapurple>x</span></span>",
		"[u]under[/u] [color=nope]x[/color]":        "<span>[u]under[/u] [color=nope]x[/color]</span>",
		"[/b]stray":                                 "<span>[/b]stray</span>",
		"[b]unclosed":                               "<span><strong>unclosed</strong></span>",
	}
	for src, want := range cases {
		assert.Equal(t, want, shape(kit.RichText(src)), src)
	}
}

func TestRichTextTextIsPlainAndShortcodesExpand(t *testing.T) {
	r := kit.RichText("[b]Gold[/b] :star: found", kit.ExpandShortcodes())
	assert.Equal(t, "Gold ⭐ found", r.Text())
	assert.Equal(t, "Gold :star: found", kit.RichText("[b]Gold[/b] :star: found").Text(), "kept without ExpandShortcodes")
	l := kit.Label("hi :wave:", kit.ExpandShortcodes())
	assert.Equal(t, "hi 👋", l.Text(), "Label takes it too")
}

func TestRichTextWrapsAsALabel(t *testing.T) {
	d := ui.NewDocument(kit.Theme())
	src := "The [b]quick[/b] brown [i]fox[/i] jumps over the [color=#ffce76]lazy dog[/color]"
	one := kit.RichText(src)
	wrapped := kit.RichText(src, ui.Class("kit-wrap"))
	d.Body().Append(ui.Div(ui.Style("width: 120px"), one), ui.Div(ui.Style("width: 120px"), wrapped))
	d.Layout(geom.V2(400, 300))
	assert.Greater(t, one.Box().W, float32(120), "one line, past its box")
	assert.Equal(t, float32(120), wrapped.Box().W)
	assert.Greater(t, wrapped.Box().H, 2*one.Box().H)
	var strong *dom.Node
	wrapped.Walk(func(n *dom.Node) bool {
		if n.Tag() == dom.Strong {
			strong = n
		}
		return true
	})
	assert.NotNil(t, strong)
	assert.Empty(t, d.Warnings())
}

func TestSetRichKeepsTheTagsAndShortcodes(t *testing.T) {
	n := kit.RichText("plain", kit.ExpandShortcodes())
	kit.SetRich(n, "[b]Gold[/b] :coin:")
	require.NotEmpty(t, n.Children())
	assert.Equal(t, "strong", n.Children()[0].Tag().String())
	assert.Equal(t, "Gold", n.Children()[0].Text())
	assert.NotContains(t, n.Text(), ":coin:")
}
