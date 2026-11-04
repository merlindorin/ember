// 14b Shortcodes: kit.ShortcodeField with ":th" typed, its list open on the four
// emoji the page's set draws, the second picked, as ui fills it, on ember's window colour.
package main

import (
	"embed"

	"testing/fstest"

	"github.com/merlindorin/ember/internal/emojinames"
	"github.com/merlindorin/ember/twod"
	"github.com/merlindorin/ember/ui"
	"github.com/merlindorin/ember/ui/internal/pages"
	"github.com/merlindorin/ember/ui/kit"
)

//go:embed emoji
var emoji embed.FS

const sheet = `body { background-color: var(--window-color); padding: 16px; }`

func main() {
	set, err := twod.LoadEmoji(emoji, "emoji")
	if err != nil {
		panic(err)
	}
	d := ui.NewDocument(kit.Theme(), ui.Emoji(set), ui.Stylesheet(fstest.MapFS{"page.css": {Data: []byte(sheet)}},
		"page.css"))
	field := kit.ShortcodeField(kit.TextInput("Nice one :th"))
	list := kit.SuggestionList(field)
	for i, m := range emojinames.Matching("th", 6, func(e emojinames.Entry) bool { return set.Has(e.Emoji) }) {
		selected := "false"
		if i == 1 {
			selected = "true"
		}
		list.Append(ui.Div(ui.Class("ember-suggestion"), ui.Attr("role", "option"), ui.Attr("aria-selected", selected),
			ui.Span(ui.Class("ember-suggestion-emoji"),
				ui.Text(m.Emoji)), ui.Text(" "), ui.Span(ui.Class("ember-suggestion-name"), ui.Text(":"+m.Name+":"))))
	}
	list.RemoveAttr("hidden")
	field.Children()[0].SetAttr("aria-expanded", "true")
	d.Body().Append(field)
	pages.Run(d, 300, 200)
}
