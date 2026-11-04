package main

import (
	_ "embed"
	"fmt"
	"slices"
	"strconv"
	"strings"

	"github.com/merlindorin/ember/twod"
	"github.com/merlindorin/ember/ui"
	"github.com/merlindorin/ember/world"
)

//go:embed inspector.css
var inspectorCSS []byte

// inspectorDoc is the inspector as a ui document: one grid of a name
// column as wide as its widest name and a value column taking the rest,
// the properties then the objectgroup, styled by inspector.css. It sits
// in the shell's tile section, in the shell's Document.
type inspectorDoc struct {
	intents inspectorIntents
	doc     *ui.Document
	// title is the tile section's summary, "tile <id>".
	title *ui.Node
	// props are the properties' controls and hints their "(default)".
	props  map[string]*ui.Node
	propOf map[*ui.Node]string
	hints  map[string]*ui.Node
	// animation and nearMiss are read only.
	animation, nearMiss *ui.Node
	// objects holds the objectgroup's rows, made again by objectsOf.
	objects   *ui.Node
	objectsOf string
	objField  map[*ui.Node]objectField
	objRow    map[*ui.Node]int
	add, del  *ui.Node
	// changed and pressed are the fields changed and the rows and Delete
	// pressed since the last Update, in order.
	changed, pressed []*ui.Node
}

func newInspectorDoc(intents inspectorIntents, doc *ui.Document, section *ui.Node) *inspectorDoc {
	v := &inspectorDoc{intents: intents, doc: doc, title: section.Children()[0],
		props: map[string]*ui.Node{}, propOf: map[*ui.Node]string{}, hints: map[string]*ui.Node{},
		objField: map[*ui.Node]objectField{}, objRow: map[*ui.Node]int{}}
	v.objects = ui.Div(ui.ID("objects"))
	v.add = v.onChange(ui.Select(ui.ID("add-object"), ui.Placeholder("+ Add object")))
	for _, k := range objectKinds {
		v.add.Append(ui.Option(ui.Text(k)))
	}
	v.del = v.onClick(ui.Button(ui.ID("delete"), ui.Text("Delete object")))
	section.Children()[1].Append(ui.Div(ui.ID("inspector"), ui.Class("inspector"),
		v.properties(),
		v.objects,
		ui.Div(ui.Class("actions"), v.add, ui.Span(ui.Class("spacer")), v.del),
	))
	return v
}

// onChange and onClick note n's changes and presses for Update.
func (v *inspectorDoc) onChange(n *ui.Node) *ui.Node {
	ui.On(n, ui.EventChange, func(e ui.Event) { v.changed = append(v.changed, e.Node) })
	return n
}

func (v *inspectorDoc) onClick(n *ui.Node) *ui.Node {
	ui.On(n, ui.EventClick, func(e ui.Event) { v.pressed = append(v.pressed, e.Node) })
	return n
}

func caption(text string, classes ...string) *ui.Node {
	return ui.Span(ui.ID("caption"), ui.Class(append([]string{"caption"}, classes...)...), ui.Text(text))
}

// properties is the table's head and the engine's tile properties: a
// switch for a bool, a select for an enum and the occluder group, else a
// field; then the animation and the near misses.
func (v *inspectorDoc) properties() *ui.Node {
	p := ui.Div(ui.ID("properties"),
		ui.Span(ui.ID("name-heading"), ui.Class("head"), ui.Text("NAME")),
		ui.Span(ui.ID("value-heading"), ui.Class("head"), ui.Text("VALUE")),
		ui.Span(ui.ID("caption"), ui.Class("group", "full"), ui.Text("properties")),
	)
	for _, d := range engineProperties() {
		var field *ui.Node
		switch values := valuesOf(d); {
		case d.Kind == twod.KindBool:
			sw := ui.Input(ui.ID(d.Name), ui.Type("checkbox"), ui.Attr("switch", ""), ui.Attr("title", d.Doc))
			hint := ui.Span(ui.ID(d.Name+"-default"), ui.Class("hint"), ui.Text("(default)"))
			v.props[d.Name], v.hints[d.Name] = sw, hint
			field = ui.Div(ui.Class("row"), sw, hint)
		case values != nil:
			sel := ui.Select(ui.ID(d.Name), ui.Placeholder(""), ui.Attr("title", d.Doc))
			for _, item := range values {
				sel.Append(ui.Option(ui.Text(item)))
			}
			v.props[d.Name], field = sel, sel
		default:
			in := ui.Input(ui.ID(d.Name), ui.Placeholder(placeholder(d)), ui.Attr("title", d.Doc))
			v.props[d.Name], field = in, in
		}
		v.propOf[v.onChange(v.props[d.Name])] = d.Name
		p.Append(caption(d.Name), field)
	}
	v.animation = ui.Span(ui.ID("animation"), ui.Class("hint"))
	v.nearMiss = ui.Span(ui.ID("near-misses"), ui.Class("hint", "lines"))
	p.Append(caption("animation"), v.animation, caption(""), v.nearMiss)
	return p
}

