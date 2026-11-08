package ui

import (
	"slices"

	"github.com/merlindorin/ember/geom"
	"github.com/merlindorin/ember/input"
	"github.com/merlindorin/ember/internal/inputfeed"
	"github.com/merlindorin/ember/internal/textedit"
	"github.com/merlindorin/ember/platform"
	"github.com/merlindorin/ember/twod"
	"github.com/merlindorin/ember/ui/control"
	"github.com/merlindorin/ember/ui/internal/box"
	"github.com/merlindorin/ember/ui/internal/dom"
	"github.com/merlindorin/ember/ui/internal/style"
	"github.com/merlindorin/ember/ui/paint"
	"github.com/merlindorin/ember/world"
)

// interaction is what the pointer, the keys and the focus are doing to a
// Document.
type interaction struct {
	machines map[*Node]any
	owner    owner
	// pendingFocus is the element a Focus asked for before it could take
	// the focus, given it at the next retarget where it can.
	pendingFocus *Node
	// pendingReveal is the element a ScrollIntoView asked to show.
	pendingReveal *Node
	// opening is set by ShowModal: the keys and pad buttons down at the
	// next read, openKeys and openPad, opened the modal and act in it only
	// once released.
	opening  bool
	openKeys []input.Key
	openPad  []input.GamepadButton
	// controls are the elements with a machine the last walk of the tree
	// found in it, in tree order.
	controls []*Node
	// targeted is whether the focus was given its targets once, and
	// retargeting whether the last interact changed something, laid out
	// only after the targets were taken.
	targeted, retargeting bool
	// candidates are the elements that take the focus as the tree and its
	// attributes stand, and shows the tabs and reset buttons that show or
	// hide what they control: found again when restructure, set by css
	// meeting a tree or attribute change. targets are the focus's targets,
	// two in turn, so the focus reads the last while the next is made.
	candidates []candidate
	shows      []*Node
	// shortcutItems are the menu items with aria-keyshortcuts, parsed their
	// keys as last read.
	shortcutItems []*Node
	parsed        map[*Node]parsedKeys
	// armed is the keys that choose an item with the modifiers last held,
	// captured each frame.
	armed       []int
	restructure bool
	// autofocused are the elements with autofocus that had their chance.
	autofocused map[*Node]bool
	targets     [2][]control.Target[*Node]
	flip        int
	focus       control.Focus[*Node]
	// hovered and pressed are the element chains under the pointer and
	// held by its left button, innermost first.
	hovered, pressed []*Node
	// pointerClick is the element a pointer click is on while it is
	// handled, and lastClick, lastClickAt and clicks the run of clicks it
	// counts in (Clicked.Clicks).
	pointerClick, lastClick *Node
	lastClickAt             float32
	clicks                  int
	// pointerAt is where the pointer was this frame, in the Document's
	// space.
	pointerAt geom.Vec2
	// stated are the elements states() gave a pointer or focus state.
	stated   []*Node
	platform textedit.Platform
	events   []any
	// scrolls are the scroll containers' thumb drags, and dragging the
	// container whose thumb the pointer holds.
	// pinned are pseudo-classes a test page holds on, so a still
	// picture shows a hovered or focused look (assets.Pin).
	pinned   map[*Node]dom.State
	scrolls  map[*Node]*paint.Scrolling
	dragging *Node
	// scrolled is set when input moved a scroll offset since the last
	// layout, so the boxes\' rects are copied again.
	scrolled bool
	// pressOpened is the menu title or select a press opened, until its
	// release: over an item or option it chooses it, back on the opener it
	// leaves the menu open, over nothing it closes it (pressOpen). As with
	// native menus, the release skips the opener's own click: an OnClick
	// or Clicked on a menu bar title doesn't fire for a press and release
	// on it.
	pressOpened *Node
	// drag is a press on a Draggable element, a drag once past dragStart;
	// rightPressed the element the right button went down on (decision 0062).
	drag         *dragState
	rightPressed *Node
	// caretSeen is the focused text field's caret and length the last
	// time it was scrolled into view (followCaret).
	caretSeen caretState
	// modals are the dialogs open modal, the newest last; focusing is a
	// dialog just shown, until its focus (and trap, when modal) is set.
	modals   []*Node
	focusing *Node
	// menuFocus is a menu opened from the keys whose first item (last with
	// menuFocusLast) takes the focus once it is laid out.
	menuFocus     *Node
	menuFocusLast bool
	// suggests are the shortcode fields' offers.
	suggests map[*Node]*suggestions
	// revealing is a grid row the game selected, scrolled into view once
	// laid out.
	revealing *Node
	// sorts are the grids' own row orders (SortBy).
	sorts map[*Node]func(a, b *Node, column int, descending bool) int
	// lastPress is the grid row or listbox option clicked last, at
	// lastPressAt on the clock, for a double click.
	lastPress   *Node
	lastPressAt float32
	// held is a window dragged by its bar or an edge; raised is the window raised last,
	// at raisedZ.
	held    *windowDrag
	raised  *Node
	raisedZ int
}

// navKeys and navPad are what focus navigation reads while it shows.
//
//nolint:gochecknoglobals // read-only key tables, built once
var (
	navKeys = []int{int(input.KeyUp), int(input.KeyDown), int(input.KeyLeft), int(input.KeyRight),
		int(input.KeyTab), int(input.KeyEnter), int(input.KeySpace), int(input.KeyEscape)}
	navPad = []int{int(input.GamepadUp), int(input.GamepadDown), int(input.GamepadLeft), int(input.GamepadRight),
		int(input.GamepadLeftStickLeft), int(input.GamepadLeftStickRight),
		int(input.GamepadLeftStickUp), int(input.GamepadLeftStickDown),
		int(input.GamepadSouth), int(input.GamepadEast)}
	intentKeys = []input.Key{
		input.KeyEnter, input.KeyPadEnter, input.KeySpace, input.KeyEscape, input.KeyUp, input.KeyDown,
		input.KeyLeft, input.KeyRight, input.KeyHome, input.KeyEnd, input.KeyPageUp, input.KeyPageDown}
	intentPad = []input.GamepadButton{input.GamepadSouth, input.GamepadEast, input.GamepadUp, input.GamepadDown,
		input.GamepadLeft, input.GamepadRight, input.GamepadLeftStickUp, input.GamepadLeftStickDown,
		input.GamepadLeftStickLeft, input.GamepadLeftStickRight}
)

