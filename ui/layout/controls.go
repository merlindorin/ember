package layout

import (
	"math"
	"strconv"
	"strings"

	"github.com/lantervale/ember/ui/internal/box"
	"github.com/lantervale/ember/ui/internal/dom"
	"github.com/lantervale/ember/ui/internal/style"
)

type controlKind uint8

const (
	noControl controlKind = iota
	inputControl
	selectControl
	textareaControl
)

// maxCharFactor is Chrome's max-char width over the average, as it measures the Go faces;
// an input's width is ceil(size × average) + (max - average).
const maxCharFactor = 1.875

// emptyLine holds the line after a textarea's last line break, which Chrome counts as
// one: its caret stands there, and the textarea scrolls to it.
const emptyLine = "\u00a0"

// textareaCaret is the point Chrome adds beside a scrolling textarea's scrollbar gutter.
const textareaCaret = 1

func controlOf(e *dom.Node) controlKind {
	if dom.Data(e).Shadow == nil {
		return noControl
	}
	switch e.Tag() {
	case dom.Input:
		return inputControl
	case dom.Select:
		return selectControl
	case dom.Textarea:
		return textareaControl
	// no control
	case dom.TextTag, dom.HTML, dom.Body, dom.Div, dom.Section, dom.Header, dom.Footer, dom.Nav, dom.Main, dom.Aside,
		dom.Article, dom.P, dom.H1, dom.H2, dom.H3, dom.H4, dom.Span, dom.Strong, dom.Em, dom.Code, dom.Label, dom.Button,
		dom.Option, dom.Details, dom.Summary, dom.Dialog, dom.Table, dom.Thead, dom.Tbody, dom.Tr, dom.Th, dom.Td,
		dom.Progress, dom.Meter, dom.Img, dom.Hr, dom.Canvas, dom.Kbd, dom.Samp, dom.Pre, dom.Caption, dom.Tfoot:
	}
	return noControl
}

// buildControl gives an input, select or textarea its shadow text as its only inline
// content: on one unwrapped line, except a textarea's, and never a select's options.
func (t *tree) buildControl(n *node, kind controlKind) {
	shadow := dom.Data(n.box.Node).Shadow
	st := *styleOf(shadow)
	if kind != textareaControl {
		st.WhiteSpace = style.Pre
	}
	text := Translated(shadow, t.translate)
	if kind == textareaControl && strings.HasSuffix(text, "\n") {
		text += emptyLine
	}
	b := &box.Box{Kind: box.TextRun, Node: shadow, Style: styleOf(shadow)}
	c := &inlineContent{owner: n, runs: []*textRun{{box: b, parent: n.box, st: &st, text: text}},
		unit: unitsPerPoint(t.fonts)}
	c.cut(t.fonts)
	n.inline = c
	n.control = kind
}

// caretWidth is the point a field's caret takes after its text.
const caretWidth = 1

// inputText is how wide an input's text runs as shown (a password's bullets), its caret
// after it: what it scrolls along, as Chrome's inner editor does, though its text never
// widens it. A placeholder scrolls nothing.
func (t *tree) inputText(n *node) float32 {
	d := dom.Data(n.box.Node)
	if d.Placeholder {
		return 0
	}
	return t.fonts.Advance(styleOf(d.Shadow).Font, Translated(d.Shadow, t.translate)) + caretWidth
}

// centresContent reports a button, input or select, whose content Chrome centres
// vertically in its content box.
func centresContent(n *node) bool {
	if n.box.Node == nil {
		return false
	}
	tag := n.box.Node.Tag()
	return tag == dom.Button || tag == dom.Input || tag == dom.Select || tag == dom.Td || tag == dom.Th
}

// textTop is where the owner's first line box starts: the top of its content box, or the
// lines centred in it for a button, input or select.
func (t *tree) textTop(n *node, ls []line, top, contentHeight float32) float32 {
	if !centresContent(n) {
		return top
	}
	h := float32(0)
	for _, l := range ls {
		lh, _ := t.lineBox(n.inline, l)
		h += lh
	}
	return top + max(contentHeight-h, 0)/2
}

func intAttr(e *dom.Node, name string, fallback int) int {
	if v, ok := e.Attr(name); ok {
		if i, err := strconv.Atoi(v); err == nil && i > 0 {
			return i
		}
	}
	return fallback
}

// measureControl is a control's content size as Chrome sizes it: its text never widens
// it. An input is size (20) average characters, a textarea cols (20) by rows (2) with
// room for its scrollbar, unless its gutter already holds it, a select its widest option,
// its placeholder counted while it shows it, as Chrome counts the hidden option that holds it.
func (t *tree) measureControl(n *node, known size) (size, float32, float32) {
	st := n.st
	e := n.box.Node
	lh, base := t.lineHeight(n.inline, st)
	avg := t.fonts.Advance(st.Font, "x")
	rows := 1
	var w float32
	switch n.control {
	case inputControl:
		w = float32(math.Ceil(float64(avg*float32(intAttr(e, "size", 20))))) + (maxCharFactor-1)*avg
	case textareaControl:
		w = avg * float32(intAttr(e, "cols", 20))
		if st.Overflow.Height != style.Hidden && st.Overflow.Height != style.Clip {
			w += textareaCaret
			if n.ownGutter().x == 0 {
				w += st.ScrollbarWidth
			}
		}
		rows = intAttr(e, "rows", 2)
	case selectControl:
		for _, o := range e.Children() {
			if o.Tag() == dom.Option {
				w = max(w, t.fonts.Advance(st.Font, o.Text()))
			}
		}
		if dom.Data(e).Placeholder && len(n.inline.runs) > 0 {
			w = max(w, t.fonts.Advance(st.Font, n.inline.runs[0].text))
		}
		w = float32(math.Ceil(float64(w)))
	case noControl:
	}
	s := size{or(known.w, w), or(known.h, float32(rows)*lh)}
	return s, base, base + float32(rows-1)*lh
}

// textareaText is a textarea's text wrapped in its used width, as its scrollable content
// (never less than its content box, which Chrome's inner editor fills), and its gutter: once the text overflows the
// block axis under overflow auto, Chrome
// reserves the scrollbar's room and wraps the text again beside it.
func (t *tree) textareaText(n *node, s size, pb rect, gutter point) (size, point) {
	wrap := func(g point) size {
		box := size{fmax(s.w-pb.horizontal()-g.x, 0), fmax(s.h-pb.vertical()-g.y, 0)}
		m, _, _ := t.measure(n.inline, size{none, none}, spaces{w: def(box.w), h: maxContentSpace}, size{box.w, none})
		return m.mMax(box)
	}
	m := wrap(gutter)
	if n.st.Overflow.Height == style.OverflowAuto && gutter.x == 0 && m.h > s.h-pb.vertical()-gutter.y {
		gutter.x = n.st.ScrollbarWidth
		m = wrap(gutter)
	}
	return m, gutter
}
