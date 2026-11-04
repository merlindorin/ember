package layout

import (
	"math"
	"slices"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/merlindorin/ember/ui/internal/dom"

	"github.com/rivo/uniseg"

	"github.com/merlindorin/ember/geom"
	"github.com/merlindorin/ember/ui/internal/box"
	"github.com/merlindorin/ember/ui/internal/style"
)

// inlineContent is a block container's inline formatting context: its text, the inline
// elements around it, and its atomic inlines (inline-block, inline-flex, inline-grid),
// laid out in lines.
type inlineContent struct {
	owner *node
	runs  []*textRun
	spans []*inlineSpan
	// pieces is the content cut at break opportunities, text measured once.
	pieces []piece
	// metrics caches each style's font metrics for this layout.
	metrics map[*style.Computed]box.Metrics
	// marker is a summary's disclosure triangle, nil elsewhere.
	marker *box.Box
	// oofs are the out-of-flow elements inside the content, by their node.
	oofs map[*node]*inlineOOF
	// anchors are where in the runs each sits: before run i.
	anchors []anchorAt
	// unit is the layout unit, per point, that each run's text on a line rounds up to.
	unit float32
	// inWord is set while the runs gathered so far end inside a word, for capitalize.
	inWord bool
	// broken are the lines last broken at brokenAt, and unbroken the lines with
	// no width to fit, kept while no piece's width changes, as a node is
	// measured and laid out at its width and at none several times.
	broken, unbroken []line
	brokenAt         float32
}

// inlineOOF is an absolutely positioned (or fixed) element inside inline content: its
// static position, the line spot it would take, and, when the span holding it is
// positioned, the containing block that span makes. Both are in the owner's border box.
// leafChildren reads it through oofOf.
type inlineOOF struct {
	static point
	// areaAt and areaSize are the containing block when positioned.
	areaAt     point
	areaSize   size
	positioned bool
	span       int
}

type anchorAt struct {
	node *node
	run  int
}

// oofOf is k's place in c's lines when k is out of flow inside it, nil otherwise.
func (c *inlineContent) oofOf(k *node) *inlineOOF {
	if c == nil {
		return nil
	}
	return c.oofs[k]
}

// textRun is one text node, its whitespace processed, or one atomic inline.
type textRun struct {
	box    *box.Box
	parent *box.Box
	st     *style.Computed
	text   string
	atomic *atomicInline
}

// atomicInline is an inline-level box laid out on its own and placed in a line as one piece.
type atomicInline struct {
	node          *node
	margin        rect
	width, height float32
	baseline      float32
	overflow      overflowRect
}

// inlineSpan is an inline element: its box covers the fragments of the runs inside it, and
// its margin, border and padding take room at its start and end, as edge pieces.
type inlineSpan struct {
	box    *box.Box
	parent *box.Box
	runs   []int
	st     *style.Computed
	// first and end are the runs [first, end) inside it; open and close its edge pieces.
	first, end  int
	open, close int
	// margin, border and padding are resolved against the containing block's width.
	margin, border, padding rect
}

type pieceKind uint8

const (
	word pieceKind = iota
	gap
	breakHere
	newline
	atomic
	// edge is a span's start (margin, border and padding on its left) or end; it sticks to
	// the content beside it.
	edge
	// emergency is a break between two graphemes of a word whose run has overflow-wrap:
	// taken only when the word fits no line whole.
	emergency
	// marker is a summary's disclosure triangle and the space after it, first on its first
	// line and stuck to what follows.
	marker
	// anchor is where an out-of-flow element sits, zero wide; span is its index in anchors.
	anchor
)

// piece is a word, a space, a zero-width break opportunity, a forced break, an atomic
// inline, inside run, or a span's edge (span is its index in spans, and run a run beside it).
type piece struct {
	kind  pieceKind
	run   int
	text  string
	width float32
	span  int
	start bool
}

const zeroWidthSpace = "\u200b"

// blanks turns the whitespace that collapses like a space into one.
var blanks = strings.NewReplacer("\t", " ", "\r", " ", "\f", " ") //nolint:gochecknoglobals // a Replacer built once

// collapsedBlanks is text with the whitespace ws collapses turned into spaces: newlines too,
// unless ws is pre-line.
func collapsedBlanks(text string, ws style.WhiteSpace) string {
	if !collapses(ws) {
		return text
	}
	text = blanks.Replace(text)
	if ws != style.PreLine {
		text = strings.ReplaceAll(text, "\n", " ")
	}
	return text
}

func collapses(ws style.WhiteSpace) bool {
	return ws == style.Normal || ws == style.NoWrapSpace || ws == style.PreLine
}

func wraps(ws style.WhiteSpace) bool { return ws != style.NoWrapSpace && ws != style.Pre }

func isAtomicDisplay(d style.Display) bool {
	return d == style.InlineBlock || d == style.InlineFlex || d == style.InlineGrid
}

// inlineOf gathers the inline content of items for owner, nil when it is only collapsible
// whitespace.
func (t *tree) inlineOf(owner *node, items []item) *inlineContent {
	c := &inlineContent{owner: owner, unit: unitsPerPoint(t.fonts)}
	t.gather(c, owner.box, items)
	if n := owner.box.Node; n != nil && n.Tag() == dom.Summary && owner.st.ListItem && markerOf(owner.st) != "" {
		c.marker = &box.Box{Kind: box.Marker, Node: n, Style: owner.st}
	}
	visible := c.marker != nil
	for _, r := range c.runs {
		if r.atomic != nil || !collapses(r.st.WhiteSpace) || !isCollapsibleSpace(r.text) {
			visible = true
		}
	}
	if !visible {
		return nil
	}
	for _, r := range c.runs {
		if r.atomic != nil {
			owner.kids = append(owner.kids, r.atomic.node)
		}
	}
	c.cut(t.fonts)
	return c
}