func (d *Document) focused() *Node { return d.focus.Key() }

// Focused reports whether n holds its window's focus.
func (d *Document) Focused(n *Node) bool { return n != nil && d.focused() == n }

// Focus gives n the focus, shown as when reached by Tab. One that can't
// take it yet, such as an element shown this frame and not laid out, gets
// it at the first layout where it can, unless another Focus came first or
// n has left the Document by then.
func (d *Document) Focus(n *Node) {
	d.pendingFocus = nil
	if !d.moveFocus(func() bool { return d.focus.Set(n, true) }) {
		d.pendingFocus = n
	}
}

// ScrollIntoView scrolls the containers holding n by the least that shows
// it, as the DOM's scrollIntoView({block: "nearest"}). An element not laid
// out yet, such as one shown this frame, is shown once it is; one that
// leaves the Document first is forgotten.
func (d *Document) ScrollIntoView(n *Node) { d.pendingReveal = n }

// revealPending does what a ScrollIntoView asked once n has a box.
func (d *Document) revealPending() {
	n := d.pendingReveal
	if n == nil {
		return
	}
	if !d.inDocument(n) {
		d.pendingReveal = nil
		return
	}
	if d.tree != nil && d.tree.Of[n] != nil {
		d.reveal(n)
		d.pendingReveal = nil
	}
}

// focusPending gives the focus a Focus asked for before n could take it,
// now n is a target, or drops it once n is gone from the Document.
func (d *Document) focusPending() {
	n := d.pendingFocus
	if n == nil {
		return
	}
	if !d.inDocument(n) {
		d.pendingFocus = nil
		return
	}
	if d.moveFocus(func() bool { return d.focus.Set(n, true) }) {
		d.pendingFocus = nil
	}
}

// inDocument reports whether n is in d's tree.
func (d *Document) inDocument(n *Node) bool {
	for p := n; p != nil; p = p.Parent() {
		if p == d.html {
			return true
		}
	}
	return false
}

// focusable reports whether n takes the focus, and its tabindex.
func focusable(n *Node) (int, bool) {
	if n.IsText() || has(n, "disabled") {
		return 0, false
	}
	if s, ok := n.Attr("tabindex"); ok {
		v := int(number(n, "tabindex", 0))
		return v, s != ""
	}
	if isMenuItem(n) {
		return 0, !menuDisabled(n)
	}
	if isTab(n) || isGrid(n) || isListbox(n) {
		return 0, true
	}
	if n.Tag() == dom.Dialog {
		return -1, true
	}
	if !slices.Contains([]dom.Tag{dom.Button, dom.Input, dom.Select, dom.Textarea, dom.Summary}, n.Tag()) {
		return 0, false
	}
	if t, _ := n.Attr("type"); t == "hidden" {
		return 0, false
	}
	return 0, true
}

// candidate is an element that takes the focus, with its own tabindex.
type candidate struct {
	n        *Node
	tabIndex int
}

// retarget gives the focus its targets: what takes the focus, where and in
// what order, and shows or hides the tab panels and reset buttons. Only a
// change css or layout saw, or one interact made, can move them, so a
// frame without one keeps the last. Which elements take the focus changes
// only with the tree or an attribute, so the tree is walked for them only
// then; otherwise the last walk's are placed again, with their tab
// indexes, scopes and rects as they stand.
func (d *Document) retarget() {
	if d.restructure || !d.targeted {
		d.candidates, d.shows, d.shortcutItems = d.candidates[:0], d.shows[:0], d.shortcutItems[:0]
		d.html.Walk(func(n *Node) bool {
			if n.IsText() {
				return false
			}
			if isMenuItem(n) && has(n, "aria-keyshortcuts") {
				d.shortcutItems = append(d.shortcutItems, n)
			}
			if isTab(n) || d.resetField(n) != nil {
				d.shows = append(d.shows, n)
			}
			if i, ok := focusable(n); ok {
				d.candidates = append(d.candidates, candidate{n, i})
			}
			return true
		})
		d.restructure = false
	}
	for _, n := range d.shows {
		if isTab(n) {
			d.showPanel(n)
		} else {
			d.showReset(n)
		}
	}
	d.flip ^= 1
	targets := d.targets[d.flip][:0]
	for _, c := range d.candidates {
		n, i := c.n, c.tabIndex
		if n.Box().W <= 0 {
			continue
		}
		if isTab(n) {
			i = d.tabTabIndex(n, i)
		}
		if d.isRadio(n) {
			i = d.radioTabIndex(n, i)
		}
		if isMenuItem(n) {
			i = d.menuTabIndex(n)
		}
		targets = append(targets, control.Target[*Node]{Key: n, Rect: n.Box(), TabIndex: i, Scope: d.scopeOf(n)})
	}
	d.targets[d.flip] = targets
	d.focus.SetTargets(targets)
	d.autofocus(targets)
	d.focusPending()
	d.targeted = true
}

// autofocus gives the focus to the first target with autofocus, in tree
// order, when nothing has it, as HTML does when a page loads. Each such
// element gets that one chance, when it first becomes a target, so one
// left by the player never takes the focus back.
func (d *Document) autofocus(targets []control.Target[*Node]) {
	for _, t := range targets {
		n := t.Key
		if d.autofocused[n] || !has(n, "autofocus") {
			continue
		}
		if d.autofocused == nil {
			d.autofocused = map[*Node]bool{}
		}
		d.autofocused[n] = true
		if d.focused() == nil {
			d.focus.Set(n, true)
		}
	}
}

// interact runs one frame of input over the laid-out tree, and reports
// whether anything changed that needs another layout before painting.

func (d *Document) interact(c *twod.Canvas) bool {
	d.platform = currentPlatform()
	d.drawnBy(c)
	d.claimed(c)
	if d.replay.repaint || d.retargeting || !d.targeted {
		d.retarget()
	}
	d.revealPending()
	before := d.focused()
	d.layText()
	d.trap()
	if r := d.revealing; r != nil {
		d.revealing = nil
		if g := gridOf(r); g != nil {
			d.revealRow(g, r)
		}
	}
	d.menuFocusNow()
	if before != nil && d.focused() != before {
		d.left(before)
	}
	d.acts(c)
	if in := c.Input(); in != nil {
		d.html.SetAttr("data-input-device", in.LastDevice().String())
		d.pointer(c, in)
		if !d.fresh {
			d.keys(c, in)
		}
	}
	d.states()
	d.syncControls()
	changed := dom.Data(d.html).Dirty != 0
	if w := c.World(); w != nil {
		// A callback may raise events of its own (a back closing a
		// dialog): they join the list, emitted and dispatched in turn.
		for i := 0; i < len(d.events); i++ {
			e := d.events[i]
			emit(w, e)
			d.dispatch(w, e)
		}
	}
	d.events = d.events[:0]
	d.retargeting = changed
	return changed
}

