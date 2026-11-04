package kit

import (
	"github.com/merlindorin/ember/input"
	"github.com/merlindorin/ember/ui"
)

// Hint is a line of help that reads for the device the player used last:
// a span of class "kit-hint" holding a span of class hint-<device> per text
// given, shown by kit's stylesheet from the data-input-device that ui
// keeps on html, a game may also set on a panel to show it for one device
// whatever was used (an ancestor's own attribute wins over html's). Give
// KeyboardHint always: a device without a text of its own shows it, as
// does the page before any input.
func Hint(args ...ui.Arg) *ui.Node {
	return ui.Span(append([]ui.Arg{ui.Class("kit-hint")}, args...)...)
}

// KeyboardHint, MouseHint, GamepadHint and TouchHint are a Hint's text for
// each device.
func KeyboardHint(text string) ui.Arg { return deviceHint{input.KeyboardDevice, text} }
func MouseHint(text string) ui.Arg    { return deviceHint{input.MouseDevice, text} }
func GamepadHint(text string) ui.Arg  { return deviceHint{input.GamepadDevice, text} }
func TouchHint(text string) ui.Arg    { return deviceHint{input.TouchDevice, text} }

type deviceHint struct {
	device input.Device
	text   string
}

func (h deviceHint) Apply(n *ui.Node) { SetHint(n, h.device, h.text) }

// SetHint makes hint read text for device d.
func SetHint(hint *ui.Node, d input.Device, text string) {
	if s := hintSpan(hint, d); s != nil {
		s.SetText(text)
		return
	}
	hint.AddClass("kit-has-" + d.String())
	hint.Append(ui.Span(ui.Class("kit-hint-"+d.String()), ui.Text(text)))
}

// HintText is what hint reads for device d, "" when it has no text of its
// own for it.
func HintText(hint *ui.Node, d input.Device) string {
	if s := hintSpan(hint, d); s != nil {
		return s.Text()
	}
	return ""
}

func hintSpan(hint *ui.Node, d input.Device) *ui.Node {
	for _, c := range hint.Children() {
		if !c.IsText() && c.HasClass("kit-hint-"+d.String()) {
			return c
		}
	}
	return nil
}
