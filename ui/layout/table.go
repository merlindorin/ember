package layout

import (
	"cmp"
	"slices"
	"strconv"

	"github.com/merlindorin/ember/ui/internal/dom"
	"github.com/merlindorin/ember/ui/internal/style"
)

// tableRole marks the nodes of a table: the table, its row groups and its rows; its
// cells are ordinary block containers.
type tableRole uint8

const (
	roleNone tableRole = iota
	roleTable
	roleGroup
	roleRow
	roleCaption
)

// isTable reports a table element left at its table display (an author's flex, grid or
// none wins).
func isTable(e *dom.Node) bool {
	if e.Tag() != dom.Table {
		return false
	}
	d := styleOf(e).Display
	return d == style.Inline || d == style.Block || d == style.InlineBlock
}

func shown(e *dom.Node) bool { return !e.IsText() && styleOf(e).Display != style.DisplayNone }

// buildTable gives a table its captions, its row groups and rows, and each row its
// cells. The groups go as Chrome draws them: the first thead on top, the bodies and loose
// rows, then the first tfoot.
func (t *tree) buildTable(n *node, e *dom.Node) {
	n.role = roleTable
	row := func(r *dom.Node, parent *node) *node {
		if k := t.kept(r); k != nil {
			t.adopt(k, parent)
			return k
		}
		rn := t.leafBox(r, parent, roleRow)
		for _, c := range r.Children() {
			if shown(c) && (c.Tag() == dom.Td || c.Tag() == dom.Th) {
				rn.kids = append(rn.kids, t.build(c, rn))
			}
		}
		return rn
	}
	group := func(c *dom.Node) *node {
		if k := t.kept(c); k != nil {
			t.adopt(k, n)
			return k
		}
		g := t.leafBox(c, n, roleGroup)
		for _, r := range c.Children() {
			if shown(r) && r.Tag() == dom.Tr {
				g.kids = append(g.kids, row(r, g))
			}
		}
		return g
	}
	var head, foot *node
	var body []*node
	for _, c := range e.Children() {
		if !shown(c) {
			continue
		}
		switch c.Tag() {
		case dom.Caption:
			cn := t.build(c, n)
			cn.role = roleCaption
			n.kids = append(n.kids, cn)
		case dom.Thead:
			if g := group(c); head == nil {
				head = g
			} else {
				body = append(body, g)
			}
		case dom.Tfoot:
			if g := group(c); foot == nil {
				foot = g
			} else {
				body = append(body, g)
			}
		case dom.Tbody:
			body = append(body, group(c))
		case dom.Tr:
			body = append(body, row(c, n))
		case dom.TextTag, dom.HTML, dom.Body, dom.Div, dom.Section, dom.Header, dom.Footer, dom.Nav, dom.Main, dom.Aside,
			dom.Article, dom.P, dom.H1, dom.H2, dom.H3, dom.H4, dom.Span, dom.Strong, dom.Em, dom.Code, dom.Label, dom.Button,
			dom.Input, dom.Select, dom.Option, dom.Textarea, dom.Details, dom.Summary, dom.Dialog, dom.Table, dom.Th, dom.Td,
			dom.Progress, dom.Meter, dom.Img, dom.Hr, dom.Canvas, dom.Kbd, dom.Samp, dom.Pre: // not placed by the table
		}
	}
	if head != nil {
		n.kids = append(n.kids, head)
	}
	n.kids = append(n.kids, body...)
	if foot != nil {
		n.kids = append(n.kids, foot)
	}
}

// leafBox is the node of a row group or row, whose children the table places itself.
func (t *tree) leafBox(e *dom.Node, parent *node, role tableRole) *node {
	n := t.element(e, parent)
	n.role = role
	return n
}

// rows lists the table's rows in order, through its row groups.
func (n *node) rows() []*node {
	var rows []*node
	for _, k := range n.kids {
		switch k.role {
		case roleGroup:
			rows = append(rows, k.kids...)
		case roleRow:
			rows = append(rows, k)
		case roleNone, roleTable, roleCaption:
		}
	}
	return rows
}