func emit(w *world.World, e any) {
	switch e := e.(type) {
	case Clicked:
		world.Emit(w, e)
	case Edited:
		world.Emit(w, e)
	case Changed:
		world.Emit(w, e)
	case Submitted:
		world.Emit(w, e)
	case Toggled:
		world.Emit(w, e)
	case Closed:
		world.Emit(w, e)
	case Sorted:
		world.Emit(w, e)
	case KeyPressed:
		world.Emit(w, e)
	case BackPressed:
		world.Emit(w, e)
	case PointerEntered:
		world.Emit(w, e)
	case PointerLeft:
		world.Emit(w, e)
	case WindowMoved:
		world.Emit(w, e)
	case WindowResized:
		world.Emit(w, e)
	case WindowFolded:
		world.Emit(w, e)
	case ContextClicked:
		world.Emit(w, e)
	case Dropped:
		world.Emit(w, e)
	case DragEnded:
		world.Emit(w, e)
	}
}

// fire turns a machine's events into the Document's.
func (d *Document) fire(n *Node, e control.Events) {
	if e.Has(control.Clicked) {
		d.events = append(d.events, Clicked{Node: n, Clicks: d.clicking(n)})
	}
	if e.Has(control.Input) {
		d.events = append(d.events, Edited{n})
		d.typing = n
	}
	if e.Has(control.Changed) {
		d.events = append(d.events, Changed{n})
	}
	if e.Has(control.Submitted) {
		d.events = append(d.events, Submitted{n})
	}
	if e.Has(control.Changed) || e.Has(control.Submitted) {
		d.typing = nil
	}
	if e.Has(control.Toggled) {
		d.events = append(d.events, Toggled{n})
	}
	if e.Has(control.Closed) {
		v := ""
		if m, ok := d.machine(n).(*control.Dialog); ok {
			v = m.ReturnValue
		}
		d.events = append(d.events, Closed{n, v})
	}
}

// left is the focus leaving n: a text field commits its edit.
func (d *Document) left(n *Node) {
	d.closeSuggestions(n)
	settle(d.machine(n))
	if t := textOf(d.machine(n)); t != nil {
		d.fire(n, t.Blur())
	}
}

// moveFocus runs a focus move and tells the fields that lose and take it.
func (d *Document) moveFocus(move func() bool) bool {
	before := d.focused()
	moved := move()
	if after := d.focused(); after != before {
		if before != nil {
			d.left(before)
		}
		if t := textOf(d.machine(after)); t != nil {
			t.Focus()
		}
		if after != nil && d.focus.Active() {
			d.reveal(after)
		}
	}
	return moved
}

// click is n activated by the pointer or automation.
func (d *Document) click(n *Node) {
	d.windowAction(n)
	if d.suggestionClick(n) || d.menuClick(n) || d.gridClick(n) || d.listboxClick(n) {
		return
	}
	if d.resetField(n) != nil {
		d.reset(n)
	}
	if n.Tag() == dom.Label {
		if t := d.labelled(n); t != nil {
			d.moveFocus(func() bool { return d.focus.Set(t, false) })
			d.click(t)
		}
		return
	}
	if n.Tag() == dom.Summary {
		if p := n.Parent(); p != nil && p.Tag() == dom.Details {
			if m, ok := d.machine(p).(*control.Details); ok {
				d.fire(p, m.Click())
			}
		}
		d.fire(n, control.Clicked)
		return
	}
	if isTab(n) {
		d.chooseTab(n)
	}
	if d.isRadio(n) {
		d.checkRadio(n)
		if _, native := d.machine(n).(*control.Radio); native {
			return
		}
	}
	switch m := d.machine(n).(type) {
	case *control.Button:
		e := m.Click()
		d.fire(n, e)
		if e.Has(control.Clicked) {
			d.dialogClick(n)
		}
	case *control.Checkbox:
		d.fire(n, m.Click())
	case *control.Select:
		d.fire(n, m.Click())
	case *control.Canvas:
		d.fire(n, control.Clicked)
	case nil:
		if _, ok := n.Attr("tabindex"); ok {
			d.fire(n, control.Clicked)
		}
	}
}

func (d *Document) byID(id string) *Node {
	var found *Node
	d.html.Walk(func(n *Node) bool {
		if found == nil && n.ID() == id {
			found = n
		}
		return found == nil
	})
	return found
}

// chain is n and its element ancestors, innermost first.
func chain(n *Node) []*Node {
	var c []*Node
	for ; n != nil; n = n.Parent() {
		if !n.IsText() {
			c = append(c, n)
		}
	}
	return c
}

// activates reports whether a click on n does something of its own: a
// control (not a details or a dialog), a summary, a label, a tabindex, a
// menu item, a grid row, a sort header or an option.
func (d *Document) activates(n *Node) bool {
	switch {
	case d.machine(n) != nil && n.Tag() != dom.Details && n.Tag() != dom.Dialog:
		return true
	case n.Tag() == dom.Summary || n.Tag() == dom.Label || has(n, "tabindex"):
		return true
	}
	return isMenuItem(n) || isGridRow(n) || isSortHeader(n) || isOption(n)
}

// target is the element a click on chain activates: the innermost with
// behaviour, a summary, a label or a tabindex.
func (d *Document) target(c []*Node) *Node {
	for _, n := range c {
		if d.activates(n) {
			return n
		}
	}
	return nil
}

