package kit

import "github.com/lantervale/ember/ui"

// The looks below are kit's variants of its elements, given as args where
// the element takes them. Each is a class with kit's prefix, kit-, so a
// game's own classes and kit's never meet: a game reaches kit's looks
// through these, never by a class name.

// Caps sets a Label in the old ember/ui's small capitals; FormLabel is a form's
// muted label; AlertTitle and AlertMessage are an alert's two text styles.
func Caps() ui.Arg         { return ui.Class("kit-caps") }
func FormLabel() ui.Arg    { return ui.Class("kit-form-label") }
func AlertTitle() ui.Arg   { return ui.Class("kit-alert-title") }
func AlertMessage() ui.Arg { return ui.Class("kit-alert-message") }

// Wrap wraps a Label's text in its box; Truncate cuts it there with an
// ellipsis; Center and Right align it.
func Wrap() ui.Arg     { return ui.Class("kit-wrap") }
func Truncate() ui.Arg { return ui.Class("kit-truncate") }
func Center() ui.Arg   { return ui.Class("kit-center") }
func Right() ui.Arg    { return ui.Class("kit-right") }

// Primary and Destructive are a Button's strong and dangerous looks.
func Primary() ui.Arg     { return ui.Class("kit-primary") }
func Destructive() ui.Arg { return ui.Class("kit-destructive") }

// Divider makes a Separator the plain line across forms and dialogs;
// Vertical stands it up between a row's items.
func Divider() ui.Arg  { return ui.Class("kit-divider") }
func Vertical() ui.Arg { return ui.Class("kit-vertical") }

// AlignStart, AlignCenter and AlignEnd place a Row's or Column's children
// across it; JustifyStart, JustifyCenter, JustifyEnd and JustifyBetween
// along it. Expand makes a child grow into the room left; Fill makes it
// fill its container across, and a Choice share its width among its items.
func AlignStart() ui.Arg     { return ui.Class("kit-align-start") }
func AlignCenter() ui.Arg    { return ui.Class("kit-align-center") }
func AlignEnd() ui.Arg       { return ui.Class("kit-align-end") }
func JustifyStart() ui.Arg   { return ui.Class("kit-justify-start") }
func JustifyCenter() ui.Arg  { return ui.Class("kit-justify-center") }
func JustifyEnd() ui.Arg     { return ui.Class("kit-justify-end") }
func JustifyBetween() ui.Arg { return ui.Class("kit-justify-between") }
func Expand() ui.Arg         { return ui.Class("kit-expand") }
func Fill() ui.Arg           { return ui.Class("kit-fill") }

// Body pads a Row or Column as a window's body, the old ember/ui's, between kit's
// Toolbar and Footer.
func Body() ui.Arg { return ui.Class("kit-body") }

// Left shows a Tooltip's tip on its element's left, Below under it; Below
// also puts a Table's caption under its rows.
func Left() ui.Arg  { return ui.Class("kit-left") }
func Below() ui.Arg { return ui.Class("kit-below") }

// Segmented joins a Choice's items as tabs.
func Segmented() ui.Arg { return ui.Class("kit-segmented") }

// Checked marks a Panel as the chosen one.
func Checked() ui.Arg { return ui.Class("kit-checked") }

// Keep, KeepCentered, KeepAspect, KeepAspectCentered and KeepAspectCovered
// are an Image's stretch modes, the old ember/ui's.
func Keep() ui.Arg               { return ui.Class("kit-keep") }
func KeepCentered() ui.Arg       { return ui.Class("kit-keep-centered") }
func KeepAspect() ui.Arg         { return ui.Class("kit-keep-aspect") }
func KeepAspectCentered() ui.Arg { return ui.Class("kit-keep-aspect-centered") }
func KeepAspectCovered() ui.Arg  { return ui.Class("kit-keep-aspect-covered") }