func (t *tree) gather(c *inlineContent, parent *box.Box, items []item) {
	for _, it := range items {
		if it.text != nil {
			st := c.owner.st
			if p := it.text.Parent(); p != nil && p != c.owner.box.Node {
				st = styleOf(p)
			}
			b := &box.Box{Kind: box.TextRun, Node: it.text, Style: st}
			c.runs = append(c.runs,
				&textRun{box: b, parent: parent, st: st,
					text: shownText(Translated(it.text, t.translate), st, it.text.Parent(), &c.inWord)})
			continue
		}
		st := styleOf(it.element)
		if st.Display == style.DisplayNone {
			continue
		}
		if outOfFlow(st.Position) {
			if parent == c.owner.box {
				continue
			}
			n := t.build(it.element, c.owner)
			c.owner.kids = append(c.owner.kids, n)
			if c.oofs == nil {
				c.oofs = map[*node]*inlineOOF{}
			}
			c.oofs[n] = &inlineOOF{span: -1}
			c.anchors = append(c.anchors, anchorAt{node: n, run: len(c.runs)})
			continue
		}
		if isAtomicDisplay(st.Display) {
			n := t.build(it.element, c.owner)
			n.atomic = true
			c.runs = append(c.runs, &textRun{box: n.box, parent: parent, st: st, atomic: &atomicInline{node: n}})
			continue
		}
		b := &box.Box{Kind: box.Element, Node: it.element, Style: st}
		t.place(it.element, b)
		first := len(c.runs)
		t.gather(c, b, t.flatten(c.owner, it.element.Children()))
		s := &inlineSpan{box: b, parent: parent, st: st, first: first, end: len(c.runs)}
		for i := first; i < len(c.runs); i++ {
			s.runs = append(s.runs, i)
		}
		c.spans = append(c.spans, s)
	}
}

// cut splits the runs into pieces: whitespace collapsed as each run's white-space says,
// across runs too, so one space separates two runs at most; each span's start and end are
// edge pieces around its runs.
func (c *inlineContent) cut(fonts box.Fonts) {
	c.pieces, c.broken, c.unbroken = c.pieces[:0], nil, nil
	afterSpace := true
	soft := c.softBreaks()
	if c.marker != nil {
		c.pieces = append(c.pieces, piece{kind: marker, width: markerWidth(fonts, c.owner.st)})
	}
	for i, r := range c.runs {
		c.openSpans(i)
		c.anchorsAt(i)
		if r.atomic != nil {
			c.pieces = append(c.pieces, piece{kind: atomic, run: i})
			afterSpace = false
			c.closeSpans(i)
			continue
		}
		afterSpace = c.cutRun(fonts, i, r, soft[i], afterSpace)
		c.closeSpans(i)
	}
	c.openSpans(len(c.runs))
	c.anchorsAt(len(c.runs))
}

// cutRun cuts run i's text into pieces, afterSpace saying whether a collapsible space
// came last before it, and reports the same after it.
func (c *inlineContent) cutRun(fonts box.Fonts, i int, r *textRun, soft map[int]bool, afterSpace bool) bool {
	ws := r.st.WhiteSpace
	text := collapsedBlanks(r.text, ws)
	var b strings.Builder
	splits := r.st.BreakWord && wraps(ws)
	emit := func(kind pieceKind) {
		if b.Len() == 0 && kind == word {
			return
		}
		s := b.String()
		b.Reset()
		if kind == word && splits {
			c.graphemes(fonts, r.st, i, s)
			return
		}
		p := piece{kind: kind, run: i, text: s}
		if kind == word || kind == gap {
			p.width = fonts.Advance(r.st.Font, strings.ReplaceAll(s, zeroWidthSpace, "")) +
				r.st.LetterSpacing*float32(len([]rune(s)))
		}
		c.pieces = append(c.pieces, p)
	}
	for at, ch := range text {
		if soft[at] && ch != ' ' && ch != '\n' && string(ch) != zeroWidthSpace {
			if b.Len() > 0 {
				emit(word)
				c.pieces = append(c.pieces, piece{kind: breakHere, run: i})
			} else if at == 0 && c.afterWord() {
				c.pieces = append(c.pieces, piece{kind: breakHere, run: i})
			}
		}
		switch {
		case ch == '\n':
			emit(word)
			afterSpace = true
			c.pieces = append(c.pieces, piece{kind: newline, run: i})
		case ch == ' ':
			emit(word)
			if collapses(ws) && afterSpace {
				continue
			}
			b.WriteRune(' ')
			emit(gap)
			afterSpace = true
		case string(ch) == zeroWidthSpace:
			emit(word)
			c.pieces = append(c.pieces, piece{kind: breakHere, run: i})
		default:
			b.WriteRune(ch)
			afterSpace = false
		}
	}
	emit(word)
	return afterSpace
}

// anchorsAt adds the anchors of the out-of-flow elements before run i, and notes the
// innermost open span each sits in.
func (c *inlineContent) anchorsAt(i int) {
	for k, a := range c.anchors {
		if a.run != i {
			continue
		}
		o := c.oofs[a.node]
		o.span = -1
		for si, sp := range c.spans {
			if sp.first <= i && i < sp.end || sp.first == i && sp.end == i {
				if o.span < 0 || c.spans[o.span].first < sp.first || c.spans[o.span].end > sp.end {
					o.span = si
				}
			}
		}
		c.pieces = append(c.pieces, piece{kind: anchor, run: min(i, max(len(c.runs)-1, 0)), span: k})
	}
}

// graphemes adds word s of run i as one piece per grapheme, with emergency breaks between
// them, each measured as the growth of the word's advance so that they add up to it.
func (c *inlineContent) graphemes(fonts box.Fonts, st *style.Computed, i int, s string) {
	rest, state := s, -1
	var g string
	prev := float32(0)
	for at := 0; len(rest) > 0; {
		g, rest, _, state = uniseg.FirstGraphemeClusterInString(rest, state)
		at += len(g)
		adv := fonts.Advance(st.Font, strings.ReplaceAll(s[:at], zeroWidthSpace, ""))
		w := adv - prev + st.LetterSpacing*float32(len([]rune(g)))
		prev = adv
		if at > len(g) {
			c.pieces = append(c.pieces, piece{kind: emergency, run: i})
		}
		c.pieces = append(c.pieces, piece{kind: word, run: i, text: g, width: w})
	}
}