// captions are the table's captions, in their order.
func (n *node) captions() []*node {
	var out []*node
	for _, k := range n.kids {
		if k.role == roleCaption {
			out = append(out, k)
		}
	}
	return out
}

// captionMin is the width the table's captions need at least, margins included.
func (t *tree) captionMin(n *node) float32 {
	w := float32(0)
	for _, c := range n.captions() {
		m := edges(c.st.Margin, 0)
		w = max(w,
			t.measureChildSize(c, noSize, noSize, spaces{minContentSpace, maxContentSpace},
				inherentSize, horizontal, [2]bool{})+m.horizontal())
	}
	return w
}

// captionHeights are the heights the captions above the table and those below take at
// its width w, margins included.
func (t *tree) captionHeights(n *node, w float32) (float32, float32) {
	var above, below float32
	for _, c := range n.captions() {
		m := edges(c.st.Margin, w)
		h := t.measureChildSize(c, size{w - m.horizontal(), none}, size{w, none},
			spaces{def(w - m.horizontal()), maxContentSpace}, inherentSize, vertical, [2]bool{}) + m.vertical()
		if c.st.CaptionSide == style.CaptionBottom {
			below += h
		} else {
			above += h
		}
	}
	return above, below
}

// placeCaptions lays the captions out at the table's width w: those above from its top,
// those below from y, under the grid.
func (t *tree) placeCaptions(n *node, w, y float32) {
	top := float32(0)
	for _, c := range n.captions() {
		m := edges(c.st.Margin, w)
		out := t.performChildLayout(c, size{w - m.horizontal(), none}, size{w, none},
			spaces{def(w - m.horizontal()), maxContentSpace}, inherentSize)
		at := &top
		if c.st.CaptionSide == style.CaptionBottom {
			at = &y
		}
		c.unrounded = placed{
			location: point{m.l, *at + m.t},
			size:     out.size,
			overflow: out.overflow,
			padding:  edges(c.st.Padding, w),
			border:   borders(c.st),
			margin:   m,
		}
		*at += out.size.h + m.vertical()
	}
}

