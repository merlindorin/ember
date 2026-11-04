package ui

import (
	"fmt"
	"io/fs"
	"slices"
	"strings"

	"github.com/merlindorin/ember/automation"
	"github.com/merlindorin/ember/geom"
	"github.com/merlindorin/ember/twod"
	"github.com/merlindorin/ember/ui/control"
	"github.com/merlindorin/ember/ui/css"
	"github.com/merlindorin/ember/ui/internal/assets"
	"github.com/merlindorin/ember/ui/internal/box"
	"github.com/merlindorin/ember/ui/internal/dom"
	"github.com/merlindorin/ember/ui/internal/fonts"
	"github.com/merlindorin/ember/ui/internal/style"
	"github.com/merlindorin/ember/ui/layout"
	"github.com/merlindorin/ember/ui/paint"
)

// Document is one window's tree, from its html element down. Added to a
// window, it lays out at the window's size and draws over the 2D world
// each frame.
type Document struct {
	twod.Item
	// zoom draws the Document that many times its size (Zoom); 0 is 1.
	zoom float32
	// drawn is the World frame d was last drawn in, and fresh whether this
	// frame is the first after frames it wasn't: keys pressed then were
	// pressed before it showed, so they are the game's (drawnIn).
	drawn uint64
	// typing is the text field the player has typed in since it last
	// changed or was submitted: SetValue leaves it be while it has the
	// focus, and sets it again once its text is committed.
	typing     *Node
	drawnOnce  bool
	fresh      bool
	html, body *Node
	tree       *box.Tree
	fonts      *fonts.Set
	faces      map[*fonts.Face]*twod.Face
	sheets     []*css.Sheet
	warns      []error
	warned     map[string]bool
	// files are the stylesheets' file systems, where their images are;
	// images the textures decoded from them.
	files  []fs.FS
	images map[string]*twod.Texture
	trans  *css.Transitions
	emoji  *twod.EmojiSet
	// clock is the World's real time at the last Draw, -1 before it.
	clock float32
	// viewport is the canvas: the window, or the host's area.
	viewport geom.Rect
	// laidAt and laidScale are the viewport and font scale tree was laid
	// out at.
	laidAt    geom.Vec2
	laidScale float32
	// kept carries layout's nodes between layouts, so a change lays out only its path.
	kept layout.Retained
	interaction
	replay replay
	// placed counts the nodes Layout pointed at their boxes, for tests.
	placed int
	// translator shows the text, said remembers what it said, and relayout
	// lays everything out again after it changed.
	translator Translator
	said       map[string]string
	relayout   bool
	// tips are the tooltips the last layout showed; placedTips those
	// placeTips has looked at since they showed.
	tips       []*Node
	placedTips map[*Node]bool
	// followers are the nodes following an entity that the last layout
	// placed, looked for again once followings moved past followingsSeen.
	followers      []*Node
	followingsSeen uint64
}

// DocumentOption configures a Document.
type DocumentOption func(*Document)

func NewDocument(opts ...DocumentOption) *Document {
	d := &Document{
		Item:  twod.Base(),
		html:  dom.NewElement(dom.HTML),
		body:  dom.NewElement(dom.Body),
		fonts: fonts.Builtin(),
		trans: css.NewTransitions(),
		clock: -1,
	}
	d.html.Append(d.body)
	twod.Build(d, "document", twod.Overlay())
	for _, o := range opts {
		o(d)
	}
	return d
}

// Stylesheet adds the CSS file name in fsys, after those added before.
// Its @font-face files are read from fsys too. What it gets wrong is in
// Warnings.
func Stylesheet(fsys fs.FS, name string) DocumentOption {
	return func(d *Document) {
		s, warns, err := css.Load(fsys, name)
		if err != nil {
			d.warns = append(d.warns, err)
			return
		}
		for _, w := range warns {
			d.warns = append(d.warns, w)
		}
		for _, f := range s.FontFaces() {
			ttf, readErr := fs.ReadFile(fsys, f.Src)
			if readErr == nil {
				readErr = d.fonts.Add(f.Family, f.Weight, f.Italic, ttf)
			}
			if readErr != nil {
				d.warns = append(d.warns, readErr)
			}
		}
		d.sheets = append(d.sheets, s)
		d.files = append(d.files, fsys)
	}
}