// markerOf is the disclosure triangle list-style gives a ::marker, as
// Chrome's text for it: "" for none.
func markerOf(st *style.Computed) string {
	switch st.ListStyle {
	case style.ListDisclosureClosed:
		return "▸"
	case style.ListDisclosureOpen:
		return "▾"
	case style.ListDisc, style.ListNone:
	}
	return ""
}

// markerWidth is a summary's ::marker, the triangle and a space in its font.
func markerWidth(fonts box.Fonts, st *style.Computed) float32 {
	return fonts.Advance(st.Font, markerOf(st)+" ")
}

// afterWord reports whether the last piece, edges aside, is a word.
func (c *inlineContent) afterWord() bool {
	for k := len(c.pieces) - 1; k >= 0; k-- {
		switch c.pieces[k].kind {
		case edge:
			continue
		case word:
			return true
		case gap, breakHere, newline, atomic, emergency, marker, anchor:
		}
		return false
	}
	return false
}

// openSpans adds the start edges of the spans whose runs begin at run i, outer spans first
// (c.spans lists inner spans first), and both edges of the empty ones there.
func (c *inlineContent) openSpans(i int) {
	run := min(i, len(c.runs)-1)
	for k := len(c.spans) - 1; k >= 0; k-- {
		s := c.spans[k]
		if s.first != i {
			continue
		}
		s.open = len(c.pieces)
		c.pieces = append(c.pieces, piece{kind: edge, run: run, span: k, start: true})
		if s.end == i {
			s.close = len(c.pieces)
			c.pieces = append(c.pieces, piece{kind: edge, run: run, span: k})
		}
	}
}

// closeSpans adds the end edges of the spans whose last run is i, inner spans first.
func (c *inlineContent) closeSpans(i int) {
	for k, s := range c.spans {
		if s.end == i+1 && s.first <= i {
			s.close = len(c.pieces)
			c.pieces = append(c.pieces, piece{kind: edge, run: i, span: k})
		}
	}
}

// softBreaks finds UAX #14's line break opportunities inside words, run by run: soft[i][at]
// is set when a line may break before byte at of run i's text. The runs are read as one text,
// atoms as U+FFFC, so the rules see across run boundaries; breaks at spaces and newlines are
// cut's own. As in Chrome, a path such as a/b doesn't break after its slash.
func (c *inlineContent) softBreaks() []map[int]bool {
	var all strings.Builder
	starts := make([]int, len(c.runs))
	for i, r := range c.runs {
		starts[i] = all.Len()
		if r.atomic != nil {
			all.WriteString("\uFFFC")
		} else {
			all.WriteString(r.text)
		}
	}
	soft := make([]map[int]bool, len(c.runs))
	text := all.String()
	rest, pos, state := text, 0, -1
	run := 0
	for len(rest) > 0 {
		var seg string
		seg, rest, _, state = uniseg.FirstLineSegmentInString(rest, state)
		pos += len(seg)
		if len(rest) == 0 {
			break
		}
		for run+1 < len(c.runs) && starts[run+1] <= pos {
			run++
		}
		if r, _ := utf8.DecodeLastRuneInString(text[:pos]); r == '/' {
			if next, _ := utf8.DecodeRuneInString(rest); unicode.IsLetter(next) || unicode.IsDigit(next) {
				continue
			}
		}
		if c.runs[run].atomic != nil {
			continue
		}
		if soft[run] == nil {
			soft[run] = map[int]bool{}
		}
		soft[run][pos-starts[run]] = true
	}
	return soft
}

// sizeAtomics sizes the spans' edges against the containing block's width, and lays out
// each atomic inline as a shrink-to-fit box in avail, setting its piece's width to its
// margin box.
func (t *tree) sizeAtomics(c *inlineContent, avail space, parent size) {
	for _, sp := range c.spans {
		sp.margin = edges(sp.st.Margin, parent.w)
		sp.padding = edges(sp.st.Padding, parent.w)
		b := sp.st.Border
		sp.border = rect{l: b.Left, r: b.Right, t: b.Top, b: b.Bottom}
	}
	for i := range c.pieces {
		if p := &c.pieces[i]; p.kind == edge {
			sp := c.spans[p.span]
			if p.start {
				c.resize(p, sp.margin.l+sp.border.l+sp.padding.l)
			} else {
				c.resize(p, sp.margin.r+sp.border.r+sp.padding.r)
			}
		}
	}
	for i := range c.pieces {
		p := &c.pieces[i]
		if p.kind != atomic {
			continue
		}
		a := c.runs[p.run].atomic
		st := a.node.st
		a.margin = edges(st.Margin, parent.w)
		inner := avail.mSub(a.margin.horizontal()).mMax(0)
		if k, sp := resolveSizingKeyword(st.Size.Width, inner.value(), parent.w); k == keywordMeasure {
			inner = sp
		}
		a.width = t.measureChildSize(a.node, noSize, parent,
			spaces{inner, maxContentSpace}, inherentSize, horizontal, [2]bool{})
		out := t.performChildLayout(a.node, size{a.width, none}, parent,
			spaces{def(a.width), maxContentSpace}, inherentSize)
		a.height = out.size.h
		a.baseline = out.size.h
		a.overflow = out.overflow
		a.node.pendingOOF = out.oof
		switch {
		case isScrollContainer(st) && a.node.control != inputControl && a.node.control != selectControl:
		case some(out.lastBaseline):
			a.baseline = out.lastBaseline
		case some(out.firstBaseline):
			a.baseline = out.firstBaseline
		}
		c.resize(p, a.width+a.margin.horizontal())
	}
}