func (d *Document) pointer(c *twod.Canvas, in *input.State) {
	at := d.mouse(c)
	d.pointerAt = at
	hit, n := d.hitAt(at)
	if !d.replay.quiet || len(d.hovered) == 0 || d.hovered[0] != n {
		was := d.hovered
		d.hovered = chain(n)
		d.crossed(was, d.hovered)
	}
	d.menuHover(d.hovered)
	if n != nil && n != d.html && n != d.body {
		inputfeed.Of(in).CapturePointer()
		d.cover(c)
	}
	if !d.edgeCursor(c, at) {
		d.cssCursor(c)
	}
	d.rightClick(in)
	if d.dragPointer(c, in, at) {
		return
	}
	if d.windowPointer(in, at) {
		return
	}
	if d.wheelNumber(in) {
		return
	}
	if d.scrollPointer(c, in, hit) {
		return
	}
	pick := d.pickAt(at)
	open, _ := d.machine(pick.Select).(*control.Select)
	if open != nil && pick.Index >= 0 && pick.Index < len(open.Options) && !open.Options[pick.Index].Disabled {
		open.Highlight = pick.Index
	}
	if in.Pressed(input.MouseLeft) && open != nil && pick.List {
		if pick.Index >= 0 {
			d.fire(pick.Select, open.ClickOption(pick.Index))
		}
		return
	}
	if in.Pressed(input.MouseLeft) && d.leftPress(in, at, open, pick) {
		return
	}
	if in.Down(input.MouseLeft) && !in.Pressed(input.MouseLeft) && len(d.pressed) > 0 {
		if t := textOf(d.machine(d.pressed[0])); t != nil && d.focused() == d.pressed[0] {
			d.caretAt(d.pressed[0], t, at, true)
		}
		if held, r := d.dragged(); r != nil {
			d.fire(held, r.Drag(d.along(held, at)))
		}
	}
	if in.Released(input.MouseLeft) {
		d.leftRelease(open, pick)
	}
}

// hitAt is the box under at and the innermost element holding it, the
// replay's own while the frame replays quietly.
func (d *Document) hitAt(at geom.Vec2) (*box.Box, *Node) {
	hit := d.replay.hit
	if !d.replay.quiet {
		hit = nil
		if d.tree != nil {
			hit = paint.HitTest(d.tree, at, paint.Controls(d.machine), paint.Viewport(d.viewport))
		}
		d.replay.hit = hit
	}
	var n *Node
	if hit != nil {
		n = hit.Node
		for b := hit; n == nil && b != nil; b = b.Parent {
			n = b.Node
		}
	}
	return hit, n
}

// pickAt is the option of an open select under at, the replay's own while
// the frame replays quietly.
func (d *Document) pickAt(at geom.Vec2) paint.Pick {
	pick := d.replay.pick
	if !d.replay.quiet {
		pick = paint.Pick{Index: -1}
		if d.tree != nil && d.selectOpen() {
			pick = paint.PickOption(d.tree, at, paint.Controls(d.machine), paint.Fonts(d.fonts), paint.Viewport(d.viewport))
		}
		d.replay.pick = pick
	}
	return pick
}

// leftPress handles a left press outside an open list: it closes what the
// press is outside of, arms a drag and moves the focus to what takes it,
// reporting whether the press stops there.
func (d *Document) leftPress(in *input.State, at geom.Vec2, open *control.Select, pick paint.Pick) bool {
	if open != nil && !pick.Self {
		open.ClickOutside()
	}
	d.menuPressOutside(d.hovered)
	d.pressed = d.hovered
	if src := draggableIn(d.hovered); src != nil {
		d.drag = &dragState{source: src, from: at}
	}
	d.pressOpened = d.pressOpen(d.target(d.hovered))
	var takes *Node
	for _, h := range d.hovered {
		if _, ok := focusable(h); ok && h.Tag() != dom.Dialog {
			takes = h
			break
		}
	}
	if d.resetField(takes) != nil || d.suggestingFor(takes) != nil {
		return true
	}
	if len(d.modals) > 0 && (takes == nil || d.scopeOf(takes) != len(d.modals)) {
		// a press on what a modal dialog makes inert leaves the focus in it
		return true
	}
	d.moveFocus(func() bool {
		if takes == nil {
			d.focus.Blur()
			return true
		}
		return d.focus.Set(takes, false)
	})
	if t := textOf(d.machine(takes)); t != nil {
		d.caretAt(takes, t, at, inputfeed.Of(in).RawDown(int(input.KeyShift)))
	}
	if r, ok := d.machine(takes).(*control.Range); ok {
		d.fire(takes, r.Press(d.along(takes, at)))
	}
	return false
}

// leftRelease ends a press: a range drag, a press that opened something,
// else a click on what was pressed if the pointer is still over it.
func (d *Document) leftRelease(open *control.Select, pick paint.Pick) {
	if held, r := d.dragged(); r != nil {
		d.fire(held, r.Release())
	}
	if o := d.pressOpened; o != nil {
		d.pressOpened = nil
		d.releaseOpened(o, open, pick)
		d.pressed = nil
		return
	}
	if t := d.target(d.pressed); t != nil && contains(d.hovered, t) {
		d.pointerClick = t
		d.click(t)
		d.pointerClick = nil
	}
	d.pressed = nil
}

// dragStart is how far, in points, the pointer moves with the left button
// held on a Draggable element before a drag starts; less stays a click.
const dragStart = 4

// dragState is a press on a Draggable element: where it began, whether it
// became a drag, and the drop target the pointer is over.
type dragState struct {
	source *Node
	from   geom.Vec2
	on     bool
	over   *Node
}

// draggableIn is the innermost element of the chain marked draggable="true".
func draggableIn(c []*Node) *Node {
	for _, n := range c {
		if v, _ := n.Attr("draggable"); v == attrTrue {
			return n
		}
	}
	return nil
}

// dragPointer moves a press on a Draggable element: past dragStart it is a
// drag, no longer a click, its source marked data-dragging and the drop
// target under the pointer data-drag-over, the cursor a hand; let go, it
// drops on that target or, over none, is cancelled. It reports whether the
// drag took the pointer.
func (d *Document) dragPointer(c *twod.Canvas, in *input.State, at geom.Vec2) bool {
	g := d.drag
	if g == nil {
		return false
	}
	if !g.on {
		if !in.Down(input.MouseLeft) || at.Sub(g.from).Len() < dragStart {
			if !in.Down(input.MouseLeft) {
				d.drag = nil
			}
			return false
		}
		g.on = true
		g.source.SetAttr("data-dragging", "")
		d.pressed = nil
	}
	c.ShowCursor(platform.CursorHand)
	if over := dropTarget(d.hovered); over != g.over {
		if g.over != nil {
			g.over.RemoveAttr("data-drag-over")
		}
		if over != nil {
			over.SetAttr("data-drag-over", "")
		}
		g.over = over
	}
	if !in.Down(input.MouseLeft) {
		d.endDrag(g.over != nil)
	}
	return true
}

