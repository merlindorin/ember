# Text input

![Text inputs with a value and with a placeholder, in each state: default, hover, focus, focus-visible, read-only, password, search](text-input.png "ember · text input")

<!-- chrome 14-textinput -->
Matches Chrome within 0.30% of its pixels (threshold 0.40%).
<!-- /chrome -->

```go
name := kit.TextInput("", ui.Placeholder("Hero name"), kit.MaxLength(16))
```

| Option | Default | |
|---|---|---|
| `ui.Placeholder("Hero name")` | none | muted text while empty |
| `kit.MaxLength(16)` | no limit | typing stops at 16 characters |
| `kit.Password()` | plain | bullets; never copied |
| `kit.ReadOnly()` | editable | selected and copied, not edited |
| `kit.Search()` | plain | a magnifier, "Search" while empty, Escape empties it |

Text longer than the field scrolls sideways to keep the caret in view.

```go
s := d.Value(name)
d.SetValue(name, "Heron")
```

While it's focused, a field keeps what the player is typing; once it has changed or been submitted, `SetValue` sets it, as a chat clearing its field on Enter.

| Event | When |
|---|---|
| `ui.Edited{Node}` | every keystroke, paste, cut or undo |
| `ui.Changed{Node}` | Enter, or the focus leaving, after an edit |
| `ui.Submitted{Node}` | Enter |

| CSS | |
|---|---|
| `.kit-text-input` | its look, with `::placeholder`, `:focus` and `:focus-visible` |
| `.kit-text-input.kit-search` | the magnifier and its room |

Automation: its `id`, else its `aria-label`, else a `<label>` round it, else its placeholder (`hero-name`); `type`, `set`, `select-all`, `submit`.

## Search field

![Search fields with text and their clear button, empty, hovered and focused](text-input-search.png "ember · search field")

<!-- chrome 14b-search -->
Matches Chrome within 0.26% of its pixels (threshold 0.38%).
<!-- /chrome -->

```go
find := kit.SearchField("", ui.ID("find"))
q := d.Value(kit.SearchInput(find))
```

| Event | When |
|---|---|
| `ui.Edited{Node}`, `ui.Changed{Node}` | the ×, shown while it holds text, pressed: it empties the input (`Node`) |

| CSS | |
|---|---|
| `.kit-search-field` | the span round input and button |
| `.kit-search-clear` | the × button |

Automation: the input as above; the button is `clear`, `press`.

## Shortcodes

![A field holding "Nice one :th", its list open on four emoji, the second picked](text-input-shortcodes.png "ember · shortcodes")

<!-- chrome 14b-shortcodes -->
Matches Chrome within 0.49% of its pixels (threshold 0.64%).
<!-- /chrome -->

```go
d := ui.NewDocument(kit.Theme(), ui.Emoji(set))
chat := kit.ShortcodeField(kit.TextInput("", ui.Placeholder("Say something")))
```

| Key | |
|---|---|
| `:` after a name | turns `:name:` into its emoji |
| Up, Down, the pad | walk the list (`kit.SuggestionList(chat)`) |
| Tab, Enter, South, a click | put the picked emoji in: `ui.Edited` |
| Escape, East | close the list |

| CSS | |
|---|---|
| `.kit-shortcode-field` | the span round field and list |
| `.kit-suggestions` | the list, with `[hidden]`; under the field, `[data-side="above"]` above it when there's no room below |
| `.ember-suggestion` | one emoji and its name, with `[aria-selected="true"]` and `:hover` |
| `.ember-suggestion-emoji`, `.ember-suggestion-name` | its two parts |

Automation: the list is the field's name + `-suggestions` (its `id`, `data-name`, `aria-label` or placeholder slug, else `shortcodes`), each suggestion an option in it by its name: `say-something-suggestions/fire`, `press`. Two shortcode fields in one Document need different names. The `ember-` classes are ui's own, apart from a game's and kit's `kit-`.

```
go run ./examples/92-ui-kit -element text-input
```