func hide(n *ui.Node, hidden bool) {
	if hidden {
		n.AddClass("hidden")
	} else {
		n.RemoveClass("hidden")
	}
}

// Show shows s; the objects' rows are made again when the tile, their
// count, the selection or its shape change, and otherwise only set, but
// in the field being typed in.
func (v *inspectorDoc) Show(s inspectorState) {
	if v.title.Text() != s.Title {
		v.title.SetText(s.Title)
	}
	d := v.doc
	for _, p := range s.Properties {
		n := v.props[p.Name]
		if n == nil {
			continue
		}
		v.showProperty(n, p)
	}
	v.animation.SetText(s.Animation)
	misses := strings.Join(s.NearMisses, "\n")
	v.nearMiss.SetText(misses)
	hide(v.nearMiss, misses == "")

	shape := ""
	if s.Selected >= 0 && s.Selected < len(s.Objects) {
		shape = s.Objects[s.Selected].Shape
	}
	if key := fmt.Sprint(s.Title, len(s.Objects), s.Selected, shape); key != v.objectsOf {
		v.objectsOf = key
		v.rebuildObjects(s)
	}
	f := func(x float32) string { return strconv.FormatFloat(float64(x), 'g', -1, 32) }
	for n, at := range v.objField {
		if at.i >= len(s.Objects) {
			continue
		}
		o := s.Objects[at.i]
		poly := o.Shape == keyPolygon
		switch at.key {
		case keyType:
			d.SetSelected(n, slices.Index(objectTypes, o.Type))
		case "shape":
			d.SetSelected(n, slices.Index(objectKinds, o.Shape))
		case "x":
			d.SetValue(n, f(o.X))
		case "y":
			d.SetValue(n, f(o.Y))
		case "width", keyHeight:
			w := o.W
			if at.key == keyHeight {
				w = o.H
			}
			if poly {
				w = 0
				n.SetAttr("readonly", "")
			} else {
				n.RemoveAttr("readonly")
			}
			d.SetValue(n, f(w))
		case "points":
			var pts []string
			for _, p := range o.Points {
				pts = append(pts, f(p.X)+","+f(p.Y))
			}
			d.SetValue(n, strings.Join(pts, " "))
		}
	}
	if !s.Loaded || s.Selected < 0 || s.Selected >= len(s.Objects) {
		v.del.SetAttr("disabled", "")
	} else {
		v.del.RemoveAttr("disabled")
	}
}

// showProperty sets property field n to p: a check with its default hint, a
// choice or a text field, each showing "—" when the marked tiles differ.
func (v *inspectorDoc) showProperty(n *ui.Node, p propertyState) {
	d := v.doc
	switch {
	case v.hints[p.Name] != nil:
		d.SetChecked(n, p.Value == jsonTrue && !p.Mixed)
		if p.Set {
			n.RemoveClass("unset")
		} else {
			n.AddClass("unset")
		}
		hint := v.hints[p.Name]
		hint.SetText("(default)")
		if p.Mixed {
			hint.SetText("—")
		}
		hide(hint, p.Set && !p.Mixed)
	case p.Values != nil || n.Tag().String() == tagSelect:
		n.SetAttr("placeholder", p.Default+" (default)")
		switch {
		case p.Mixed:
			n.SetAttr("placeholder", "—")
			d.SetSelected(n, -1)
		case !p.Set:
			d.SetSelected(n, -1)
		default:
			d.SetSelected(n, slices.Index(p.Values, p.Value))
		}
	default:
		n.SetAttr("placeholder", p.Hint)
		if p.Mixed {
			n.SetAttr("placeholder", "—")
			d.SetValue(n, "")
		} else {
			d.SetValue(n, p.Value)
		}
	}
}