// contentWidths is the min-content width (the widest unbreakable segment, or the widest
// line where nothing wraps) and the max-content width (the widest line between forced
// breaks).
func (t *tree) contentWidths(c *inlineContent, parent size) (float32, float32) {
	var minW, maxW float32
	t.sizeAtomics(c, maxContentSpace, parent)
	for _, l := range c.lines(unbounded) {
		maxW = max(maxW, l.width)
	}
	t.sizeAtomics(c, minContentSpace, parent)
	if !c.anyWraps() {
		for _, l := range c.lines(unbounded) {
			minW = max(minW, l.width)
		}
		return minW, maxW
	}
	cur := newPen(c.unit, 0)
	for _, p := range c.pieces {
		switch p.kind {
		case word, edge, marker:
			cur.step(p, 0)
			minW = max(minW, cur.end())
		case emergency:
			if c.runs[p.run].st.BreakAnywhere {
				cur = newPen(c.unit, 0)
			}
		case atomic:
			minW = max(minW, p.width)
			cur = newPen(c.unit, 0)
		case gap, breakHere, newline, anchor:
			cur = newPen(c.unit, 0)
		}
	}
	return minW, maxW
}

func (c *inlineContent) anyWraps() bool {
	for _, r := range c.runs {
		if r.atomic == nil && wraps(r.st.WhiteSpace) {
			return true
		}
	}
	return false
}

// line is the pieces [from, to) on one line, without its trailing spaces; forced is set when
// a newline ends it.
type line struct {
	from, to int
	width    float32
	forced   bool
}

func (c *inlineContent) wrapsAt(i int) bool {
	r := c.runs[c.pieces[i].run]
	if r.atomic != nil {
		return wraps(c.owner.st.WhiteSpace)
	}
	return wraps(r.st.WhiteSpace)
}

// unitEnd is where the unit starting at piece i ends: a word, edge, marker or anchor
// takes the pieces glued after it.
func (c *inlineContent) unitEnd(i int) int {
	end := i + 1
	if k := c.pieces[i].kind; k == word || k == edge || k == marker || k == anchor {
		for end < len(c.pieces) && glued(c.pieces[end].kind) {
			end++
		}
	}
	return end
}

// measureUnit is the pen past pieces i to end from at, with the gaps pending since then,
// and the pen past them alone.
func (c *inlineContent) measureUnit(at pen, pending, i, end int) (pen, pen) {
	next := at
	if pending >= 0 {
		for k := pending; k < i; k++ {
			if c.pieces[k].kind == gap {
				next.step(c.pieces[k], 0)
			}
		}
	}
	alone := newPen(c.unit, 0)
	for k := i; k < end; k++ {
		next.step(c.pieces[k], 0)
		alone.step(c.pieces[k], 0)
	}
	return next, alone
}

// breaker is lines' state while it breaks: the lines so far, the open one and the pen
// across it, the first gap pending before the next unit, and whether the line is empty.
type breaker struct {
	c       *inlineContent
	out     []line
	cur     line
	at      pen
	pending int
	empty   bool
}

func (b *breaker) finish(next int) {
	b.cur.width = b.at.end()
	b.out = append(b.out, b.cur)
	b.cur = line{from: next, to: next}
	b.at = newPen(b.c.unit, 0)
	b.pending = -1
	b.empty = true
}

func (b *breaker) breakAll(width float32) []line {
	c := b.c
	i := 0
	for i < len(c.pieces) {
		p := c.pieces[i]
		switch p.kind {
		case newline:
			b.cur.forced = true
			b.finish(i + 1)
			i++
			continue
		case gap:
			if b.empty && collapses(c.runs[p.run].st.WhiteSpace) {
				b.cur.from, b.cur.to = i+1, i+1
				i++
				continue
			}
			if b.pending < 0 {
				b.pending = i
			}
			i++
			continue
		case breakHere:
			i++
			continue
		case word, atomic, edge, emergency, marker, anchor:
		}
		end := c.unitEnd(i)
		next, alone := c.measureUnit(b.at, b.pending, i, end)
		if !b.empty && c.wrapsAt(i) && next.end() > width {
			b.finish(i)
			continue
		}
		if b.empty && alone.end() > width {
			if take, ok := c.emergencyBreak(i, end, width); ok {
				for k := i; k < take; k++ {
					b.at.step(c.pieces[k], 0)
				}
				b.cur.to = take
				b.empty = false
				b.finish(take + 1)
				i = take + 1
				continue
			}
		}
		b.at = next
		b.pending = -1
		b.cur.to = end
		b.empty = false
		i = end
	}
	if !b.empty || len(b.out) == 0 && b.cur.to > b.cur.from {
		b.cur.width = b.at.end()
		b.out = append(b.out, b.cur)
	}
	if b.out == nil {
		b.out = []line{}
	}
	return b.out
}

// lines breaks the pieces greedily into lines no wider than width where they may wrap.
// unbounded is the width lines take for no width to fit: every break forced.
const unbounded = float32(1e30)

func (c *inlineContent) lines(width float32) []line {
	switch {
	case width >= unbounded && c.unbroken != nil:
		return c.unbroken
	case width < unbounded && c.broken != nil && c.brokenAt == width:
		return c.broken
	}
	b := breaker{c: c, at: newPen(c.unit, 0), pending: -1, empty: true}
	out := b.breakAll(width)
	if width >= unbounded {
		c.unbroken = out
	} else {
		c.broken, c.brokenAt = out, width
	}
	return out
}

// resize gives piece p width w, forgetting the lines broken at the old one.
func (c *inlineContent) resize(p *piece, w float32) {
	if p.width != w {
		p.width = w
		c.broken, c.unbroken = nil, nil
	}
}

func glued(k pieceKind) bool {
	return k == word || k == edge || k == emergency || k == marker || k == anchor
}