// dropTarget is the innermost element of the chain taking OnDrop.
func dropTarget(c []*Node) *Node {
	for _, n := range c {
		if handles(n, EventDrop) {
			return n
		}
	}
	return nil
}

// endDrag ends a drag, dropped on its target or cancelled: Dropped for a
// drop, then DragEnded, each once.
func (d *Document) endDrag(dropped bool) {
	g := d.drag
	d.drag = nil
	if g == nil || !g.on {
		return
	}
	g.source.RemoveAttr("data-dragging")
	if g.over != nil {
		g.over.RemoveAttr("data-drag-over")
	}
	if dropped && g.over != nil {
		d.events = append(d.events, Dropped{Target: g.over, Dragged: g.source})
	}
	d.events = append(d.events, DragEnded{Source: g.source, Dropped: dropped && g.over != nil})
}

// rightClick reports a right click: the right button pressed and released
// on the same element, not the page's html or body, which are the game's.
func (d *Document) rightClick(in *input.State) {
	n := firstOf(d.hovered)
	if n == d.html || n == d.body {
		n = nil
	}
	if in.Pressed(input.MouseRight) {
		d.rightPressed = n
	}
	if in.Released(input.MouseRight) {
		if n != nil && n == d.rightPressed {
			d.events = append(d.events, ContextClicked{Node: n})
		}
		d.rightPressed = nil
	}
}

// pressOpen opens t on the press, as macOS menu bars and Chrome's select
// do, when t is a closed menu bar title or a closed select under the
// pointer, and returns it; nil otherwise, the press then a click as before.
func (d *Document) pressOpen(t *Node) *Node {
	if t == nil {
		return nil
	}
	if m := d.submenu(t); m != nil && !isOpenMenu(m) && roleOf(menuBoxOf(t)) == roleMenubar && !menuDisabled(t) {
		d.openMenu(t, false, false)
		return t
	}
	if s, ok := d.machine(t).(*control.Select); ok && !s.Open && !s.Disabled {
		d.fire(t, s.Click())
		return t
	}
	return nil
}

// releaseOpened ends a press that opened o: released over one of its items
// or options it chooses it, back on o it leaves o open for a click, over
// anything else it closes o as it was.
func (d *Document) releaseOpened(o *Node, list *control.Select, pick paint.Pick) {
	if s, ok := d.machine(o).(*control.Select); ok {
		switch {
		case list == s && pick.List && pick.Index >= 0:
			d.fire(o, s.ClickOption(pick.Index))
		case list == s && pick.Self:
		default:
			s.ClickOutside()
			d.sync(o, s)
		}
		return
	}
	u := d.target(d.hovered)
	switch {
	case u == o:
	case u != nil && isMenuItem(u) && isOpenMenu(menuBoxOf(u)):
		d.pointerClick = u
		d.click(u)
		d.pointerClick = nil
	case u != nil && isMenuItem(u):
	default:
		d.closeAll(o, false)
	}
}

// cssCursor shows the CSS cursor of the element under the pointer, the arrow for auto.
func (d *Document) cssCursor(c *twod.Canvas) {
	n := firstOf(d.hovered)
	if n == nil {
		return
	}
	if st := dom.Data(n).Style; st != nil {
		if cur, ok := cssCursors[st.Cursor]; ok {
			c.ShowCursor(cur)
		}
	}
}

var cssCursors = map[string]platform.Cursor{ //nolint:gochecknoglobals // a read-only table
	"default": platform.CursorArrow, "pointer": platform.CursorHand,
	"text": platform.CursorText, "none": platform.CursorHidden,
	"ew-resize": platform.CursorResizeEW, "col-resize": platform.CursorResizeEW,
	"ns-resize": platform.CursorResizeNS, "row-resize": platform.CursorResizeNS,
	"grab": platform.CursorHand, "grabbing": platform.CursorHand,
}

// dragged is the range a press holds, and its machine, nil when none.
func (d *Document) dragged() (*Node, *control.Range) {
	for _, n := range d.pressed {
		if r, ok := d.machine(n).(*control.Range); ok && r.Dragging() {
			return n, r
		}
	}
	return nil, nil
}

// along is where at lies across n's content box, 0 to 1: a range's
// value under the pointer, as paint centres its thumb on the value.
func (d *Document) along(n *Node, at geom.Vec2) float64 {
	r := n.Box()
	if b := d.tree.Of[n]; b != nil {
		r.X += b.Border.Left + b.Padding.Left
		r.W -= b.Border.Horizontal() + b.Padding.Horizontal()
	}
	if r.W <= 0 {
		return 0
	}
	return float64(min(max((at.X-r.X)/r.W, 0), 1))
}

// selectOpen reports whether a select shows its list, the only time the
// pointer can be over one of its options.
func (d *Document) selectOpen() bool {
	for _, m := range d.machines {
		if s, ok := m.(*control.Select); ok && s.Open {
			return true
		}
	}
	return false
}

func contains(c []*Node, n *Node) bool {
	for _, x := range c {
		if x == n {
			return true
		}
	}
	return false
}

// drawnIn notes the World frame d draws in. A key the game read to show d,
// such as the Escape opening a pause menu, is still pressed in the frame d
// first draws, as a keydown made before a page loaded never reaches it.
func (d *Document) drawnIn(frame uint64) {
	d.fresh = !d.drawnOnce || frame > d.drawn+1
	d.drawn, d.drawnOnce = frame, true
}

func (d *Document) keys(c *twod.Canvas, in *input.State) {
	f := inputfeed.Of(in)
	if d.stillHeld(f) {
		return
	}
	d.shortcuts(c, in)
	pressed := func(k input.Key) bool { return f.RawPressed(int(k)) || f.RawRepeated(int(k)) }
	n := d.focused()
	m := d.machine(n)
	if t := textOf(m); t != nil && n != nil {
		f.CaptureKeyboard()
		d.edit(c, in, n, t)
		return
	}
	if g := d.drag; g != nil && g.on {
		f.CaptureKeys(int(input.KeyEscape))
		if pressed(input.KeyEscape) {
			d.endDrag(false)
		}
		return
	}
	var taken keySet
	defer d.keysLeft(in, n, &taken)
	if n != nil && d.focus.Active() {
		f.CaptureKeys(navKeys...)
		f.CaptureGamepadButtons(navPad...)
	}
	if n != nil && menuBoxOf(n) != nil && isMenuItem(n) {
		f.CaptureKeys(navKeys...)
		for _, r := range in.Text() {
			d.menuType(n, r)
			n = d.focused()
		}
		taken.all = true
	}
	if pressed(input.KeyTab) && n != nil {
		taken.add(input.KeyTab)
		if isMenuItem(n) && roleOf(menuBoxOf(n)) == roleMenu {
			d.closeAll(n, true)
		}
		back := f.RawDown(int(input.KeyShift))
		d.moveFocus(func() bool { return d.focus.Tab(back) })
		return
	}
	if n != nil && isGrid(n) && (f.RawDown(int(input.KeySuper)) || f.RawDown(int(input.KeyControl))) {
		return
	}
	intents, from := pressedIntents(f, pressed)
	for j, i := range intents {
		var took bool
		if n, took = d.keyIntent(f, n, m, i); took {
			taken.add(from[j])
		}
	}
}