// table lays out a table by CSS's automatic table layout, as Chrome does for the plain
// case: columns from their cells' min- and max-content widths, extra width shared, rows
// as tall as their tallest cell, border-spacing around and between the cells.
//
//nolint:funlen // Chrome's automatic table layout in one pass, as its doc walks it
func (t *tree) table(n *node, in input) output {
	st := n.st
	padding := edges(st.Padding, in.parent.w)
	border := borders(st)
	inset := padding.add(border)
	rows := n.rows()
	if fixedLayout(st, in) || slices.ContainsFunc(rows, func(r *node) bool { return t.rowWidths(r).spans }) {
		return t.spanned(n, in, rows)
	}
	sx, sy := st.BorderSpacing[0], st.BorderSpacing[1]
	cols := 0
	for _, r := range rows {
		cols = max(cols, len(r.kids))
	}
	colMin, colMax := make([]float32, cols), make([]float32, cols)
	fixed := make([]bool, cols)
	for _, r := range rows {
		tr := t.rowWidths(r)
		for i := range r.kids {
			colMin[i], colMax[i] = max(colMin[i], tr.lo[i]), max(colMax[i], tr.hi[i])
			fixed[i] = fixed[i] || tr.fixed[i]
		}
	}
	sumMin, sumMax := float32(0), float32(0)
	for i := range cols {
		sumMin += colMin[i]
		sumMax += colMax[i]
	}
	spacingW := sx * float32(cols+1)
	if cols == 0 {
		spacingW = 0
	}
	extra := spacingW + inset.horizontal()

	w := in.known.w
	if !some(w) {
		w = resolve(st.Size.Width, in.parent.w)
		if some(w) && contentBox(st) {
			w += inset.horizontal()
		}
	}
	w = or(w, autoTableWidth(in.avail.w, sumMin, sumMax, extra))
	w = max(w, sumMin+extra, t.captionMin(n))
	if in.run == computeSize && in.axis == axisHorizontal {
		return sizeOutput(size{w, 0})
	}

	widths := columnWidths(colMin, colMax, fixed, sumMin, sumMax, w-extra)
	content := w - inset.horizontal()
	heights := make([]float32, len(rows))
	for ri, r := range rows {
		heights[ri] = t.rowHeight(r, widths, content)
	}
	above, below := t.captionHeights(n, w)
	hgt := inset.vertical() + above + below
	if len(rows) > 0 {
		hgt += sy
	}
	for _, h := range heights {
		hgt += h + sy
	}
	if some(in.known.h) {
		hgt = in.known.h
	} else if sh := resolve(st.Size.Height, in.parent.h); some(sh) {
		hgt = max(hgt, sh)
	}
	s := size{w, hgt}
	if in.run == computeSize {
		return sizeOutput(s)
	}

	rowW := float32(0)
	for _, cw := range widths {
		rowW += cw
	}
	if cols > 1 {
		rowW += sx * float32(cols-1)
	}
	var candidates []oofCandidate
	y := above + inset.t + sy
	ri := 0
	placeRow := func(r *node, origin point) {
		h := heights[ri]
		ri++
		r.unrounded = placed{location: point{inset.l + sx - origin.x, y - origin.y}, size: size{rowW, h}}
		tr := r.row
		if tr.placedAt(widths, content, h) {
			y += h + sy
			return
		}
		from := len(candidates)
		x := float32(0)
		for i, c := range r.kids {
			out := t.performChildLayout(c, size{widths[i], h}, size{content, none},
				spaces{def(widths[i]), def(h)}, inherentSize)
			c.unrounded = placed{
				location: point{x, 0},
				size:     out.size,
				overflow: out.overflow,
				padding:  edges(c.st.Padding, content),
				border:   borders(c.st),
			}
			candidates = append(candidates, translateCandidates(out.oof, point{inset.l + sx + x, y})...)
			x += widths[i] + sx
		}
		tr.place(widths, content, h, len(candidates) == from)
		y += h + sy
	}
	for _, k := range n.kids {
		switch k.role {
		case roleRow:
			placeRow(k, point{})
		case roleGroup:
			top := y
			for _, r := range k.kids {
				placeRow(r, point{inset.l + sx, top})
			}
			bottom := y - sy
			if len(k.kids) == 0 {
				bottom = top
			}
			k.unrounded = placed{location: point{inset.l + sx, top}, size: size{rowW, max(bottom-top, 0)}}
		case roleNone, roleTable, roleCaption:
		}
	}
	t.placeCaptions(n, w, hgt-below)
	n.captionRoom = [2]float32{above, below}
	padded := rect{r: s.w - border.horizontal(), b: s.h - border.vertical()}
	return output{
		size:          s,
		overflow:      finalizeOverflow(padded, s.sub(border.sum()), [2]bool{}, false),
		firstBaseline: none,
		lastBaseline:  none,
		oof:           candidates,
		area:          &oofArea{size: s.sub(border.sum()), offset: point{border.l, border.t}},
	}
}

// tableRow is what a table keeps of a row across its passes, and across layouts while the
// row is kept: its cells' min- and max-content widths, its height at the column widths
// it was measured at, and the column widths and height its cells were last placed at.
type tableRow struct {
	lo, hi []float32
	fixed  []bool
	// colspan and rowspan are each cell's, spans whether any is more than one.
	colspan, rowspan []int
	spans            bool

	tall    bool
	widths  []float32
	content float32
	height  float32

	placed        bool
	placedWidths  []float32
	placedContent float32
	placedHeight  float32
}

