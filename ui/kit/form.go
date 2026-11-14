package kit

import (
	"strconv"
	"strings"

	"github.com/lantervale/ember/automation"
	"github.com/lantervale/ember/ui"
)

// Form is a div of class "kit-form", a grid of a column of
// labels, right-aligned and muted, beside a column of fields, each row a
// Field, with Separators (class "kit-divider") across. Fields stretch across
// their column unless SetFieldWidth gives them one width; a field with a
// width of its own keeps it.
func Form(rows ...*ui.Node) *ui.Node {
	f := ui.Div(ui.Class("kit-form"))
	f.Append(rows...)
	return f
}

// Field is a form's row: a label element of classes "kit-label" and
// "kit-form-label" reading label, then field, given class "kit-form-field". The
// label is for field's control (its first input, select or textarea, else
// field), which gets the id slug of label when it has none, so automation
// names it by its label.
func Field(label string, field *ui.Node) *ui.Node {
	field.AddClass("kit-form-field")
	control := controlIn(field)
	if control.ID() == "" {
		control.SetID(automation.Slug(label))
	}
	text := ui.Label(ui.Class("kit-label", "kit-form-label"), ui.For(control.ID()), ui.Text(label))
	return ui.Div(ui.Class("kit-form-row"), text, field)
}

// controlIn is the first input, select or textarea from n down, else n.
func controlIn(n *ui.Node) *ui.Node {
	found := n
	n.Walk(func(c *ui.Node) bool {
		if found != n {
			return false
		}
		switch c.Tag().String() {
		case "input", "select", "textarea":
			found = c
		}
		return found == n
	})
	return found
}

// FieldLabel is the label of field's row, nil when field isn't in a row.
func FieldLabel(field *ui.Node) *ui.Node {
	row := field.Parent()
	if row == nil || !row.HasClass("kit-form-row") {
		return nil
	}
	return row.Children()[0]
}

// SetLabelWidth makes form's label column at least w points wide, so
// forms side by side or one above another line their fields up.
func SetLabelWidth(form *ui.Node, w float32) { setVar(form, "--label-width", px(w)) }

// SetFieldWidth gives form's fields without a width of their own w
// points, 0 to stretch them across their column again.
func SetFieldWidth(form *ui.Node, w float32) {
	if w <= 0 {
		form.RemoveClass("kit-fixed-fields")
		setVar(form, "--field-width", "")
		return
	}
	form.AddClass("kit-fixed-fields")
	setVar(form, "--field-width", px(w))
}

func px(w float32) string { return strconv.FormatFloat(float64(w), 'f', -1, 32) + "px" }

// setVar sets the custom property name in n's style attribute to value,
// or removes it when value is empty, keeping its other declarations.
func setVar(n *ui.Node, name, value string) {
	var kept []string
	for _, d := range strings.Split(n.Style(), ";") {
		if d = strings.TrimSpace(d); d != "" && !strings.HasPrefix(d, name+":") {
			kept = append(kept, d)
		}
	}
	if value != "" {
		kept = append(kept, name+": "+value)
	}
	n.SetStyle(strings.Join(kept, "; "))
}