// Images adds fsys to where url() and an img's src look for pictures, as a
// stylesheet's own files are: the game's art passed in, read by name, the
// last added first. It adds no rules.
func Images(fsys fs.FS) DocumentOption {
	return func(d *Document) { d.files = append(d.files, fsys) }
}

// CSS adds a stylesheet written in the code, as a <style> element does,
// after the sheets before it. Its @font-face rules have no files to read,
// so they are warnings: fonts come with Stylesheet.
func CSS(text string) DocumentOption {
	return func(d *Document) {
		s, warns := css.Parse("style", text)
		for _, w := range warns {
			d.warns = append(d.warns, w)
		}
		for _, f := range s.FontFaces() {
			d.warns = append(d.warns,
				fmt.Errorf("css: @font-face %q in CSS has no file to read; load it with Stylesheet", f.Family))
		}
		d.sheets = append(d.sheets, s)
	}
}

// ByID is the element whose id is id, as getElementById: the first in tree
// order, or nil.
func (d *Document) ByID(id string) *Node {
	var found *Node
	d.html.Walk(func(n *Node) bool {
		if found == nil && n.ID() == id {
			found = n
		}
		return found == nil
	})
	return found
}

// Sheets are the document's stylesheets in order, after css.Default.
func (d *Document) Sheets() []*css.Sheet { return d.sheets }

// Warnings is what the stylesheets and style attributes got wrong, and
// files that could not be read.
func (d *Document) Warnings() []error { return d.warns }

// HTML is the root element.
func (d *Document) HTML() *Node { return d.html }

// Body is where the game's elements go.
func (d *Document) Body() *Node { return d.body }

// Layout styles and lays the tree out in a viewport of size points, as
// each frame does before drawing, and updates every Node.Box. When nothing
// changed that layout reads, the boxes are kept and only scroll offsets
// and sticky boxes are redone. Controls
// show their state before styling, so their pseudo-classes restyle them
// the same frame.
func (d *Document) Layout(size geom.Vec2) {
	d.style()
	if d.relayout {
		d.kept, d.relayout = layout.Retained{}, false
	} else if d.tree != nil && size == d.laidAt && d.fonts.Scale() == d.laidScale &&
		dom.Data(d.html).Dirty&(dom.DirtyLayout|dom.DirtyDescendant) == 0 {
		moved := layout.Refresh(d.tree, d.html) || d.scrolled
		d.scrolled = false
		if !moved {
			return
		}
		d.replay.repaint = true
		return
	}
	d.replay.repaint = true
	d.scrolled = false
	d.lay(size)
	if tips, lists := d.placeTips(), d.placeSuggestions(); tips || lists {
		d.style()
		d.lay(size)
	}
}

// lay lays the tree out at size, points each node at its box, a node the
// layout left out (under display: none) at none, and notes the tooltips
// showing. A node moved to another Document is that one's to place.
func (d *Document) lay(size geom.Vec2) {
	d.tree = layout.Layout(d.html, size, d.fonts, layout.Images(d), layout.Retain(&d.kept), layout.Translate(d.tr()))
	for _, n := range d.tree.Dropped {
		if d.inDocument(n) {
			dom.Data(n).Placed = nil
		}
		d.tips, d.followers = dropNode(d.tips, n), dropNode(d.followers, n)
	}
	d.laidAt, d.laidScale = size, d.fonts.Scale()
	for _, n := range d.tree.Written {
		b := d.tree.Of[n]
		if b == nil {
			continue
		}
		dom.Data(n).Placed = b
		if tip := b.Parent != nil && b.Kind == box.Element && isTip(n); tip != slices.Contains(d.tips, n) {
			if tip {
				d.tips = append(d.tips, n)
			} else {
				d.tips = dropNode(d.tips, n)
			}
		}
		if dom.Data(n).Follow != nil && !slices.Contains(d.followers, n) {
			d.followers = append(d.followers, n)
		}
	}
	d.findFollowers()
	d.placed += len(d.tree.Of)
}

