package ui

import (
	"slices"
	"testing"
	"testing/fstest"

	"github.com/lantervale/ember/geom"
	"github.com/lantervale/ember/ui/css"
	"github.com/lantervale/ember/ui/internal/dom"
	"github.com/stretchr/testify/assert"
)

func TestElementsTakeAttributesAndChildrenAsOptions(t *testing.T) {
	label := Span(Text("name"))
	in := Input(ID("name"), Class("field", "wide"), Type("text"), Value("Heron"), Style("width: 120px"),
		Attr("placeholder", "who"))
	row := Div(Class("row"), label, in)

	assert.True(t, slices.Equal([]*Node{label, in}, row.Children()))
	assert.Equal(t, "name", in.ID())
	assert.Equal(t, []string{"field", "wide"}, in.Classes())
	assert.Equal(t, "width: 120px", in.Style())
	for name, want := range map[string]string{"type": "text", "value": "Heron", "placeholder": "who"} {
		v, _ := in.Attr(name)
		assert.Equal(t, want, v)
	}
	assert.Equal(t, "name", row.Text())
}

func TestLayoutSetsEveryElementsBox(t *testing.T) {
	d := NewDocument()
	p := P(Style("height: 30px; margin: 5px"))
	d.Body().Append(Div(Style("padding: 10px"), p))
	d.Layout(geom.V2(400, 300))

	assert.Equal(t, geom.Rect{X: 23, Y: 23, W: 354, H: 30}, p.Box(), "inside body's 8px margin")
	assert.Equal(t, float32(384), d.Body().Box().W)
	assert.Same(t, d.HTML(), d.Body().Parent())
}

func TestAStyleAttributesWarningIsKeptOnce(t *testing.T) {
	d := NewDocument()
	n := Div(Style("float: left"))
	d.Body().Append(n)
	d.Layout(geom.V2(400, 300))
	for range 3 {
		n.SetState(1, true)
		d.Layout(geom.V2(400, 300))
		n.SetState(1, false)
		d.Layout(geom.V2(400, 300))
	}
	assert.Len(t, d.Warnings(), 1)
}

func TestADisabledAttributeRestylesTheFirstLayout(t *testing.T) {
	d := NewDocument(Stylesheet(fstest.MapFS{"a.css": {Data: []byte("button:disabled { opacity: 0.5; }")}}, "a.css"))
	b := Button(Text("x"), Disabled())
	d.Body().Append(b)
	d.Layout(geom.V2(200, 100))
	assert.Equal(t, float32(0.5), dom.Data(b).Style.Opacity)
	b.RemoveAttr("disabled")
	d.Layout(geom.V2(200, 100))
	assert.Equal(t, float32(1), dom.Data(b).Style.Opacity)
}

func TestAnUnchangedDocumentKeepsItsBoxes(t *testing.T) {
	d := NewDocument()
	box := Div(Style("height: 40px; overflow-y: scroll"), Div(Style("height: 200px")))
	d.Body().Append(box)
	size := geom.V2(400, 300)
	d.Layout(size)
	tree := d.tree

	dom.Data(box).Scroll = geom.V2(0, 30)
	d.Layout(size)
	assert.Same(t, tree, d.tree, "nothing to lay out again")
	assert.Equal(t, geom.V2(0, 30), d.tree.Of[box].Scroll, "the scroll offset still moves")

	box.SetStyle("height: 50px; overflow-y: scroll")
	d.Layout(size)
	assert.NotSame(t, tree, d.tree)
	assert.Equal(t, float32(50), box.Box().H)

	tree = d.tree
	d.Layout(geom.V2(500, 300))
	assert.NotSame(t, tree, d.tree, "a new viewport lays out again")
}

func TestImagesComeFromTheEngineAndTheSheetsFiles(t *testing.T) {
	png := css.DefaultImages()["ember:select-arrow"]
	d := NewDocument(Stylesheet(fstest.MapFS{"ui/a.css": {Data: []byte("")}, "ui/img/x.png": {Data: png}}, "ui/a.css"))
	assert.NotNil(t, d.image("ember:select-arrow"))
	assert.NotNil(t, d.image("ui/img/x.png"))
	assert.Same(t, d.image("ui/img/x.png"), d.image("ui/img/x.png"), "decoded once")
	assert.True(t, d.image("ui/img/x.png").Encoded(), "filtered in encoded sRGB, as CSS (decision 0056)")
	assert.Nil(t, d.image("ui/img/none.png"))
	assert.Nil(t, d.image("ui/img/none.png"))
	assert.Len(t, d.Warnings(), 1)
}

func TestAnImagesNaturalSizeIsItsPixels(t *testing.T) {
	d := NewDocument()
	size, ok := d.ImageSize("ember:select-arrow")
	assert.True(t, ok)
	assert.Equal(t, geom.V2(10, 6), size)
	_, ok = d.ImageSize("nope.png")
	assert.False(t, ok)
}

func TestAStaticFrameWithTextFieldsAllocatesNothing(t *testing.T) {
	d := NewDocument()
	d.Body().Append(Input(Value("Heron")), Textarea(Text("notes")), Input(Type("password"), Value("x")))
	size := geom.V2(400, 300)
	d.Layout(size)
	d.layText()
	allocs := testing.AllocsPerRun(20, func() { d.layText() })
	assert.Zero(t, allocs)
}

func TestDocOfNamesWhatItGotInsteadOfADocument(t *testing.T) {
	assert.PanicsWithValue(t, "ui: internal caller passed int, not *Document", func() { docOf(42) })
	d := NewDocument()
	assert.Same(t, d, docOf(d))
}