// stillHeld reports whether a key or pad button that was down when a modal
// opened still is: the press that opened it isn't one made in it.
func (d *Document) stillHeld(f inputfeed.Feed) bool {
	if d.opening {
		d.opening = false
		d.openKeys, d.openPad = d.openKeys[:0], d.openPad[:0]
		for _, k := range intentKeys {
			if f.RawDown(int(k)) {
				d.openKeys = append(d.openKeys, k)
			}
		}
		for _, b := range intentPad {
			if f.RawGamepadDown(int(b)) {
				d.openPad = append(d.openPad, b)
			}
		}
	}
	d.openKeys = slices.DeleteFunc(d.openKeys, func(k input.Key) bool { return !f.RawDown(int(k)) })
	d.openPad = slices.DeleteFunc(d.openPad, func(b input.GamepadButton) bool { return !f.RawGamepadDown(int(b)) })
	return len(d.openKeys) > 0 || len(d.openPad) > 0
}

// pressedIntents are the intents of the keys and pad buttons pressed this
// frame, each with its key (KeyUnknown for a pad button).
func pressedIntents(f inputfeed.Feed, pressed func(input.Key) bool) ([]control.Intent, []input.Key) {
	var intents []control.Intent
	var from []input.Key
	for _, k := range intentKeys {
		if pressed(k) {
			intents, from = append(intents, control.KeyIntent(k)), append(from, k)
		}
	}
	for _, b := range intentPad {
		if f.RawGamepadPressed(int(b)) {
			intents, from = append(intents, control.PadIntent(b)), append(from, input.KeyUnknown)
		}
	}
	return intents, from
}

// keyIntent answers i for n, whose machine is m: a menu, a modal's Back,
// the control, a scroll, Back, else a focus move. It returns the element the
// next intent goes to and whether i was taken.
func (d *Document) keyIntent(f inputfeed.Feed, n *Node, m any, i control.Intent) (*Node, bool) {
	if n != nil && d.menuIntent(n, i) {
		return d.focused(), true
	}
	if i == control.Back && d.backFromModal() {
		return n, true
	}
	if n == nil {
		if i == control.Back && hasBack(d.body) {
			d.events = append(d.events, bodyBack{})
			return n, true
		}
		if d.scrollKey(d.pointed(), i) {
			f.CaptureKeys(int(pageKey(i)))
			return n, true
		}
		return n, false
	}
	if d.intent(n, m, i) {
		return n, true
	}
	if d.scrollKey(n, i) {
		f.CaptureKeys(int(pageKey(i)))
		return n, true
	}
	if i == control.Back {
		d.events = append(d.events, BackPressed{Node: n})
		return n, true
	}
	if dir := focusDir(i); dir != (geom.Vec2{}) {
		d.moveFocus(func() bool { return d.focus.Move(dir) })
		return n, true
	}
	return n, false
}

// keySet is the keys a frame took, all of them when a menu takes the typing.
type keySet struct {
	bits [int(input.KeyMenu)/64 + 1]uint64
	all  bool
}

func (s *keySet) add(k input.Key) {
	if k >= 0 && k <= input.KeyMenu {
		s.bits[k/64] |= 1 << (k % 64)
	}
}

func (s *keySet) has(k input.Key) bool { return s.all || s.bits[k/64]&(1<<(k%64)) != 0 }

// keysLeft reports the keys pressed this frame that nothing took as KeyPressed: on the
// focused element n, else the open modal dialog; with neither, they are the game's.
func (d *Document) keysLeft(in *input.State, n *Node, taken *keySet) {
	f := inputfeed.Of(in)
	if taken.all || !f.Changed() {
		return
	}
	if n == nil && len(d.modals) > 0 {
		n = d.modals[len(d.modals)-1]
	}
	if n == nil {
		return
	}
	command := input.KeyControl
	if d.platform == textedit.Mac {
		command = input.KeySuper
	}
	text := string(in.Text())
	for k := input.Key(0); k <= input.KeyMenu; k++ {
		if taken.has(k) || modifier(k) || !f.RawPressed(int(k)) {
			continue
		}
		d.events = append(d.events, KeyPressed{Node: n, Key: k, Text: text, Command: f.RawDown(int(command))})
		text = ""
	}
}

func modifier(k input.Key) bool { return k >= input.KeyLeftShift && k <= input.KeyRightSuper }

// currentPlatform is the platform whose key conventions the Document follows.
var currentPlatform = textedit.Current //nolint:gochecknoglobals // tests swap it to check other platforms' keys

// intent hands i to n's machine, reporting whether it took it.
func (d *Document) intent(n *Node, m any, i control.Intent) bool {
	type doer interface {
		Handles(control.Intent) bool
		Do(control.Intent) control.Events
	}
	if n.Tag() == dom.Summary && i == control.Activate {
		d.click(n)
		return true
	}
	if isGrid(n) {
		return d.gridIntent(n, i)
	}
	if isListbox(n) {
		return d.listboxIntent(n, i)
	}
	if isTab(n) {
		if i == control.Activate {
			d.click(n)
			return true
		}
		return d.tabArrow(n, i)
	}
	if d.isRadio(n) {
		if i == control.Activate {
			d.click(n)
			return true
		}
		return d.radioArrow(n, i)
	}
	if m == nil && i == control.Activate && has(n, "tabindex") {
		d.fire(n, control.Clicked)
		return true
	}
	if x, ok := m.(doer); ok && x.Handles(i) {
		e := x.Do(i)
		d.fire(n, e)
		if _, button := m.(*control.Button); button && e.Has(control.Clicked) {
			d.dialogClick(n)
		}
		return true
	}
	return false
}