// dropNode is s less n.
func dropNode(s []*Node, n *Node) []*Node {
	if i := slices.Index(s, n); i >= 0 {
		return slices.Delete(s, i, i+1)
	}
	return s
}

// style shows the controls' state, then computes the styles of what
// changed, a Shadow text's from its element's.
func (d *Document) style() {
	d.pin()
	if dom.Data(d.html).Dirty == 0 {
		// Nothing was added or changed: only the controls' own state can
		// have moved, so sync those and leave the rest of the tree.
		d.syncControls()
	} else {
		d.controls = d.controls[:0]
		d.html.Walk(func(n *Node) bool {
			if dom.Data(n).Dirty&dom.DirtyTree != 0 {
				d.restructure = true
			}
			if m := d.machine(n); m != nil {
				d.controls = append(d.controls, n)
				d.sync(n, m)
			}
			return !n.IsText()
		})
	}
	if dom.Data(d.html).Dirty == 0 {
		return
	}
	d.replay.repaint = true
	for _, w := range d.trans.Compute(d.html, d.sheets) {
		d.warn(w)
	}
	for _, n := range d.controls {
		if sh := dom.Data(n).Shadow; sh != nil {
			dom.Data(sh).Style = dom.Data(n).Style
		}
	}
}

// syncControls syncs the controls the last walk of the tree found in it.
func (d *Document) syncControls() {
	for _, n := range d.controls {
		d.sync(n, d.machines[n])
	}
}

// docOf is the Document the internal packages hand back through an any:
// they only ever hold a *Document there.
func docOf(d any) *Document {
	doc, ok := d.(*Document)
	if !ok {
		panic(fmt.Sprintf("ui: internal caller passed %T, not *Document", d))
	}
	return doc
}

//nolint:gochecknoinits // the internal packages reach the Document through these hooks, set once
func init() {
	assets.Of = func(d any, url string) ([]byte, bool) { return docOf(d).imageBytes(url) }
	fonts.Of = func(d any) *fonts.Set { return docOf(d).fonts }
	assets.Styled = func(d any) { docOf(d).style() }
	assets.Modal = func(doc any, n *dom.Node) (uint64, bool) {
		m, ok := docOf(doc).machine(n).(*control.Dialog)
		return m.Layer, ok && m.Open && m.Modal
	}
	assets.Tree = func(d any) *box.Tree { return docOf(d).tree }
	assets.Viewport = func(d any) geom.Vec2 { return docOf(d).laidAt }
	assets.Path = func(doc any, n *dom.Node) string {
		leaf := docOf(doc).leaf(n)
		if leaf == "" {
			return ""
		}
		return strings.Join(append(scope(n), automation.Slug(leaf)), "/")
	}
	assets.Pin = func(doc any, n *dom.Node, s dom.State) {
		d := docOf(doc)
		if d.pinned == nil {
			d.pinned = map[*Node]dom.State{}
		}
		d.pinned[n] |= s
	}
}