// emergencyBreak finds where to cut the unbreakable pieces [from, end) that fit no line:
// at the last emergency break that fits in width, or after the first grapheme. It returns
// the emergency piece's index and the width before it, false without such breaks.
func (c *inlineContent) emergencyBreak(from, end int, width float32) (int, bool) {
	take := -1
	acc := newPen(c.unit, 0)
	for k := from; k < end; k++ {
		if c.pieces[k].kind == emergency {
			w := acc.end()
			if w > width && take >= 0 {
				break
			}
			if take < 0 || w <= width {
				take = k
			}
		}
		acc.step(c.pieces[k], 0)
	}
	return take, take >= 0
}

// metricsOf is st's font metrics, measured once per inline context and layout.
func (t *tree) metricsOf(c *inlineContent, st *style.Computed) box.Metrics {
	if m, ok := c.metrics[st]; ok {
		return m
	}
	if c.metrics == nil {
		c.metrics = map[*style.Computed]box.Metrics{}
	}
	m := t.fonts.Metrics(st.Font)
	c.metrics[st] = m
	return m
}

// lineHeight is one run's line height and the baseline's place in it (half-leading above
// the ascent).
func (t *tree) lineHeight(c *inlineContent, st *style.Computed) (float32, float32) {
	return lineHeightOf(t.metricsOf(c, st), st)
}

// lineHeightOf is the line height a face with metrics m takes in a run of style st, and
// the baseline's place in it.
func lineHeightOf(m box.Metrics, st *style.Computed) (float32, float32) {
	h := m.Ascent + m.Descent + m.LineGap
	switch {
	case st.LineHeight.Normal:
	case st.LineHeight.Factor != 0:
		h = st.LineHeight.Factor * st.Font.Size
	default:
		h = st.LineHeight.Px
	}
	// Chrome puts the floor of half the leading above the glyphs, the rest below.
	return h, m.Ascent + float32(math.Floor(float64(h-m.Ascent-m.Descent)/2))
}

// lineBox is the height of a line and its baseline: the strut of the owner, then each run
// on it, atomic inlines by their margin boxes on their baselines.
func (t *tree) lineBox(c *inlineContent, l line) (float32, float32) {
	h, base := t.lineHeight(c, c.owner.st)
	above, below := base, h-base
	for i := l.from; i < l.to; i++ {
		if k := c.pieces[i].kind; k == edge || k == marker || k == anchor {
			continue
		}
		r := c.runs[c.pieces[i].run]
		if a := r.atomic; a != nil {
			top := a.margin.t + a.baseline
			above = max(above, top)
			below = max(below, a.height+a.margin.vertical()-top)
			continue
		}
		rh, rb := t.lineHeight(c, r.st)
		above, below = max(above, rb), max(below, rh-rb)
		if m, ok := t.fallbackMetrics(c.pieces[i], r.st); ok {
			fh, fb := lineHeightOf(m, r.st)
			above, below = max(above, fb), max(below, fh-fb)
		}
	}
	return above + below, above
}

// fallbackMetrics is the metrics of the fallback faces a word draws with, when the fonts
// have fallbacks and its text needs them: Chrome stretches a line-height: normal line to
// fit them, and ignores them under any other line-height.
func (t *tree) fallbackMetrics(p piece, st *style.Computed) (box.Metrics, bool) {
	ff, ok := t.fonts.(box.FallbackFonts)
	if !ok || p.kind != word || !st.LineHeight.Normal || isASCII(p.text) {
		return box.Metrics{}, false
	}
	return ff.FallbackMetrics(st.Font, p.text)
}

func isASCII(s string) bool {
	for i := 0; i < len(s); i++ {
		if s[i] >= utf8.RuneSelf {
			return false
		}
	}
	return true
}

// measure is the inline content's size and baselines, Taffy's measure function for text.
// Lines break at the available width; content wider than that (a word that breaks only
// with overflow-wrap, or not at all) widens the size it reports, not the lines.
func (t *tree) measure(c *inlineContent, known size, avail spaces, parent size) (size, float32, float32) {
	minW, maxW := t.contentWidths(c, parent)
	w := known.w
	lineW := w
	if !some(w) {
		switch avail.w.kind {
		case minContent:
			lineW = minW
		case maxContent:
			lineW = maxW
		case definite:
			lineW = min(avail.w.v, maxW)
		}
		w = max(lineW, minW)
	}
	t.sizeAtomics(c, def(lineW), parent)
	var h float32
	first, last := none, none
	for i, l := range c.lines(lineW) {
		lh, lb := t.lineBox(c, l)
		if i == 0 {
			first = h + lb
		}
		last = h + lb
		h += lh
	}
	if some(known.h) {
		h = known.h
	}
	return size{w, h}, first, last
}

// placeText lays the text out at the owner's content width and places its atomic inlines.
func (t *tree) placeText(n *node, s size, inset rect) {
	c := n.inline
	content := s.w - inset.horizontal()
	t.sizeAtomics(c, def(content), size{content, none})
	ls := c.lines(content)
	y := t.textTop(n, ls, inset.t, s.h-inset.vertical())
	var spanLines map[*box.Box][]geom.Rect
	if len(c.oofs) > 0 {
		spanLines = map[*box.Box][]geom.Rect{}
	}
	for li, l := range ls {
		lh, lb := t.lineBox(c, l)
		off, perGap := c.align(n.st.TextAlign, l, li == len(ls)-1, content-l.width)
		start := inset.l + off
		if spanLines != nil {
			t.spanRects(c, l, c.pieces[l.from:l.to], start, perGap, y, lb,
				map[*box.Box]geom.Rect{}, make([]bool, len(c.spans)), spanLines)
		}
		at := newPen(c.unit, start)
		for i := l.from; i < l.to; i++ {
			p := c.pieces[i]
			x := at.step(p, perGap)
			if p.kind == anchor {
				c.oofs[c.anchors[p.span].node].static = point{x, y}
			}
			if p.kind == atomic {
				a := c.runs[p.run].atomic
				l := &a.node.unrounded
				l.location = point{x + a.margin.l, y + lb - a.baseline}
				l.margin = a.margin
				l.size = size{a.width, a.height}
				st := a.node.st
				l.padding = edges(st.Padding, content)
				l.border = borders(st)
				g := scrollbarGutter(st)
				l.scrollbar = size{g.x, g.y}
				l.overflow = a.overflow
			}
		}
		y += lh
	}
	for _, o := range c.oofs {
		c.containingSpan(o, spanLines)
	}
}