// edit hands a focused text field the typed text and the editing keys.
func (d *Document) edit(c *twod.Canvas, in *input.State, n *Node, t *control.Text) {
	f := inputfeed.Of(in)
	m := textedit.Mods{
		Shift:   f.RawDown(int(input.KeyShift)),
		Control: f.RawDown(int(input.KeyControl)),
		Alt:     f.RawDown(int(input.KeyAlt)),
		Super:   f.RawDown(int(input.KeySuper)),
	}
	if !m.Super && !m.Control {
		for _, r := range in.Text() {
			if num, ok := d.machine(n).(*control.Number); ok {
				d.fire(n, num.Type(string(r)))
				continue
			}
			d.fire(n, t.Type(string(r)))
			if r == ':' && isShortcodeField(n) {
				d.expandShortcode(n, t)
			}
		}
	}
	if isShortcodeField(n) {
		defer d.suggest(n, t)
		if sg := d.suggests[n]; sg != nil && len(sg.matches) > 0 {
			f.CaptureGamepadButtons(navPad...)
		}
		if d.suggestionKeys(in, n, t) {
			return
		}
	}
	var clip textedit.Clipboard
	if w := c.Window(); w != nil {
		clip = w
	}
	if f.RawPressed(int(input.KeyEscape)) {
		if t.Handles(control.Back) {
			d.fire(n, t.Do(control.Back))
		} else if !d.backFromModal() {
			d.moveFocus(func() bool { d.focus.Blur(); return true })
		}
		return
	}
	for _, k := range textedit.Keys {
		if !f.RawPressed(int(k)) && !f.RawRepeated(int(k)) {
			continue
		}
		if k == input.KeyTab {
			d.moveFocus(func() bool { return d.focus.Tab(m.Shift) })
			return
		}
		if num, ok := d.machine(n).(*control.Number); ok && (k == input.KeyUp || k == input.KeyDown) {
			d.stepNumber(n, num, control.KeyIntent(k), m.Shift)
			continue
		}
		a, extend := textedit.Resolve(d.platform, k, m)
		if a == textedit.None {
			continue
		}
		if a == textedit.Submit || a == textedit.Newline {
			settle(d.machine(n))
		}
		d.fire(n, t.Key(a, extend, clip))
	}
}

// states sets each element's hover, active and focus pseudo-classes.
//
// Only the elements that held a state last frame or hold one now can
// change, so it visits those rather than the tree.
func (d *Document) states() {
	focused := d.focused()
	set := func(n *Node) {
		held := d.pinned[n]
		n.SetState(dom.Hover, contains(d.hovered, n) || held&dom.Hover != 0)
		n.SetState(dom.Active, contains(d.pressed, n) || held&dom.Active != 0)
		n.SetState(dom.Focus, n == focused || held&dom.Focus != 0)
		n.SetState(dom.FocusVisible, n == focused && d.focus.Active() || held&dom.FocusVisible != 0)
	}
	for _, n := range d.stated {
		set(n)
	}
	d.stated = d.stated[:0]
	add := func(n *Node) {
		if n != nil && !n.IsText() {
			d.stated = append(d.stated, n)
			set(n)
		}
	}
	for _, n := range d.hovered {
		add(n)
	}
	for _, n := range d.pressed {
		add(n)
	}
	add(focused)
	for n := range d.pinned {
		add(n)
	}
	d.pin()
}

// pin sets the pseudo-classes a test page pinned on its nodes.
func (d *Document) pin() {
	for n, s := range d.pinned {
		n.SetState(s, true)
	}
}

// scrollPointer scrolls by the pointer, before anything else takes it: a
// held thumb drags its container, a press on a bar takes the pointer for
// its container, and the wheel (Shift+wheel sideways) turns the innermost
// container under the pointer that can still scroll that way, else the
// next one out. It reports whether the bars took the pointer.
func (d *Document) scrollPointer(c *twod.Canvas, in *input.State, hit *box.Box) bool {
	if d.tree == nil {
		return false
	}
	at := d.mouse(c)
	if n := d.dragging; n != nil {
		b := d.tree.Of[n]
		s := d.scrolling(n)
		if b != nil {
			off, _ := s.Input(paint.BarsOf(b, origin(b)), paint.Pointer{At: at, Down: in.Down(input.MouseLeft)}, d.lineStep(b))
			d.setScroll(b, off)
		}
		if b == nil || !s.Dragging() {
			d.dragging = nil
		}
		return true
	}
	held := scrollContainers(hit)
	if in.Pressed(input.MouseLeft) {
		for _, b := range held {
			bars := paint.BarsOf(b, origin(b))
			if !bars.Over(at) {
				continue
			}
			s := d.scrolling(b.Node)
			off, _ := s.Input(bars, paint.Pointer{At: at, Pressed: true, Down: true}, d.lineStep(b))
			d.setScroll(b, off)
			if s.Dragging() {
				d.dragging = b.Node
			}
			return true
		}
	}
	w := in.Wheel()
	if inputfeed.Of(in).RawDown(int(input.KeyShift)) && w.X == 0 {
		w = geom.V2(w.Y, 0)
	}
	if w == (geom.Vec2{}) {
		return false
	}
	for _, b := range held {
		bars := paint.BarsOf(b, origin(b))
		if !canScroll(bars, w) {
			continue
		}
		off, _ := d.scrolling(b.Node).Input(bars, paint.Pointer{At: at, Hover: true, Wheel: w}, d.lineStep(b))
		d.setScroll(b, off)
		break
	}
	return false
}

// scrollKey scrolls the innermost container holding n, n included, that
// can still move that way: by PageUp and PageDown a view at a time, to
// its start or end by Home and End. n is the focused element, whose own
// control left the key, or with nothing focused what is under the
// pointer, as a browser pages what the pointer is over. It reports
// whether it scrolled.
func (d *Document) scrollKey(n *Node, i control.Intent) bool {
	if d.tree == nil || n == nil {
		return false
	}
	switch i {
	case control.PageUp, control.PageDown, control.Home, control.End:
	case control.None, control.Activate, control.Back, control.Up, control.Down, control.Left, control.Right:
		return false
	}
	b := d.tree.Of[n]
	for _, s := range scrollContainers(b) {
		bars := paint.BarsOf(s, origin(s))
		end := max(bars.Content.Y-bars.View.H, 0)
		off := bars.Scroll
		switch i {
		case control.PageUp:
			off.Y -= bars.View.H
		case control.PageDown:
			off.Y += bars.View.H
		case control.Home:
			off.Y = 0
		case control.End:
			off.Y = end
		case control.None, control.Activate, control.Back, control.Up, control.Down, control.Left, control.Right:
			// scrollKey lets only the four above through
		}
		off.Y = min(max(off.Y, 0), end)
		if off == bars.Scroll {
			continue
		}
		d.setScroll(s, off)
		return true
	}
	return false
}