// rowWidths is r's table row, its cells measured once.
func (t *tree) rowWidths(r *node) *tableRow {
	if r.row != nil {
		return r.row
	}
	tr := &tableRow{lo: make([]float32, len(r.kids)), hi: make([]float32, len(r.kids)), fixed: make([]bool, len(r.kids)),
		colspan: make([]int, len(r.kids)), rowspan: make([]int, len(r.kids))}
	for i, c := range r.kids {
		tr.colspan[i], tr.rowspan[i] = cellSpan(c, "colspan", 1, 1000), cellSpan(c, "rowspan", 0, 65534)
		tr.spans = tr.spans || tr.colspan[i] != 1 || tr.rowspan[i] != 1
		lo := t.measureChildSize(c, noSize, noSize,
			spaces{minContentSpace, maxContentSpace}, inherentSize, horizontal, [2]bool{})
		hi := t.measureChildSize(c, noSize, noSize, maxContentSpaces, inherentSize, horizontal, [2]bool{})
		if w := resolve(c.st.Size.Width, none); some(w) {
			lo, hi = max(lo, w), max(lo, w)
			tr.fixed[i] = true
		}
		tr.lo[i], tr.hi[i] = lo, hi
	}
	r.row = tr
	return tr
}

// rowHeight is r's height with its cells at widths in a table of content width, measured
// again only when those changed.
func (t *tree) rowHeight(r *node, widths []float32, content float32) float32 {
	tr := t.rowWidths(r)
	if tr.tall && tr.content == content && slices.Equal(tr.widths, widths) {
		return tr.height
	}
	h := or(resolve(r.st.Size.Height, none), 0)
	for i, c := range r.kids {
		h = max(h,
			t.measureChildSize(c, size{widths[i], none}, size{content, none},
				spaces{def(widths[i]), maxContentSpace}, inherentSize, vertical, [2]bool{}))
	}
	tr.tall, tr.widths, tr.content, tr.height = true, append(tr.widths[:0], widths...), content, h
	return h
}

// placedAt reports whether the row's cells stand placed at widths and height already,
// with nothing out of flow among them.
func (tr *tableRow) placedAt(widths []float32, content, h float32) bool {
	return tr.placed && tr.placedContent == content && tr.placedHeight == h && slices.Equal(tr.placedWidths, widths)
}

// place notes the widths and height the row's cells were placed at, worth keeping when
// none of them surfaced anything out of flow.
func (tr *tableRow) place(widths []float32, content, h float32, keep bool) {
	tr.placed, tr.placedContent, tr.placedHeight = keep, content, h
	tr.placedWidths = append(tr.placedWidths[:0], widths...)
}

// columnWidths shares a table's content width out: each column its max-content and the
// rest in proportion to it when there is room, to the columns without a width of their
// own (fixed) while there are any, the space above the min-contents in
// proportion to what each can still grow when there isn't, the min-contents at least.
func columnWidths(colMin, colMax []float32, fixed []bool, sumMin, sumMax, content float32) []float32 {
	widths := make([]float32, len(colMin))
	switch {
	case content >= sumMax:
		grows := func(int) bool { return true }
		if slices.Contains(fixed, true) && slices.Contains(fixed, false) {
			grows = func(i int) bool { return !fixed[i] }
		}
		n, growMax := 0, float32(0)
		for i := range widths {
			if grows(i) {
				n++
				growMax += colMax[i]
			}
		}
		for i := range widths {
			widths[i] = colMax[i]
			switch {
			case !grows(i):
			case growMax > 0:
				widths[i] += (content - sumMax) * colMax[i] / growMax
			default:
				widths[i] += (content - sumMax) / float32(n)
			}
		}
	case content <= sumMin || sumMax == sumMin:
		copy(widths, colMin)
	default:
		f := (content - sumMin) / (sumMax - sumMin)
		for i := range widths {
			widths[i] = colMin[i] + (colMax[i]-colMin[i])*f
		}
	}
	return widths
}

// cellSpan is a cell's colspan or rowspan attribute, 1 when missing or not a number,
// held to [lo, hi].
func cellSpan(c *node, name string, lo, hi int) int {
	v, ok := c.box.Node.Attr(name)
	if !ok {
		return 1
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return 1
	}
	return min(max(n, lo), hi)
}

