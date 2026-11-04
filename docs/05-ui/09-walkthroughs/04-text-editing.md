# Text editing

A notes app: a title and a note that edit the way any desktop app does. You select with Shift or the mouse, jump by word, cut, copy and paste, and undo. A long note scrolls, keeping the caret in view. The word count follows as you type.

![New lines pressed until the note scrolls, its scrollbar showing, then Cmd+A selecting every line](text-editing.gif "ember · text editing")

## One line, or many

`kit.TextInput` edits one line, as on the [Controls](03-controls.md) page. `kit.TextArea` edits many: it wraps long lines at words, scrolls with the wheel, the trackpad or its scrollbar, and keeps the caret in view; `kit.Rows(12)` is its natural height in lines. A kit text field has a width of its own, so the page's sheet gives the title and the note `width: 100%` of the panel. The note's `ui.OnInput` updates the count at each key, and the Wrap check's `ui.OnChange` calls `kit.SetNoWrap`: unchecked, the note's lines stay whole and scroll sideways. Their other options, `MaxLength`, `ReadOnly` and `Password`, are on the [Text input](../07-kit/14-text-input.md) and [Text area](../07-kit/15-text-area.md) pages.

[main.go](../../../examples/14-text-editing/main.go)

![Wrap unchecked: the note's lines stay whole, with a scrollbar along the bottom](no-wrap.png "ember · text editing")

`countOf` counts the words and characters the count shows.

[count.go](../../../examples/14-text-editing/count.go)

## Keys

The keys follow the platform the game runs on:

| | macOS | Windows, Linux |
|---|---|---|
| Move a character, a word | ←/→, Alt+←/→ | ←/→, Ctrl+←/→ |
| Line start, end | Cmd+←/→, Home/End | Home/End |
| Text start, end | Cmd+↑/↓ | Ctrl+Home/End |
| Up and down a line (area) | ↑/↓, PageUp/PageDown | ↑/↓, PageUp/PageDown |
| Select | Shift with any move, Cmd+A for all | Shift with any move, Ctrl+A for all |
| Delete | Backspace/Delete, Alt+Backspace a word, Cmd+Backspace to the line start | Backspace/Delete, Ctrl+Backspace a word |
| Cut, copy, paste | Cmd+X/C/V | Ctrl+X/C/V |
| Undo, redo | Cmd+Z, Shift+Cmd+Z | Ctrl+Z, Ctrl+Y or Shift+Ctrl+Z |
| Send (area): `ui.Submitted` | Cmd+Enter | Ctrl+Enter |

Typing replaces the selection. Undo takes back a whole typed run at once, and each other edit on its own. Paste keeps emoji whole; it keeps line breaks in an area, and turns them into spaces in an input.

## The mouse

A click places the caret, and dragging selects. A double click takes a word and a triple click a line. Shift and a click extends the selection to there.

## What isn't supported

- **Composing Japanese, Chinese or Korean.** Characters the system types still arrive, but not the underlined text shown while a word is being composed.

## Try it

[14-text-editing](../../../examples/14-text-editing)
