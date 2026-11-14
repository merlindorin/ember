package kit

import (
	"strconv"

	"github.com/lantervale/ember/automation"
	"github.com/lantervale/ember/ui"
)

// TextInput is an input of class "kit-text-input"
// holding text. the old ember/ui's options are args here: Password, Search,
// ReadOnly, MaxLength, and ui.Placeholder.
func TextInput(text string, args ...ui.Arg) *ui.Node {
	return ui.Input(append([]ui.Arg{ui.Type("text"), ui.Class("kit-text-input"), ui.Value(text)}, args...)...)
}

// Password hides a TextInput's text behind bullets and never copies it.
func Password() ui.Arg { return ui.Type("password") }

// Search makes a TextInput a search field: "Search" while it is empty
// unless a placeholder says otherwise, and Escape empties it.
func Search() ui.Arg {
	return args{ui.Type("search"), ui.Class("kit-search"), ui.Placeholder("Search")}
}

// args are several args given as one.
type args []ui.Arg

func (a args) Apply(n *ui.Node) {
	for _, arg := range a {
		arg.Apply(n)
	}
}

// ReadOnly lets a TextInput's text be selected and copied, not edited.
func ReadOnly() ui.Arg { return ui.Attr("readonly", "") }

// MaxLength keeps a TextInput or TextArea to n characters.
func MaxLength(n int) ui.Arg { return ui.Attr("maxlength", strconv.Itoa(n)) }

// SearchField is the old ember/ui's search field: a span of class "kit-search-field"
// holding a TextInput with Search, and its clear button, an × at its
// right shown only while it holds text, which empties it, the input
// emitting ui.Edited and ui.Changed, and leaves the focus in it.
// Automation names the button "clear", under the field's data-name: the
// input's id, else its placeholder's slug, else "kit-search". args go on the
// input; the button clears the input beside it, by its id when args give one.
func SearchField(text string, args ...ui.Arg) *ui.Node {
	in := TextInput(text, append([]ui.Arg{Search()}, args...)...)
	clearButton := ui.Button(ui.Type("reset"), ui.Class("kit-search-clear"), ui.Attr("aria-label", "clear"),
		ui.TabIndex(-1), ui.Text("×"))
	if in.ID() != "" {
		clearButton.SetAttr("aria-controls", in.ID())
	}
	if text == "" {
		clearButton.SetAttr("hidden", "")
	}
	name := in.ID()
	if p, _ := in.Attr("placeholder"); name == "" {
		name = automation.Slug(p)
	}
	if name == "" {
		name = "search"
	}
	return ui.Span(ui.Class("kit-search-field"), ui.Attr("data-name", name), in, clearButton)
}

// SearchInput is a SearchField's input, whose value and events the
// field's are.
func SearchInput(field *ui.Node) *ui.Node { return field.Children()[0] }