// Draw advances the transitions by the real time since the last frame,
// so they run in a paused game too, then lays the document out and paints
// it: at the window's size, or at c.Size() when an element hosting the
// document draws it in its own space.
func (d *Document) Draw(c *twod.Canvas) {
	if win := c.Window(); win != nil {
		d.fonts.SetScale(win.ContentScale() * d.Zoom())
	}
	animating := d.tick(c)
	size := d.drawSize(c)
	d.viewport = geom.Rect{W: size.X, H: size.Y}
	d.Layout(size)
	if d.follow(c) {
		d.Layout(size)
	}
	d.replay.quiet = d.quietNow(c, animating)
	if d.interact(c) {
		d.Layout(size)
	}
	d.followCaret(size)
	d.record(c)
	why := d.repaintReason(c, size, animating)
	d.replay.animating = animating
	live := &d.replay.live
	invalidated := d.replay.invalidated
	d.replay.invalidated = false
	switch {
	case why != "":
	case live.Any() && d.replay.inPlace && live.Timed():
		why = repaintLive
	case live.Any() && d.replay.inPlace && invalidated:
		why = repaintCanvas
	case c.Replay(&d.replay.last):
		if live.Any() && !d.replay.inPlace {
			d.zoomed(c, func() { paint.PaintLive(c, live, d.clock, d.options()...) })
			count(c, replayedLive)
		} else {
			count(c, replayed)
		}
		d.replay.replayedLast = true
		return
	default:
		why = repaintRefused
	}
	count(c, why)
	m := c.Mark()
	apart := !d.replay.inPlace
	d.replay.frame = paint.Frame{Time: d.clock, Live: live, Apart: apart, Keep: d.tree.Of[d.focused()]}
	d.zoomed(c, func() { paint.Paint(c, d.tree, d.frameOptions()...) })
	d.replay.recorded = c.Keep(m, &d.replay.last)
	if apart && live.Any() {
		d.zoomed(c, func() { paint.PaintLive(c, live, d.clock, d.options()...) })
		if live.Covered() {
			// Kept apart, they are in no recording: drawn on top this once,
			// the next frame paints them in their place.
			d.replay.recorded = false
		}
	}
	d.replay.inPlace = live.Covered()
	d.replay.replayedLast = false
	d.replay.repaint, d.replay.size, d.replay.scale = false, size, d.fonts.Scale()
}

// tick moves the document's clock to the World's real time and steps its
// transitions, reporting whether one is still running.
func (d *Document) tick(c *twod.Canvas) bool {
	animating := false
	if w := c.World(); w != nil {
		now := w.Time().RealElapsed
		if d.clock >= 0 {
			animating = d.trans.Step(now - d.clock)
		}
		d.clock = now
		d.drawnIn(w.Time().Frame)
	}
	return animating
}

// drawSize is the size the document lays out at, in CSS pixels: c's own, or
// the window's when c has none.
func (d *Document) drawSize(c *twod.Canvas) geom.Vec2 {
	size := c.Size()
	if size.X <= 0 || size.Y <= 0 {
		s := c.Screen()
		size = geom.V2(s.W, s.H)
	}
	return size.Scale(1 / d.Zoom())
}

func (d *Document) face(f style.Font) *twod.Face {
	ff := d.fonts.Face(f)
	if tf, ok := d.faces[ff]; ok {
		return tf
	}
	var opts []twod.FaceOption
	if fb := d.fonts.Fallback(); fb != nil && fb != ff {
		opts = append(opts, twod.Fallback(d.face(style.Font{Family: []string{fb.Family}, Weight: fb.Weight})))
	}
	if d.emoji != nil {
		opts = append(opts, twod.Emoji(d.emoji))
	}
	tf, err := twod.NewFace(ff.TTF, opts...)
	if err != nil {
		tf = twod.Regular()
	}
	if d.faces == nil {
		d.faces = map[*fonts.Face]*twod.Face{}
	}
	d.faces[ff] = tf
	return tf
}

// warn keeps a style attribute's warning once, however often its element
// restyles.
func (d *Document) warn(err error) {
	if d.warned == nil {
		d.warned = map[string]bool{}
	}
	if !d.warned[err.Error()] {
		d.warned[err.Error()] = true
		d.warns = append(d.warns, err)
	}
}