// fixedLayout reports a table laid out by table-layout: fixed, which needs a width of
// its own; without one it is laid out automatically.
func fixedLayout(st *style.Computed, in input) bool {
	return st.TableLayout == style.TableFixed && (some(in.known.w) || st.Size.Width.Kind != style.Auto)
}

// slot is a cell where the grid puts it: its row and column, and the rows and columns it
// spans.
type slot struct {
	cell             *node
	row, col, rs, cs int
	ci               int
}

// slots places each cell in the grid, past the slots cells above already take, its
// rowspan held to the end of its row group (0 spanning to there); cols is the grid's
// width.
func (t *tree) slots(n *node, rows []*node) ([]slot, int) {
	var cells []slot
	var cols int
	end := make([]int, len(rows))
	i := 0
	for _, k := range n.kids {
		switch k.role {
		case roleRow:
			end[i] = i + 1
			i++
		case roleGroup:
			last := i + len(k.kids)
			for ; i < last; i++ {
				end[i] = last
			}
		case roleNone, roleTable, roleCaption:
		}
	}
	var busy []int
	for ri, r := range rows {
		tr := t.rowWidths(r)
		col := 0
		for ci, c := range r.kids {
			for col < len(busy) && busy[col] > 0 {
				col++
			}
			cs, rs := tr.colspan[ci], tr.rowspan[ci]
			if rs == 0 || ri+rs > end[ri] {
				rs = end[ri] - ri
			}
			for len(busy) < col+cs {
				busy = append(busy, 0)
			}
			for k := col; k < col+cs; k++ {
				busy[k] = rs
			}
			cells = append(cells, slot{cell: c, row: ri, col: col, rs: rs, cs: cs, ci: ci})
			col += cs
		}
		cols = max(cols, len(busy))
		for k := range busy {
			busy[k] = max(busy[k]-1, 0)
		}
	}
	return cells, cols
}