// pageKey is the key that pages a scroll container for i, taken from the
// game only when it scrolled something.
func pageKey(i control.Intent) input.Key {
	switch i {
	case control.PageUp:
		return input.KeyPageUp
	case control.PageDown:
		return input.KeyPageDown
	case control.Home:
		return input.KeyHome
	case control.End:
		return input.KeyEnd
	case control.None, control.Activate, control.Back, control.Up, control.Down, control.Left, control.Right:
		// scrollKey pages only by the four above
	}
	return 0
}

// focusDir is the direction i moves the focus in, none for the intents that
// don't.
func focusDir(i control.Intent) geom.Vec2 {
	switch i {
	case control.Up:
		return geom.Vec2{Y: -1}
	case control.Down:
		return geom.Vec2{Y: 1}
	case control.Left:
		return geom.Vec2{X: -1}
	case control.Right:
		return geom.Vec2{X: 1}
	case control.None, control.Activate, control.Back, control.Home, control.End, control.PageUp, control.PageDown:
		// they don't move the focus
	}
	return geom.Vec2{}
}

// pointed is the innermost element under the pointer, the root when
// there is none.
func (d *Document) pointed() *Node {
	if len(d.hovered) > 0 {
		return d.hovered[0]
	}
	return d.html
}

// reveal scrolls each container holding n, innermost first, by the least
// that brings n into its view: its nearest edge to the view's, or its
// start when it is bigger than the view.
func (d *Document) reveal(n *Node) {
	if d.tree == nil {
		return
	}
	b := d.tree.Of[n]
	if b == nil {
		return
	}
	for _, s := range scrollContainers(b.Parent) {
		bars := paint.BarsOf(s, origin(s))
		r, v := b.Rect(), bars.View
		off := bars.Scroll
		off.X += nearest(r.X, r.W, v.X, v.W)
		off.Y += nearest(r.Y, r.H, v.Y, v.H)
		off.X = min(max(off.X, 0), max(bars.Content.X-v.W, 0))
		off.Y = min(max(off.Y, 0), max(bars.Content.Y-v.H, 0))
		d.setScroll(s, off)
	}
}

// nearest is how far a view from v of length vl moves for a thing from
// a of length al to show: its start or end to the view's nearest edge, 0
// when it already shows.
func nearest(a, al, v, vl float32) float32 {
	switch {
	case a < v || al > vl:
		return a - v
	case a+al > v+vl:
		return a + al - (v + vl)
	}
	return 0
}

// scrollContainers are the scroll containers holding b, b included,
// innermost first.
func scrollContainers(b *box.Box) []*box.Box {
	var out []*box.Box
	for ; b != nil; b = b.Parent {
		if b.Kind != box.Element || b.Node == nil || b.Style == nil {
			continue
		}
		o := b.Style.Overflow
		if scrolls(o.Width) || scrolls(o.Height) {
			out = append(out, b)
		}
	}
	return out
}

func scrolls(o style.Overflow) bool { return o == style.Scroll || o == style.OverflowAuto }

// canScroll reports whether a container with bars still has room to
// scroll the way w turns.
func canScroll(bars paint.Bars, w geom.Vec2) bool {
	const eps = 0.01
	room := geom.V2(max(bars.Content.X-bars.View.W, 0), max(bars.Content.Y-bars.View.H, 0))
	return w.Y > 0 && bars.Scroll.Y < room.Y-eps || w.Y < 0 && bars.Scroll.Y > eps ||
		w.X > 0 && bars.Scroll.X < room.X-eps || w.X < 0 && bars.Scroll.X > eps
}

func origin(b *box.Box) geom.Vec2 { r := b.Rect(); return geom.V2(r.X, r.Y) }

// lineStep is a wheel line in b: its font's line height.
func (d *Document) lineStep(b *box.Box) float32 {
	m := d.fonts.Metrics(b.Style.Font)
	return m.Ascent + m.Descent + m.LineGap
}

func (d *Document) scrolling(n *Node) *paint.Scrolling {
	if d.scrolls == nil {
		d.scrolls = map[*Node]*paint.Scrolling{}
	}
	s := d.scrolls[n]
	if s == nil {
		s = &paint.Scrolling{}
		d.scrolls[n] = s
	}
	return s
}

// setScroll scrolls b's node to off: kept on the node for later layouts,
// which clamp it, and on the box for this frame's paint.
func (d *Document) setScroll(b *box.Box, off geom.Vec2) {
	dom.Data(b.Node).Scroll = off
	b.Scroll = off
	d.scrolled = true
}

// labelled is the control label l names: the one its for attribute gives,
// else the first one inside it, as HTML's implicit label.
func (d *Document) labelled(l *Node) *Node {
	if id, ok := l.Attr("for"); ok {
		return d.byID(id)
	}
	var found *Node
	l.Walk(func(n *Node) bool {
		if found != nil {
			return false
		}
		if slices.Contains([]dom.Tag{dom.Input, dom.Select, dom.Textarea, dom.Button}, n.Tag()) {
			found = n
			return false
		}
		return true
	})
	return found
}

// clicking is how many clicks in a row n's Clicked counts: 2 or 3 for a
// pointer's second or third within 400ms, else 1.
func (d *Document) clicking(n *Node) int {
	if d.pointerClick != n {
		return 1
	}
	if d.lastClick == n && d.clock-d.lastClickAt <= 0.4 && d.clicks < 3 {
		d.clicks++
	} else {
		d.clicks = 1
	}
	d.lastClick, d.lastClickAt = n, d.clock
	return d.clicks
}

// crossed emits PointerLeft for the elements was held and now doesn't,
// innermost first, then PointerEntered for those now holds and was
// didn't, outermost first, as mouseleave and mouseenter.
func (d *Document) crossed(was, now []*Node) {
	for _, n := range was {
		if n != nil && !contains(now, n) {
			d.events = append(d.events, PointerLeft{n})
		}
	}
	for i := len(now) - 1; i >= 0; i-- {
		if n := now[i]; n != nil && !contains(was, n) {
			d.events = append(d.events, PointerEntered{n})
		}
	}
}
