package layout

import "math"

// cache is Taffy's per-node layout cache: one final layout, and nine measured sizes reused
// second-chance style.
const cacheSize = 9

const (
	infinityBits    = uint32(0x7f800000)
	negInfinityBits = uint32(0xff800000)
	signBit1        = uint64(1) << 63
	signBit2        = uint64(1) << 31
	bothSignBits    = signBit1 | signBit2
	nonSignBits     = ^bothSignBits
	xAxisValueMask  = uint64(math.MaxUint32) << 32
)

func optionKey(v float32) uint32 {
	if some(v) {
		return math.Float32bits(v)
	}
	return infinityBits
}

func spaceKey(s space) uint32 {
	switch s.kind {
	case minContent:
		return negInfinityBits
	case maxContent:
		return infinityBits
	case definite:
	}
	return math.Float32bits(-s.v)
}

func mixedKey(known float32, s space) uint32 {
	if some(known) {
		return math.Float32bits(known)
	}
	return spaceKey(s)
}

type cacheKey struct {
	knownAvail      uint64
	parent          uint64
	knownIsDefinite bools
	mode            cacheMode
}

func keyOf(in input) cacheKey {
	var extra uint64
	switch in.axis {
	case axisHorizontal:
		extra = signBit1
	case axisVertical:
		extra = signBit2
	case axisBoth:
		extra = bothSignBits
	}
	parent := uint64(optionKey(in.parent.w))<<32 | uint64(optionKey(in.parent.h))
	return cacheKey{
		knownAvail: uint64(mixedKey(in.known.w, in.avail.w))<<32 | uint64(mixedKey(in.known.h, in.avail.h)),
		parent:     parent&nonSignBits | extra,
		knownIsDefinite: bools{
			w: in.knownIsDefinite.w || !some(in.known.w),
			h: in.knownIsDefinite.h || !some(in.known.h),
		},
		mode: modeOf(in),
	}
}

func (k cacheKey) xParent() uint64       { return k.parent & (xAxisValueMask & nonSignBits) }
func (k cacheKey) requestedBits() uint64 { return k.parent & bothSignBits }

func (k cacheKey) sizeValidFor(o cacheKey) bool {
	a := k.requestedBits()
	return a == bothSignBits || a == o.requestedBits()
}

type cacheMode uint8

func modeOf(in input) cacheMode {
	var m cacheMode
	if in.sizing == inherentSize {
		m |= 1
	}
	if in.collapsible[0] {
		m |= 2
	}
	if in.collapsible[1] {
		m |= 4
	}
	return m
}

type cacheEntry struct {
	ok   bool
	key  cacheKey
	size size
}

// cache keeps entries of every sizing and margin-collapsing mode side by side, keyed by
// it, so a flex item measured as content and then laid out as itself keeps both.
type cache struct {
	final    *output
	finalKey cacheKey
	// probe is the last baseline probe's layout. A probe that runs again
	// rewrites the node's descendants, so it drops final, which then runs
	// after it and writes them back.
	probe    *output
	probeKey cacheKey
	measures [cacheSize]cacheEntry
	recent   uint16
	next     uint8
	empty    bool
}

func (c *cache) get(in input) (output, bool) {
	key := keyOf(in)
	switch {
	case in.run == performLayout && in.probe:
		if c.probe != nil && c.probeKey == key {
			return *c.probe, true
		}
	case in.run == performLayout:
		if c.final != nil && c.finalKey == key {
			return *c.final, true
		}
	case in.run == computeSize:
		for i, e := range c.measures {
			if !e.ok {
				continue
			}
			if e.key.knownAvail == key.knownAvail && e.key.knownIsDefinite == key.knownIsDefinite && e.key.mode == key.mode &&
				e.key.xParent() == key.xParent() && e.key.sizeValidFor(key) {
				c.recent |= 1 << i
				return sizeOutput(e.size), true
			}
		}
	}
	return output{}, false
}

func (c *cache) store(in input, out output) {
	if in.run == performHiddenLayout {
		return
	}
	key := keyOf(in)
	switch {
	case in.run == performLayout && in.probe:
		c.empty = false
		o := out
		c.probe, c.probeKey, c.final = &o, key, nil
	case in.run == performLayout:
		c.empty = false
		o := out
		c.final, c.finalKey = &o, key
	case in.run == computeSize:
		if out.collapsesThrough || out.topMargin != (marginSet{}) || out.bottomMargin != (marginSet{}) {
			return
		}
		c.empty = false
		for i := range c.measures {
			if c.measures[i].ok && c.measures[i].key == key {
				c.measures[i].size = out.size
				c.recent |= 1 << i
				return
			}
		}
		for c.recent&(1<<c.next) != 0 {
			c.recent &^= 1 << c.next
			c.next = (c.next + 1) % cacheSize
		}
		c.measures[c.next] = cacheEntry{ok: true, key: key, size: out.size}
		c.recent |= 1 << c.next
		c.next = (c.next + 1) % cacheSize
	}
}

func (c *cache) clear() { *c = cache{empty: true} }