// rebuildObjects makes the objectgroup's rows for s: a folded object's
// title and summary line, both pressed to open it, and the open object's
// title, type, shape, place, size and a polygon's points.
func (v *inspectorDoc) rebuildObjects(s inspectorState) {
	for _, c := range slices.Clone(v.objects.Children()) {
		v.objects.Remove(c)
	}
	clear(v.objField)
	clear(v.objRow)
	v.objects.Append(ui.Span(ui.ID("caption"), ui.Class("group", "full"), ui.Text("objectgroup")))
	if len(s.Objects) == 0 {
		v.objects.Append(caption(""), ui.Span(ui.Class("hint"), ui.Text("none: the tile blocks as its properties say")))
	}
	for i, o := range s.Objects {
		name := fmt.Sprintf("object-%d", i+1)
		if i != s.Selected {
			title := ui.Div(
				ui.ID(name),
				ui.Class("fold", "caption"),
				ui.TabIndex(0),
				ui.Text(fmt.Sprintf("▸ object %d", o.ID)),
			)
			line := ui.Div(ui.ID(name+"-summary"), ui.Class("fold", "summary"), ui.TabIndex(-1), ui.Text(summary(o)))
			v.objRow[v.onClick(title)], v.objRow[v.onClick(line)] = i, i
			v.objects.Append(title, line)
			continue
		}
		field := func(key string, n *ui.Node) *ui.Node {
			v.objField[v.onChange(n)] = objectField{i, key}
			return n
		}
		choice := func(key string, items []string) *ui.Node {
			sel := ui.Select(ui.ID(key))
			for _, item := range items {
				sel.Append(ui.Option(ui.Text(item)))
			}
			return field(key, sel)
		}
		input := func(key string) *ui.Node { return field(key, ui.Input(ui.ID(key))) }
		band := ui.Div(
			ui.Class("band"),
			ui.Div(
				ui.ID(name),
				ui.Class("full", "object-title"),
				ui.Span(ui.ID("title"), ui.Text(fmt.Sprintf("▾ object %d", o.ID))),
			),
			caption(keyType, "sub"),
			choice(keyType, objectTypes),
			caption("shape", "sub"),
			choice("shape", objectKinds),
			caption("x · y", "sub"),
			ui.Div(ui.Class("pair"), input("x"), input("y")),
			caption("width · height", "sub"),
			ui.Div(ui.Class("pair"), input("width"), input(keyHeight)),
		)
		if o.Shape == keyPolygon {
			band.Append(caption("points", "sub"), input("points"))
		}
		v.objects.Append(band)
	}
}

// Update turns what the player changed and pressed into intents.
func (v *inspectorDoc) Update(*world.World) {
	d := v.doc
	changed, pressed := v.changed, v.pressed
	v.changed, v.pressed = nil, nil
	for _, n := range changed {
		switch {
		case n == v.add:
			i := d.Selected(n)
			d.SetSelected(n, -1)
			if i >= 0 && i < len(objectKinds) {
				v.intents.AddObject(Kind(i))
			}
		case v.propOf[n] != "":
			name := v.propOf[n]
			switch {
			case v.hints[name] != nil:
				v.intents.SetBool(name, d.Checked(n))
			case n.Tag().String() == tagSelect:
				v.intents.SetChoice(name, d.Selected(n))
			default:
				v.intents.SetField(name, d.Value(n))
			}
		default:
			if at, ok := v.objField[n]; ok {
				if n.Tag().String() == tagSelect {
					v.intents.ObjectChoice(at.i, at.key, d.Selected(n))
				} else {
					v.intents.ObjectField(at.i, at.key, d.Value(n))
				}
			}
		}
	}
	for _, n := range pressed {
		if i, ok := v.objRow[n]; ok {
			v.intents.SelectObject(i)
		}
		if n == v.del {
			v.intents.DeleteObject()
		}
	}
}