// containingSpan makes the nearest positioned span holding o its containing block: the
// box from the start of its first fragment's padding box to the end of its last one's,
// as CSS defines it for an inline element.
func (c *inlineContent) containingSpan(o *inlineOOF, lines map[*box.Box][]geom.Rect) {
	o.positioned = false
	for k := o.span; k >= 0; k = c.enclosing(k) {
		sp := c.spans[k]
		if sp.st.Position == style.Static {
			continue
		}
		frags := lines[sp.box]
		if len(frags) == 0 {
			return
		}
		first, last := frags[0], frags[len(frags)-1]
		x0 := first.X + sp.border.l
		x1 := last.X + last.W - sp.border.r
		y0 := first.Y + sp.border.t
		y1 := last.Y + last.H - sp.border.b
		o.areaAt = point{x0, y0}
		o.areaSize = size{max(x1-x0, 0), max(y1-y0, 0)}
		o.positioned = true
		return
	}
}

// enclosing is the innermost span holding span k, -1 for none.
func (c *inlineContent) enclosing(k int) int {
	in := c.spans[k]
	best := -1
	for i, sp := range c.spans {
		if i == k || sp.first > in.first || sp.end < in.end || sp.first == in.first && sp.end == in.end && i < k {
			continue
		}
		if best < 0 || c.spans[best].first <= sp.first && sp.end <= c.spans[best].end {
			best = i
		}
	}
	return best
}

// writeText writes the fragments of the laid-out text into the box tree.
func (t *tree) writeText(n *node) {
	for p := n; p != nil; p = p.parent {
		if p.st.Display == style.DisplayNone {
			return
		}
	}
	c := n.inline
	for _, s := range c.spans {
		s.box.Children = s.box.Children[:0]
	}
	for _, r := range c.runs {
		if r.atomic == nil {
			r.box.Lines = r.box.Lines[:0]
		}
	}
	f := n.final
	inset := f.padding.add(f.border)
	content := f.size.w - inset.horizontal() - f.scrollbar.w
	t.sizeAtomics(c, def(content), size{content, none})
	runRects, spanRects, spanLines := t.writeLines(n, c, inset, content)
	origin := func(b *box.Box) geom.Vec2 {
		if r, ok := spanRects[b]; ok {
			return geom.Vec2{X: r.X, Y: r.Y}
		}
		return geom.Vec2{}
	}
	placeSpans(c, spanRects, spanLines, origin)
	t.attachInline(n, c, runRects, origin)
}

// placeSpans gives each span its box from the rectangles its lines took.
func placeSpans(c *inlineContent, spanRects map[*box.Box]geom.Rect, spanLines map[*box.Box][]geom.Rect,
	origin func(*box.Box) geom.Vec2) {
	for _, s := range c.spans {
		r := spanRects[s.box]
		o := origin(s.parent)
		s.box.Pos = geom.Vec2{X: r.X - o.X, Y: r.Y - o.Y}
		s.box.Size = geom.Vec2{X: r.W, Y: r.H}
		s.box.Margin = box.Edges{Top: s.margin.t, Right: s.margin.r, Bottom: s.margin.b, Left: s.margin.l}
		s.box.Border = box.Edges{Top: s.border.t, Right: s.border.r, Bottom: s.border.b, Left: s.border.l}
		s.box.Padding = box.Edges{Top: s.padding.t, Right: s.padding.r, Bottom: s.padding.b, Left: s.padding.l}
		s.box.Fragments = s.box.Fragments[:0]
		for _, f := range spanLines[s.box] {
			s.box.Fragments = append(s.box.Fragments, geom.Rect{X: f.X - r.X, Y: f.Y - r.Y, W: f.W, H: f.H})
		}
	}
}

// attachInline places each run's box over its fragments and appends the runs, spans and
// marker to their parents, in order.
func (t *tree) attachInline(n *node, c *inlineContent, runRects []geom.Rect, origin func(*box.Box) geom.Vec2) {
	children := map[*box.Box][]*box.Box{}
	var order []*box.Box
	add := func(parent, child *box.Box) {
		if _, ok := children[parent]; !ok {
			order = append(order, parent)
		}
		children[parent] = append(children[parent], child)
	}
	openAt := func(i int) {
		for k := len(c.spans) - 1; k >= 0; k-- {
			if c.spans[k].first == i {
				add(c.spans[k].parent, c.spans[k].box)
			}
		}
	}
	if c.marker != nil {
		add(n.box, c.marker)
	}
	for i, r := range c.runs {
		openAt(i)
		if r.atomic != nil {
			continue
		}
		rr := runRects[i]
		o := origin(r.parent)
		r.box.Pos = geom.Vec2{X: rr.X - o.X, Y: rr.Y - o.Y}
		r.box.Size = geom.Vec2{X: rr.W, Y: rr.H}
		if r.box.Node != nil {
			t.place(r.box.Node, r.box)
		}
		for j := range r.box.Lines {
			r.box.Lines[j].Rect.X -= rr.X
			r.box.Lines[j].Rect.Y -= rr.Y
			r.box.Lines[j].Baseline -= rr.Y
		}
		if len(r.box.Lines) > 0 {
			r.box.Baseline = r.box.Lines[0].Baseline
		}
		add(r.parent, r.box)
	}
	openAt(len(c.runs))
	for _, p := range order {
		p.Append(children[p]...)
	}
}

