// Emoji: a chat box. Each message shows twice, once in a monochrome emoji
// font tinted like the text and once in colour from a set of pictures,
// families, skin tones, flags and keycaps included. Type below, with the
// system's emoji picker, a paste, or a :shortcode: (:fire:, :+1:): a list
// of matching names opens as one is typed, picked with the arrows and Tab.
// Enter sends; backspace deletes a whole emoji at a time.
//
//	go run ./examples/46-emoji
package main

import (
	"embed"
	"log"

	"github.com/lantervale/ember"
	"github.com/lantervale/ember/color"
	"github.com/lantervale/ember/platform"
	"github.com/lantervale/ember/twod"
	"github.com/lantervale/ember/ui"
	"github.com/lantervale/ember/ui/kit"
)

//go:embed fonts emoji
var files embed.FS

// greeting is what the chat starts with.
func greeting() []string {
	return []string{
		"Party: 👨‍👩‍👧 🧙‍♂️ 👍🏽 👋🏾",
		"Flags: 🇫🇷 🇯🇵 🇧🇷 🇨🇦 🏳️‍🌈",
		"Loot: 💎 🗡️ 🛡️ 🏹 🍖 💰 1️⃣",
		"Shortcodes: :fire: :tada: :wave::skin-tone-3: :+1:",
	}
}

func main() {
	win := platform.NewWindow(
		platform.Title("ember · emoji"),
		platform.Size(640, 420),
		platform.FixedSize(),
		platform.Background(color.Hex("#1d2129")),
	)
	mono, colour, err := faces()
	if err != nil {
		log.Fatal(err)
	}
	chat := newChat(mono, colour)
	win.Add(
		twod.NewText(
			"Monochrome",
			twod.At(20, 20),
			twod.Pivot(0, 0),
			twod.FontSize(13),
			twod.Color(color.Hex("#8a93a6")),
		),
		chat.mono,
		twod.NewText("Colour", twod.At(20, 190), twod.Pivot(0, 0), twod.FontSize(13), twod.Color(color.Hex("#8a93a6"))),
		chat.colour,
		inputPage(colour.Emoji(), chat),
	)
	ember.New().AddEntities(win).MustRun()
}

// page lays the input along the window's foot, over the game's background.
const page = `
body { background-color: transparent; }
.kit-shortcode-field { position: absolute; left: 20px; top: 366px; }
#message { width: 600px; height: 34px; font-size: 18px; }
`

// inputPage is the Document holding the chat's input: the kit's look, the
// colour pictures for its emoji, and a text input whose :shortcodes: turn
// into emoji, which sends to the chat on Enter.
func inputPage(pictures *twod.EmojiSet, chat *Chat) *ui.Document {
	doc := ui.NewDocument(kit.Theme(), ui.CSS(page), ui.Emoji(pictures))
	input := kit.TextInput(
		"",
		ui.ID("message"),
		ui.Placeholder("Say something 🙂, or type :fi…"),
		ui.Attr("autofocus", ""),
		ui.OnSubmit(func(e ui.Event) {
			chat.Send(doc.Value(e.Node))
			doc.SetValue(e.Node, "")
		}),
	)
	doc.Body().Append(kit.ShortcodeField(input))
	return doc
}

// faces are the engine's Regular with Noto Emoji behind it, tinted like
// text, and the same with Twemoji's pictures in front, in colour.
func faces() (*twod.Face, *twod.Face, error) {
	emoji, err := twod.LoadFace(files, "fonts/NotoEmoji.ttf")
	if err != nil {
		return nil, nil, err
	}
	pictures, err := twod.LoadEmoji(files, "emoji")
	if err != nil {
		return nil, nil, err
	}
	mono := twod.Regular().With(twod.Fallback(emoji))
	colour := twod.Regular().With(twod.Fallback(emoji), twod.Emoji(pictures))
	return mono, colour, nil
}
