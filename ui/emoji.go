package ui

import "github.com/merlindorin/ember/twod"

// Emoji draws set's colour emoji in the Document's text, as twod does for
// a face with twod.Emoji: each one a picture one em wide, laid out as
// wide as it draws. The engine ships no emoji; a game loads Twemoji or
// Noto PNGs with twod.LoadEmoji.
func Emoji(set *twod.EmojiSet) DocumentOption {
	return func(d *Document) {
		d.emoji = set
		d.fonts.SetEmoji(set.Has)
	}
}