// writeLines writes each line's fragments into its runs' boxes, and returns the rectangles
// the runs and spans took.
func (t *tree) writeLines(n *node, c *inlineContent, inset rect, content float32) ([]geom.Rect,
	map[*box.Box]geom.Rect, map[*box.Box][]geom.Rect) {
	f := n.final
	ellipsis := n.st.Ellipsis && n.st.Overflow.Width != style.Visible
	runRects := make([]geom.Rect, len(c.runs))
	seen := make([]bool, len(c.runs))
	spanRects := make(map[*box.Box]geom.Rect)
	spanSeen := make([]bool, len(c.spans))
	spanLines := make(map[*box.Box][]geom.Rect)
	ls := c.lines(content)
	y := t.textTop(n, ls, inset.t, f.size.h-inset.vertical()-f.scrollbar.h)
	for li, l := range ls {
		lh, lb := t.lineBox(c, l)
		off, perGap := c.align(n.st.TextAlign, l, li == len(ls)-1, content-l.width)
		x := inset.l + off
		pieces := c.pieces[l.from:l.to]
		t.spanRects(c, l, pieces, x, perGap, y, lb, spanRects, spanSeen, spanLines)
		cut, cutStart := false, false
		if ellipsis && l.width > content {
			pieces, x, cut, cutStart = t.ellipsizeLine(n, c, pieces, x, inset.l, content)
		}
		walk := newPen(c.unit, x)
		for i := 0; i < len(pieces); {
			run := pieces[i].run
			if k := pieces[i].kind; k == atomic || k == edge || k == marker || k == anchor || k == gap && perGap > 0 {
				at := walk.step(pieces[i], perGap)
				if k == marker {
					c.placeMarker(geom.Rect{X: at, Y: y, W: pieces[i].width, H: lh}, lb)
				}
				i++
				continue
			}
			var text strings.Builder
			start := walk.at()
			for i < len(pieces) && pieces[i].run == run && isText(pieces[i].kind) && (pieces[i].kind != gap || perGap == 0) {
				at := walk.step(pieces[i], perGap)
				if text.Len() == 0 {
					start = at
				}
				if pieces[i].kind == word || pieces[i].kind == gap {
					text.WriteString(pieces[i].text)
				}
				i++
			}
			end := walk.end()
			if i < len(pieces) && isText(pieces[i].kind) && pieces[i].run == run {
				end = walk.at()
			}
			r := geom.Rect{X: start, Y: y, W: end - start, H: lh}
			frag := t.fragment(c.runs[run].st, text.String(), r, y+lb, cutStart, cut && i == len(pieces))
			cutStart = false
			c.runs[run].box.Lines = append(c.runs[run].box.Lines, frag)
			runRects[run] = unionRect(runRects[run], frag.Rect, seen[run])
			seen[run] = true
		}
		y += lh
	}
	return runRects, spanRects, spanLines
}

// fragment is a line's piece of a run's text at r, with the ellipsis where the line was
// cut before or after it.
func (t *tree) fragment(st *style.Computed, text string, r geom.Rect, baseline float32,
	cutBefore, cutAfter bool) box.Fragment {
	frag := box.Fragment{Text: strings.ReplaceAll(text, zeroWidthSpace, ""), Rect: r, Baseline: baseline}
	if cutBefore {
		frag.Ellipsis = true
	}
	if cutAfter {
		frag.Text += "…"
		frag.Ellipsis = true
		frag.Rect.W += t.fonts.Advance(st.Font, "…")
	}
	return frag
}

// placeMarker puts the ::marker box at r, its triangle on baseline lb.
func (c *inlineContent) placeMarker(r geom.Rect, lb float32) {
	c.marker.Pos = geom.Vec2{X: r.X, Y: r.Y}
	c.marker.Size = geom.Vec2{X: r.W, Y: r.H}
	c.marker.Lines = []box.Fragment{{Text: markerOf(c.owner.st), Rect: geom.Rect{W: r.W, H: r.H}, Baseline: lb}}
}

// ellipsizeLine cuts a line too wide for content and puts the ellipsis at its end, or at
// its start, then flush with the content's end.
func (t *tree) ellipsizeLine(n *node, c *inlineContent, pieces []piece, x, left, content float32) ([]piece, float32,
	bool, bool) {
	if !n.st.EllipsisStart {
		kept, cut := t.ellipsize(c, pieces, content)
		return kept, x, cut, false
	}
	pieces = t.ellipsizeStart(c, pieces, content)
	w := newPen(c.unit, 0)
	for _, p := range pieces {
		w.step(p, 0)
	}
	return pieces, left + content - w.end(), false, true
}

// spanRects adds line l's part of each span to its border box: from its start edge (or the
// line's start) to its end edge (or the line's end), over its content area plus vertical
// padding and border. The pieces are the line's before any ellipsis, as Chrome's boxes are.
func (t *tree) spanRects(c *inlineContent, l line, pieces []piece, x, perGap, y,
	lb float32, rects map[*box.Box]geom.Rect, seen []bool, lines map[*box.Box][]geom.Rect) {
	at := make([]float32, len(pieces)+1)
	walk := newPen(c.unit, x)
	for i, p := range pieces {
		at[i] = walk.step(p, perGap)
	}
	at[len(pieces)] = walk.end()
	for k, sp := range c.spans {
		lo, hi := max(sp.open, l.from), min(sp.close, l.from+len(pieces)-1)
		if lo > hi {
			continue
		}
		left, right := at[lo-l.from], at[hi-l.from+1]
		if lo == sp.open {
			left += sp.margin.l
		}
		if hi == sp.close {
			right -= sp.margin.r
		}
		m := t.metricsOf(c, sp.st)
		top := y + lb - m.Ascent - sp.padding.t - sp.border.t
		bottom := y + lb + m.Descent + sp.padding.b + sp.border.b
		r := geom.Rect{X: left, Y: top, W: right - left, H: bottom - top}
		rects[sp.box] = unionRect(rects[sp.box], r, seen[k])
		seen[k] = true
		lines[sp.box] = append(lines[sp.box], r)
	}
}

