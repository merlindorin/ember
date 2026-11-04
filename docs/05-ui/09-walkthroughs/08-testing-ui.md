# Testing UI

A settings form that saves when Save is clicked, and the tests that fill it in. A World runs without opening a window, so `go test` drives the UI frame by frame: once as a player would, with the mouse and keyboard, and once through automation, by each element's ID.

![The settings form: a name, a volume slider, a music check, a Save button and its status](testing-ui.png "ember · testing ui")

## The form

`newForm` builds a `kit.Panel` with the `id` `settings`, a `kit.Form` of the fields, Save and the status. Every control a test drives has an `id`, so its automation ID is a plain path, `settings/name`, `settings/volume`, `settings/save`, whatever its text. Save's `ui.OnClick` reads the controls with `doc.Value` and `doc.Checked` into the game's `Settings`, and says so in the status.

[form.go](../../../examples/18-testing-ui/form.go)

[main.go](../../../examples/18-testing-ui/main.go)

## The tests

`game` builds the same form in a window that is never shown, and `RunFrames(1)` runs one frame of it. Input set on the window between frames, `SetMouse`, `SetMouseButton` and `AddText`, arrives with the next frame, as a real event would, so a click is three frames: the pointer over the control, the button down, the button up. `doc.ByID` and `doc.BoxOf` say where a control is.

The second test asks automation instead. `automation.Recording()` records the Document each frame, and `Queue` hands an element an act: `ActType` types into a field, `ActSet` sets a slider or a check, `ActPress` presses a button. `Snapshot` lists what was drawn, each element with its ID, role, label and value.

[main_test.go](../../../examples/18-testing-ui/main_test.go)

The same IDs drive a running game from outside: with `EMBER_AUTOMATION=127.0.0.1:50551` set, the automation server takes the same acts over gRPC ([Automation](../06-automation.md)).

## Lining up

`align.Check` says what a laid-out Document has out of line, so a test catches a crooked screen as it catches a wrong value:

```go
doc.Layout(geom.V2(520, 320))
for _, f := range align.Check(doc) {
	t.Error(f)
}
```

| Rule | Checks |
|---|---|
| `height` | controls sharing a flex row are one height; a control kept taller on purpose carries `align.OwnHeight` as a class |
| `centre` | a control's label sits in the middle of it; not a control that stacks several blocks, such as a card button or an icon over its label |
| `baseline` | the single-line texts of a row, and a label beside a block of lines, share a baseline |
| `separator` | every separator of a row stands in the same space |
| `outside` | nothing runs past the window's edge |
| `overflow` | no text or control runs past the box that holds it, unless that box clips, scrolls or cuts with an ellipsis |
| `spill` | no box runs past an ancestor whose width or height is fixed, before any scroll container; reported once, for the outermost box; a spill meant, such as a badge straddling a frame, carries `align.OverflowOK` (`data-overflow-ok`) on the box or one between, absolutely placed ones included |

Each `align.Fault` names its `Rule`, the automation `Path` of what breaks it, and `Got` and `Want` in points. Examples 92 and 93 run it on every page.

## Beyond the UI

Two packages test the rest of a game the same way:

| | |
|---|---|
| `twod/twodtest` | what a window drew after a frame, for tests only: `Drawn(w, e)` an entity's transform in window points, `Passes` and `Shown` the offscreen pictures and the images on screen, `LightingOf` its ambient, lights and shadow casters, `AnswerProbes` standing in for the GPU's light probes |
| `golden` | a game's screens against reference pictures on a GPU machine: `golden.Run(t, ".", "testdata/golden", []golden.Scene{{Name: "title", Pkg: ".", Frame: 60}})` captures each scene hidden, with scripted input, and compares it within one level a channel; `-update` rewrites the references |

```sh
go test ./examples/18-testing-ui
```

## Try it

[18-testing-ui](../../../examples/18-testing-ui)
