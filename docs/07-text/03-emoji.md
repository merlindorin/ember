# Emoji

Text can hold emoji: a chat line, a player's name, a loot label. Here a chat box shows each message twice, once in a monochrome emoji font tinted like the text, and once in colour from a set of pictures. Type below and press Enter to send.

![A chat in monochrome and in colour: families, skin tones, flags, keycaps and shortcodes, and a message being typed](emoji.png "ember · emoji")

## Fallback faces

A face draws only the characters it has. `twod.Regular()` has letters but no emoji, so a second face fills in: `twod.Regular().With(twod.Fallback(emoji))` draws each character with the first face that has it, Regular for the letters and Noto Emoji for the rest. `LoadFace` and `NewFace` take `Fallback(…)` too, and `Fallbacks()` reads them back. A monochrome emoji is a glyph like any letter, so it takes the text's colour: here, gold.

## Colour emoji

Colour emoji are pictures. `twod.LoadEmoji(files, "emoji")` reads a folder of PNGs, each named after its characters: Twemoji's `1f1eb-1f1f7.png` for 🇫🇷, or Noto's `emoji_u…` names. Adding `twod.Emoji(set)` to a face draws those as pictures, at the size of the text and in their own colours, untinted. `set.Has("👍🏽")` says whether one is in the set.

[main.go](../../examples/45-emoji/main.go)

## One emoji, one unit

Some emoji are several characters drawn as one: a family 👨‍👩‍👧 is three people joined, a skin tone 👍🏽 is a hand and a tone, a flag 🇫🇷 is two letters, a keycap 1️⃣ is a digit and a box. Text measures, wraps and deletes each of these as one unit, so a backspace in the input takes the whole family off. The input is a ui Document's text field: `ui.Emoji(pictures)` on the Document draws its emoji from the same pictures, and `ui.OnSubmit` sends the message to `Chat` and empties the field. The system's emoji picker types into it too: Ctrl+Cmd+Space on macOS. Cmd+V or Ctrl+V pastes, emoji whole and line breaks as spaces, and Cmd+C or Ctrl+C copies.

[chat.go](../../examples/45-emoji/chat.go)

## Shortcodes

Players type emoji by name, as in Slack or GitHub: `:fire:` is 🔥, `:+1:` is 👍, and `:wave::skin-tone-3:` is 👋🏽. The names are GitHub's, the same as Slack's.

- In a text input, `kit.ShortcodeField(input)` replaces a name as its closing colon is typed. While a name is being typed, a list shows up to six emoji that match, shortest name first, under the field, or above it when there's no room below, as here. The arrows or a pad pick one; Tab, Enter or the pad's south button puts it in; Escape or the pad's east button closes the list. Its automation ID is the field's name + `-suggestions`: `message-suggestions`.
- In a text, `twod.ExpandShortcodes()` turns names into emoji when it draws. `twod.Shortcodes(s)` does the same to a string. A name it doesn't know stays as written.
- In the UI, `kit.ExpandShortcodes()` does the same for labels: `kit.Label("gg :fire:", kit.ExpandShortcodes())` shows 🔥, and its text stays as written.

![The list of matches above the input while ":fi" is typed](shortcodes.png "ember · emoji")

The list only offers emoji the input's face can draw whole. The example carries 108 pictures, so it offers few; a game that loads a full set gets every name.

## What isn't supported

- **Joined emoji in monochrome.** A monochrome emoji font draws characters one by one, with no shaping, so a family shows only its first person and a flag only its first letter. The colour pictures show them whole, as in the capture.
- **Bitmap fonts.** They have no fallbacks.
- **Emoji outside the pictures, in the input.** The input draws letters in Go and emoji only from the Document's pictures: one typed from the picker that the set lacks shows no Noto glyph there, though the chat lines, twod texts with both faces, still show it.
- **Emoji in the engine.** ember ships none. A game loads the set it needs and keeps its licence: Twemoji's pictures are CC-BY 4.0, which asks for credit, and Noto Emoji is under the SIL Open Font License. The example's `CREDITS.md` names both.

## Try it

[45-emoji](../../examples/45-emoji)