// ellipsize drops the line's end until it and an ellipsis fit in width.
func (t *tree) ellipsize(c *inlineContent, pieces []piece, width float32) ([]piece, bool) {
	last := pieces[len(pieces)-1].run
	room := width - t.fonts.Advance(c.runs[last].st.Font, "…")
	out := make([]piece, 0, len(pieces))
	used := float32(0)
	for _, p := range pieces {
		if used+p.width <= room {
			out = append(out, p)
			used += p.width
			continue
		}
		if p.kind == word {
			if head, w := t.fitPrefix(c.runs[p.run].st, p.text, room-used); head != "" {
				out = append(out, piece{kind: word, run: p.run, text: head, width: w})
			}
		}
		break
	}
	for len(out) > 0 && out[len(out)-1].kind == gap {
		out = out[:len(out)-1]
	}
	if len(out) == 0 {
		out = []piece{{kind: word, run: last}}
	}
	return out, true
}

// ellipsizeStart drops the line's start until an ellipsis and the rest fit in width, the
// ellipsis a word of the first run kept. The line then ends at the box's end, as Chrome's
// right-to-left cut leaves its room before the ellipsis.
func (t *tree) ellipsizeStart(c *inlineContent, pieces []piece, width float32) []piece {
	last := pieces[len(pieces)-1].run
	room := width - t.fonts.Advance(c.runs[last].st.Font, "…")
	used := float32(0)
	i := len(pieces)
	var out []piece
	for ; i > 0; i-- {
		p := pieces[i-1]
		if used+p.width <= room {
			used += p.width
			continue
		}
		if p.kind == word {
			if tail, w := t.fitSuffix(c.runs[p.run].st, p.text, room-used); tail != "" {
				out = append(out, piece{kind: word, run: p.run, text: tail, width: w})
			}
		}
		break
	}
	out = append(out, pieces[i:]...)
	for len(out) > 0 && out[0].kind == gap {
		out = out[1:]
	}
	run := last
	for _, p := range out {
		if p.kind == word {
			run = p.run
			break
		}
	}
	e := piece{kind: word, run: run, text: "…", width: t.fonts.Advance(c.runs[run].st.Font, "…")}
	return append([]piece{e}, out...)
}

// align is where line l starts in free room and how much each of its spaces grows: justify
// spreads the room over the spaces of every line but the last and those a newline ends.
func (c *inlineContent) align(a style.TextAlign, l line, last bool, free float32) (float32, float32) {
	if a != style.Justify || last || l.forced || free <= 0 {
		return alignOffset(a, free), 0
	}
	gaps := 0
	for i := l.from; i < l.to; i++ {
		if c.pieces[i].kind == gap {
			gaps++
		}
	}
	if gaps == 0 {
		return 0, 0
	}
	return 0, free / float32(gaps)
}

// fitPrefix is the longest run of whole graphemes at the start of text no wider than room,
// and its width, found by bisection over the grapheme boundaries.
func (t *tree) fitPrefix(st *style.Computed, text string, room float32) (string, float32) {
	var buf [64]int
	ends := append(buf[:0], 0)
	rest, state := text, -1
	var g string
	for at := 0; len(rest) > 0; {
		g, rest, _, state = uniseg.FirstGraphemeClusterInString(rest, state)
		at += len(g)
		ends = append(ends, at)
	}
	width := func(i int) float32 {
		s := text[:ends[i]]
		return t.fonts.Advance(st.Font, s) + st.LetterSpacing*float32(utf8.RuneCountInString(s))
	}
	lo, hi := 0, len(ends)-1
	for lo < hi {
		mid := (lo + hi + 1) / 2
		if width(mid) <= room {
			lo = mid
		} else {
			hi = mid - 1
		}
	}
	return text[:ends[lo]], width(lo)
}

// fitSuffix is the longest run of whole graphemes at the end of text no wider than room,
// and its width.
func (t *tree) fitSuffix(st *style.Computed, text string, room float32) (string, float32) {
	var buf [64]int
	starts := append(buf[:0], 0)
	rest, state := text, -1
	var g string
	for at := 0; len(rest) > 0; {
		g, rest, _, state = uniseg.FirstGraphemeClusterInString(rest, state)
		at += len(g)
		starts = append(starts, at)
	}
	width := func(i int) float32 {
		s := text[starts[i]:]
		return t.fonts.Advance(st.Font, s) + st.LetterSpacing*float32(utf8.RuneCountInString(s))
	}
	lo, hi := 0, len(starts)-1
	for lo < hi {
		mid := (lo + hi) / 2
		if width(mid) <= room {
			hi = mid
		} else {
			lo = mid + 1
		}
	}
	return text[starts[lo]:], width(lo)
}

func alignOffset(a style.TextAlign, free float32) float32 {
	if free <= 0 {
		return 0
	}
	switch a {
	case style.Right, style.End:
		return free
	case style.Center:
		return free / 2
	case style.Start, style.Left, style.Justify:
	}
	return 0
}

func unionRect(a, b geom.Rect, haveA bool) geom.Rect {
	if !haveA {
		return b
	}
	x0, y0 := min(a.X, b.X), min(a.Y, b.Y)
	x1, y1 := max(a.X+a.W, b.X+b.W), max(a.Y+a.H, b.Y+b.H)
	return geom.Rect{X: x0, Y: y0, W: x1 - x0, H: y1 - y0}
}

// atomsIntoSpans moves the boxes of atomic inlines inside spans from the owner's box to their
// span's, keeping where they are; tree.write calls it once the owner's kids are written.
func (t *tree) atomsIntoSpans(n *node) {
	c := n.inline
	if c == nil {
		return
	}
	for _, r := range c.runs {
		if r.atomic == nil || r.parent == n.box || r.box.Parent != n.box {
			continue
		}
		kids := n.box.Children
		for i, k := range kids {
			if k == r.box {
				n.box.Children = slices.Concat(kids[:i], kids[i+1:])
				break
			}
		}
		for p := r.parent; p != nil && p != n.box; p = p.Parent {
			r.box.Pos = r.box.Pos.Sub(p.Pos)
		}
		r.parent.Append(r.box)
	}
}