// spanned lays out a table whose cells span rows or columns, or whose layout is fixed:
// each cell in the slots the grid gives it. Automatically, the columns take their single
// cells' min- and max-content widths, then each spanning cell, narrowest spans first,
// shares out what it needs beyond its columns' in proportion to their max-content
// widths, evenly where those are none. Fixed, the first row's cells with a width size
// their columns and the others share the rest. A row is as tall as its single-row cells,
// and a cell spanning rows grows them to fit it in proportion to their heights, as
// Chrome does.
//
//nolint:funlen // Chrome's table layout with spans in one pass, as its doc walks it
func (t *tree) spanned(n *node, in input, rows []*node) output {
	st := n.st
	padding := edges(st.Padding, in.parent.w)
	border := borders(st)
	inset := padding.add(border)
	sx, sy := st.BorderSpacing[0], st.BorderSpacing[1]
	cells, cols := t.slots(n, rows)
	spacingW := float32(0)
	if cols > 0 {
		spacingW = sx * float32(cols+1)
	}
	extra := spacingW + inset.horizontal()
	inner := func(cs int) float32 { return sx * float32(cs-1) }

	w := in.known.w
	if !some(w) {
		w = resolve(st.Size.Width, in.parent.w)
		if some(w) && contentBox(st) {
			w += inset.horizontal()
		}
	}
	colMin, colMax := make([]float32, cols), make([]float32, cols)
	fixed := make([]bool, cols)
	fixedW := slices.Repeat([]float32{-1}, cols)
	if fixedLayout(st, in) {
		if len(rows) > 0 {
			firstRowWidths(cells, fixedW, w-extra, inner)
		}
		for k, fw := range fixedW {
			colMin[k], colMax[k] = max(fw, 0), max(fw, 0)
		}
	} else {
		spannedColumns(cells, colMin, colMax, fixed, inner)
	}
	sumMin, sumMax := float32(0), float32(0)
	for k := range cols {
		sumMin += colMin[k]
		sumMax += colMax[k]
	}
	w = or(w, autoTableWidth(in.avail.w, sumMin, sumMax, extra))
	w = max(w, sumMin+extra, t.captionMin(n))
	if in.run == computeSize && in.axis == axisHorizontal {
		return sizeOutput(size{w, 0})
	}

	var widths []float32
	if fixedLayout(st, in) {
		widths = fixedWidths(fixedW, w-extra)
	} else {
		widths = columnWidths(colMin, colMax, fixed, sumMin, sumMax, w-extra)
	}
	colX := make([]float32, cols+1)
	for k := range cols {
		colX[k+1] = colX[k] + widths[k] + sx
	}
	cellW := func(c slot) float32 { return colX[c.col+c.cs] - colX[c.col] - sx }
	content := w - inset.horizontal()
	heights := make([]float32, len(rows))
	for ri, r := range rows {
		heights[ri] = or(resolve(r.st.Size.Height, none), 0)
	}
	measure := func(c slot) float32 {
		cw := cellW(c)
		return t.measureChildSize(c.cell, size{cw, none}, size{content, none},
			spaces{def(cw), maxContentSpace}, inherentSize, vertical, [2]bool{})
	}
	for _, c := range cells {
		if c.rs == 1 {
			heights[c.row] = max(heights[c.row], measure(c))
		}
	}
	tall := slices.DeleteFunc(slices.Clone(cells), func(c slot) bool { return c.rs == 1 })
	slices.SortStableFunc(tall, func(a, b slot) int { return cmp.Compare(a.rs, b.rs) })
	for _, c := range tall {
		rows := heights[c.row : c.row+c.rs]
		share(rows, slices.Clone(rows), measure(c)-sy*float32(c.rs-1))
	}
	above, below := t.captionHeights(n, w)
	hgt := inset.vertical() + above + below
	if len(rows) > 0 {
		hgt += sy
	}
	for _, h := range heights {
		hgt += h + sy
	}
	if some(in.known.h) {
		hgt = in.known.h
	} else if sh := resolve(st.Size.Height, in.parent.h); some(sh) {
		hgt = max(hgt, sh)
	}
	s := size{w, hgt}
	if in.run == computeSize {
		return sizeOutput(s)
	}

	rowW := max(colX[cols]-sx, 0)
	rowY := make([]float32, len(rows)+1)
	rowY[0] = above + inset.t + sy
	for ri, h := range heights {
		rowY[ri+1] = rowY[ri] + h + sy
	}
	at := 0
	place := func(r *node, origin point) {
		r.unrounded = placed{location: point{inset.l + sx - origin.x, rowY[at] - origin.y}, size: size{rowW, heights[at]}}
		at++
	}
	for _, k := range n.kids {
		switch k.role {
		case roleRow:
			place(k, point{})
		case roleGroup:
			top := rowY[at]
			for _, r := range k.kids {
				place(r, point{inset.l + sx, top})
			}
			bottom := rowY[at] - sy
			if len(k.kids) == 0 {
				bottom = top
			}
			k.unrounded = placed{location: point{inset.l + sx, top}, size: size{rowW, max(bottom-top, 0)}}
		case roleNone, roleTable, roleCaption:
		}
	}
	var candidates []oofCandidate
	for _, c := range cells {
		cw, ch := cellW(c), rowY[c.row+c.rs]-rowY[c.row]-sy
		out := t.performChildLayout(c.cell, size{cw, ch}, size{content, none},
			spaces{def(cw), def(ch)}, inherentSize)
		c.cell.unrounded = placed{
			location: point{colX[c.col], 0},
			size:     out.size,
			overflow: out.overflow,
			padding:  edges(c.cell.st.Padding, content),
			border:   borders(c.cell.st),
		}
		candidates = append(candidates, translateCandidates(out.oof, point{inset.l + sx + colX[c.col], rowY[c.row]})...)
	}
	t.placeCaptions(n, w, hgt-below)
	n.captionRoom = [2]float32{above, below}
	padded := rect{r: s.w - border.horizontal(), b: s.h - border.vertical()}
	return output{
		size:          s,
		overflow:      finalizeOverflow(padded, s.sub(border.sum()), [2]bool{}, false),
		firstBaseline: none,
		lastBaseline:  none,
		oof:           candidates,
		area:          &oofArea{size: s.sub(border.sum()), offset: point{border.l, border.t}},
	}
}

