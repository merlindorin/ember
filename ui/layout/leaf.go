package layout

import "github.com/lantervale/ember/ui/internal/style"

// leaf lays out a node without child boxes: empty, or holding inline content, Taffy's
// compute_leaf_layout with text as its measure function.
//
//nolint:cyclop,funlen,gocyclo // Taffy's compute_leaf_layout, kept in its shape
func (t *tree) leaf(n *node, in input) output {
	st := n.st
	padding := edges(st.Padding, in.parent.w)
	border := borders(st)
	pb := padding.add(border)
	pbSum := pb.sum()
	adj := boxSizingAdjustment(st, pbSum)

	var nodeSize, minSize, maxSize size
	if in.sizing == contentSize {
		nodeSize, minSize, maxSize = in.known, noSize, noSize
	} else {
		ratio := aspectRatio(st)
		styleSize := resolveSize(st.Size, in.parent).withAspectRatio(ratio).mAdd(adj)
		minSize = resolveSize(st.MinSize, in.parent).withAspectRatio(ratio).mAdd(adj)
		if !some(ratio) {
			// A natural ratio holds between content boxes, whatever box-sizing.
			ratio = n.naturalRatio()
			styleSize = styleSize.mSub(pbSum).withAspectRatio(ratio).mAdd(pbSum)
			minSize = minSize.mSub(pbSum).withAspectRatio(ratio).mAdd(pbSum)
		}
		maxSize = resolveMax(st.MaxSize, in.parent).mAdd(adj)
		nodeSize = in.known.or(styleSize)
		if n.replaced && !some(nodeSize.w) && !some(nodeSize.h) {
			nodeSize = replacedSize(n.natural.add(pbSum), ratio, minSize, maxSize)
		}
	}

	gutter := n.ownGutter()
	inset := pb
	inset.r += gutter.x
	inset.b += gutter.y

	block := isBlockDisplay(st.Display)
	preventsCollapse := !block || isScrollContainer(st) || outOfFlow(st.Position) ||
		padding.t > 0 || padding.b > 0 || border.t > 0 || border.b > 0 ||
		(some(nodeSize.h) && nodeSize.h > 0) || (some(minSize.h) && minSize.h > 0)

	if in.run == computeSize && preventsCollapse && nodeSize.both() {
		s := nodeSize.mClamp(minSize, maxSize).mMax(pbSum)
		return sizeOutput(s)
	}

	avail := spaces{w: spaceOr(in.known.w, in.avail.w), h: spaceOr(in.known.h, in.avail.h)}
	if some(nodeSize.w) {
		avail.w = def(nodeSize.w)
	}
	if some(nodeSize.h) {
		avail.h = def(nodeSize.h)
	}
	avail.w = avail.w.mapDefinite(func(v float32) float32 { return mClamp(v, minSize.w, maxSize.w) - inset.horizontal() })
	avail.h = avail.h.mapDefinite(func(v float32) float32 { return mClamp(v, minSize.h, maxSize.h) - inset.vertical() })

	known := in.known.mSub(inset.sum())
	if in.run == performLayout {
		known = noSize
	}
	if n.replaced {
		known = in.known.or(nodeSize).mSub(inset.sum())
	}
	measured, first, last := t.measureContent(n, known, avail)
	clamped := in.known.or(nodeSize).or(measured.add(inset.sum())).mClamp(minSize, maxSize)
	s := clamped
	if r := aspectRatio(st); in.sizing == inherentSize && some(r) {
		s.h = fmax(clamped.h, ratioHeight(clamped.w, r))
	}
	s = s.mMax(pbSum)
	if in.run == performLayout && n.control == textareaControl {
		measured, gutter = t.textareaText(n, s, pb, gutter)
		styled := scrollbarGutter(st)
		n.gutter = size{gutter.x - styled.x, gutter.y - styled.y}
	}

	if in.run == performLayout && n.control == inputControl {
		measured.w = max(measured.w, t.inputText(n))
	}
	scroll := isScrollContainer(st)
	end := padding.l + measured.w
	bottom := padding.t + measured.h
	if scroll {
		end += padding.r
		bottom += padding.b
	}
	overflow := finalizeOverflow(rect{r: end, b: bottom},
		s.sub(border.sum()).sub(size{gutter.x, gutter.y}), [2]bool{}, scroll)

	out := output{
		size:             s,
		overflow:         overflow,
		firstBaseline:    none,
		lastBaseline:     none,
		collapsesThrough: !preventsCollapse && s.h == 0 && measured.h == 0,
	}
	if some(first) {
		shift := float32(0)
		if centresContent(n) {
			shift = fmax(s.h-inset.vertical()-measured.h, 0) / 2
		}
		out.firstBaseline = pb.t + shift + first
		out.lastBaseline = pb.t + shift + last
	}
	if in.run == performLayout && n.inline != nil {
		t.placeText(n, s.sub(size{gutter.x, gutter.y}), pb)
	}
	if in.run == performLayout && len(n.kids) > 0 {
		t.leafChildren(n, &out, pb, border, gutter)
	}
	return out
}

// leafChildren places what an inline formatting context holds besides its text: hidden
// boxes, and positioned ones: from their spot in the line when a span holds them, else
// from the content's start, waiting for their containing block unless a positioned span
// is it.
func (t *tree) leafChildren(n *node, out *output, pb, border rect, gutter point) {
	for i, k := range n.kids {
		if k.atomic {
			out.oof = append(out.oof, translateCandidates(k.pendingOOF, k.unrounded.location)...)
			continue
		}
		if k.st.Display == style.DisplayNone {
			k.unrounded = placed{order: uint32(i)}
			t.performChildLayout(k, noSize, noSize, maxContentSpaces, inherentSize)
			continue
		}
		c := oofCandidate{
			node:     k,
			order:    uint32(i),
			position: k.st.Position,
			x:        staticFromStart(pb.l),
			y:        staticFromStart(pb.t),
		}
		if o := n.inline.oofOf(k); o != nil {
			c.x, c.y = staticFromStart(o.static.x), staticFromStart(o.static.y)
			if o.positioned && k.st.Position == style.Absolute {
				c.area = &oofArea{size: o.areaSize, offset: o.areaAt}
			}
		}
		out.oof = append(out.oof, c)
	}
	inset := border.add(rect{r: gutter.x, b: gutter.y})
	out.area = &oofArea{size: out.size.sub(inset.sum()), offset: point{inset.l, inset.t}}
}

func ratioHeight(w, ratio float32) float32 {
	if some(ratio) {
		return w / ratio
	}
	return 0
}

// measureContent is the size of a leaf's content: its text, or nothing.
func (t *tree) measureContent(n *node, known size, avail spaces) (size, float32, float32) {
	if n.replaced {
		return measureReplaced(n, known), none, none
	}
	if n.inline == nil {
		return known.or(size{}), none, none
	}
	if n.control != noControl {
		return t.measureControl(n, known)
	}
	s, first, last := t.measure(n.inline, known, avail, size{avail.w.value(), none})
	return size{or(known.w, s.w), or(known.h, s.h)}, first, last
}