// autoTableWidth is a table's width with none given: its columns' min-content or
// max-content widths and extra, or the room there is between the two.
func autoTableWidth(a space, sumMin, sumMax, extra float32) float32 {
	switch a.kind {
	case minContent:
		return sumMin + extra
	case maxContent:
		return sumMax + extra
	case definite:
	}
	return min(max(a.v, sumMin+extra), sumMax+extra)
}

// firstRowWidths sets fixedW, by fixed table layout, from the first row's cells with a
// width, a spanning one's shared evenly between its columns.
func firstRowWidths(cells []slot, fixedW []float32, content float32, inner func(int) float32) {
	for _, c := range cells {
		if c.row != 0 {
			break
		}
		if cw := resolve(c.cell.st.Size.Width, content); some(cw) {
			for k := c.col; k < c.col+c.cs; k++ {
				fixedW[k] = (cw - inner(c.cs)) / float32(c.cs)
			}
		}
	}
}

// spannedColumns sets each column's min- and max-content widths, by automatic layout:
// from its single-column cells, then from the spanning ones, narrowest span first.
func spannedColumns(cells []slot, colMin, colMax []float32, fixed []bool, inner func(int) float32) {
	for _, c := range cells {
		if c.cs == 1 {
			tr := c.cell.parent.row
			colMin[c.col], colMax[c.col] = max(colMin[c.col], tr.lo[c.ci]), max(colMax[c.col], tr.hi[c.ci])
			fixed[c.col] = fixed[c.col] || tr.fixed[c.ci]
		}
	}
	wide := slices.DeleteFunc(slices.Clone(cells), func(c slot) bool { return c.cs == 1 })
	slices.SortStableFunc(wide, func(a, b slot) int { return cmp.Compare(a.cs, b.cs) })
	for _, c := range wide {
		tr := c.cell.parent.row
		share(colMin[c.col:c.col+c.cs], colMax[c.col:c.col+c.cs], tr.lo[c.ci]-inner(c.cs))
		share(colMax[c.col:c.col+c.cs], colMax[c.col:c.col+c.cs], tr.hi[c.ci]-inner(c.cs))
		for k := c.col; k < c.col+c.cs; k++ {
			colMax[k] = max(colMax[k], colMin[k])
		}
	}
}

// share grows cols so they hold need between them, the shortfall shared in proportion
// to weights, evenly where those are all none.
func share(cols, weights []float32, need float32) {
	have, total := float32(0), float32(0)
	for k := range cols {
		have += cols[k]
		total += weights[k]
	}
	if need <= have {
		return
	}
	extra := need - have
	for k := range cols {
		if total > 0 {
			cols[k] += extra * weights[k] / total
		} else {
			cols[k] += extra / float32(len(cols))
		}
	}
}

// fixedWidths are a fixed table's columns in its content width: those its first row
// sized at their width, the others sharing what is left evenly; with none left over,
// the sized ones grow in proportion to fill it.
func fixedWidths(fixedW []float32, content float32) []float32 {
	widths := make([]float32, len(fixedW))
	sized, free := float32(0), 0
	for _, fw := range fixedW {
		if fw >= 0 {
			sized += fw
		} else {
			free++
		}
	}
	for k, fw := range fixedW {
		switch {
		case fw < 0:
			widths[k] = max(content-sized, 0) / float32(free)
		case free == 0 && sized > 0 && content > sized:
			widths[k] = fw * content / sized
		default:
			widths[k] = fw
		}
	}
	return widths
}
