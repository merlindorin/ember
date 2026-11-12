package paint //nolint:testpackage // white-box: builds box trees by hand and reads the painter's own state

import (
	"image"
	"testing"

	"github.com/merlindorin/ember/color"
	"github.com/merlindorin/ember/geom"
	"github.com/merlindorin/ember/twod"
	"github.com/merlindorin/ember/ui/internal/box"
	"github.com/merlindorin/ember/ui/internal/style"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

//nolint:gochecknoglobals // the tests' fixed values; Go has no constant colours or pictures
var names = map[color.Color]string{red: "red", green: "green", blue: "blue", white: "white", black: "black"}

func TestBackgroundsFillTheBorderBoxInTheRootsSpace(t *testing.T) {
	root := el(0, 0, 200, 100, bg(white), el(10, 20, 50, 30, bg(red)))
	r := newRecorder()
	Paint(r, tree(root))
	fills := r.of("rect")
	require.Len(t, fills, 2)
	assert.Equal(t, geom.Rect{W: 200, H: 100}, fills[0].Rect)
	assert.Equal(t, geom.Rect{X: 10, Y: 20, W: 50, H: 30}, fills[1].Rect)
}

func TestATransparentBoxDrawsNothing(t *testing.T) {
	r := newRecorder()
	Paint(r, tree(el(0, 0, 10, 10, nil)))
	assert.Empty(t, r.ops)
}

func TestRadiiRoundTheBackgroundAndTheBorder(t *testing.T) {
	b := el(0, 0, 100, 40, func(s *style.Computed) {
		s.Background = red
		s.Radius = style.Corners[style.Size[style.Length]]{
			TopLeft: style.Size[style.Length]{Width: style.Px(6), Height: style.Px(6)},
		}
		s.Border = style.All[float32](2)
		s.BorderStyle = style.All(style.BorderSolid)
		s.BorderColor = style.All(blue)
	})
	r := newRecorder()
	Paint(r, tree(el(0, 0, 400, 300, nil, b)))
	round := r.of("round4")
	require.Len(t, round, 1)
	assert.Equal(t, [4]float32{6, 0, 0, 0}, round[0].Radii, "each corner its own radius")
	border := r.of("border4")
	require.Len(t, border, 1)
	assert.Equal(t, float32(2), border[0].Width)
	assert.Equal(t, [4]float32{6, 0, 0, 0}, border[0].Radii)
	assert.Equal(t, blue, border[0].Color)
}

func TestOverlappingRadiiScaleDownTogetherAsCSSDoes(t *testing.T) {
	b := el(0, 0, 100, 20, func(s *style.Computed) {
		s.Background = red
		big := style.Size[style.Length]{Width: style.Px(30), Height: style.Px(30)}
		small := style.Size[style.Length]{Width: style.Px(10), Height: style.Px(10)}
		s.Radius = style.Corners[style.Size[style.Length]]{TopLeft: big, TopRight: small, BottomRight: small,
			BottomLeft: small}
	})
	r := newRecorder()
	Paint(r, tree(el(0, 0, 400, 300, nil, b)))
	got := r.of("round4")[0].Radii
	assert.InDelta(t, 20*30/40.0, got[0], 1e-4, "the left side's 30+10 scaled into its 20")
	assert.InDelta(t, 20*10/40.0, got[1], 1e-4)
}

func TestAPercentRadiusIsOfTheBoxAndNeverPastHalfItsSide(t *testing.T) {
	pill := el(0, 0, 100, 20, func(s *style.Computed) {
		s.Background = red
		c := style.Size[style.Length]{Width: style.Percent(50), Height: style.Percent(50)}
		s.Radius = style.Corners[style.Size[style.Length]]{TopLeft: c, TopRight: c, BottomRight: c, BottomLeft: c}
	})
	r := newRecorder()
	Paint(r, tree(el(0, 0, 400, 300, nil, pill)))
	assert.Equal(t, float32(10), r.of("round")[0].Radius)
}

func TestSquareBordersAreOneRectPerSideInTheirOwnColour(t *testing.T) {
	b := el(0, 0, 100, 40, func(s *style.Computed) {
		s.Border = style.Sides[float32]{Top: 1, Right: 2, Bottom: 3, Left: 4}
		s.BorderStyle = style.All(style.BorderSolid)
		s.BorderColor = style.Sides[color.Color]{Top: red, Right: green, Bottom: blue, Left: black}
	})
	r := newRecorder()
	Paint(r, tree(b))
	assert.Equal(t, "red green blue black", r.order(names))
	sides := r.of("rect")
	assert.Equal(t, geom.Rect{W: 100, H: 1}, sides[0].Rect)
	assert.Equal(t, geom.Rect{X: 98, W: 2, H: 40}, sides[1].Rect)
	assert.Equal(t, geom.Rect{Y: 37, W: 100, H: 3}, sides[2].Rect)
	assert.Equal(t, geom.Rect{W: 4, H: 40}, sides[3].Rect)
}

func TestABorderWithStyleNoneDrawsNothing(t *testing.T) {
	b := el(0, 0, 100, 40, func(s *style.Computed) {
		s.Border = style.All[float32](0)
		s.BorderColor = style.All(red)
	})
	r := newRecorder()
	Paint(r, tree(b))
	assert.Empty(t, r.ops)
}

func TestShadowsDrawUnderTheBoxTheFirstOnTop(t *testing.T) {
	b := el(10, 10, 50, 50, func(s *style.Computed) {
		s.Background = white
		s.Shadows = []style.Shadow{
			{X: 2, Y: 4, Blur: 0, Spread: 1, Color: red},
			{X: 0, Y: 0, Blur: 8, Color: blue},
		}
	})
	r := newRecorder()
	Paint(r, tree(el(0, 0, 400, 300, nil, b)))
	var kinds []string
	var redArea float32
	for _, o := range r.ops {
		kinds = append(kinds, o.Kind)
		if o.Kind == "rect" && o.Color == red {
			redArea += o.Rect.W * o.Rect.H
		}
	}
	assert.Equal(t, "gradient", kinds[0], "the blurred blue one, a mesh, under all")
	assert.Equal(t, "white", r.order(names)[len(r.order(names))-5:], "the box last")
	assert.InDelta(t, 52*52-49*47, redArea, 1e-3, "red offset, grown by spread, and what the box leaves of it")
}

func TestAnOutlineDrawsOutsideTheBorderAfterTheContent(t *testing.T) {
	b := el(10, 10, 50, 20, func(s *style.Computed) {
		s.Background = white
		s.OutlineWidth, s.OutlineStyle, s.OutlineColor, s.OutlineOffset = 2, style.BorderSolid, blue, 1
	}, el(0, 0, 10, 10, bg(red)))
	r := newRecorder()
	Paint(r, tree(b))
	last := r.ops[len(r.ops)-1]
	assert.Equal(t, "border", last.Kind)
	assert.Equal(t, geom.Rect{X: 7, Y: 7, W: 56, H: 26}, last.Rect)
	assert.Equal(t, float32(2), last.Width)
}

func TestAnOutlineOfStyleNoneDrawsNothingWhateverItsWidth(t *testing.T) {
	b := el(0, 0, 50, 20, func(s *style.Computed) {
		s.OutlineWidth, s.OutlineStyle, s.OutlineColor = 3, style.BorderNone, blue
	})
	r := newRecorder()
	Paint(r, tree(b))
	assert.Empty(t, r.of("border"), "Chrome computes 3px for outline-style none")
}

func TestOpacityMultipliesTheAlphaOfTheWholeSubtree(t *testing.T) {
	inner := el(0, 0, 10, 10, func(s *style.Computed) { s.Background = red; s.Opacity = 0.5 })
	outer := el(0, 0, 100, 100, func(s *style.Computed) { s.Background = white; s.Opacity = 0.5 }, inner)
	r := newRecorder()
	Paint(r, tree(el(0, 0, 400, 300, nil, outer)))
	fills := r.of("rect")
	assert.InDelta(t, 0.5, fills[0].Color.A, 1e-6)
	assert.InDelta(t, 0.25, fills[1].Color.A, 1e-6)
	assert.Equal(t, float32(1), fills[1].Color.R)
}

func TestZeroOpacityAndHiddenVisibilitySkipDrawing(t *testing.T) {
	gone := el(0, 0, 10, 10, func(s *style.Computed) { s.Background = red; s.Opacity = 0 })
	hidden := el(0, 0, 10, 10, func(s *style.Computed) { s.Background = blue; s.Visible = false },
		el(0, 0, 5, 5, bg(green)))
	r := newRecorder()
	Paint(r, tree(el(0, 0, 100, 100, nil, gone, hidden)))
	assert.Equal(t, "green", r.order(names), "a visible child of a hidden box still shows")
}

func TestTextDrawsEachFragmentInItsColourAndOpacity(t *testing.T) {
	p := el(5, 5, 100, 20, func(s *style.Computed) { s.Color = red; s.Opacity = 0.5 })
	p.Append(text(2, 0, "hello", p.Style))
	r := newRecorder()
	Paint(r, tree(p))
	ts := r.of("text")
	require.Len(t, ts, 1)
	assert.Equal(t, "hello", ts[0].Text)
	assert.Equal(t, geom.V2(7, 5+15), ts[0].At, "on the fragment's baseline")
	assert.True(t, ts[0].Baseline, "placed by layout's baseline, not twod's own ascent")
	assert.InDelta(t, 0.5, ts[0].Color.A, 1e-6)
}

type ahem struct{}

func (ahem) Metrics(f style.Font) box.Metrics {
	return box.Metrics{Ascent: 0.8 * f.Size, Descent: 0.2 * f.Size}
}
func (ahem) Advance(f style.Font, s string) float32 { return float32(len(s)) * f.Size }

func TestTextSitsOnLayoutsBaselineWhateverTheFonts(t *testing.T) {
	p := el(0, 0, 100, 20, nil)
	p.Append(text(0, 0, "x", p.Style))
	r := newRecorder()
	Paint(r, tree(p), Fonts(ahem{}))
	assert.Equal(t, geom.V2(0, 15), r.of("text")[0].At, "the same with or without metrics")
}

func TestStackingPaintsNegativeZThenFlowThenPositionedThenPositiveZ(t *testing.T) {
	pos := func(c color.Color, z int, auto bool) *box.Box {
		return el(0, 0, 10, 10, func(s *style.Computed) {
			s.Background, s.Position, s.ZIndex, s.ZAuto = c, style.Relative, z, auto
		})
	}
	root := el(0, 0, 100, 100, bg(white),
		pos(red, 2, false),
		pos(green, 0, true),
		el(0, 0, 10, 10, bg(black)),
		pos(blue, -1, false),
	)
	r := newRecorder()
	Paint(r, tree(root))
	assert.Equal(t, "white blue black green red", r.order(names))
}

func TestFlowBackgroundsAllComeBeforeFlowText(t *testing.T) {
	a := el(0, 0, 100, 20, bg(red))
	a.Append(text(0, 0, "a", a.Style))
	b := el(0, 10, 100, 20, bg(blue))
	r := newRecorder()
	Paint(r, tree(el(0, 0, 100, 100, nil, a, b)))
	kinds := []string{}
	for _, o := range r.ops {
		kinds = append(kinds, o.Kind)
	}
	assert.Equal(t, []string{"rect", "rect", "text"}, kinds, "b's background never covers a's text")
}

func TestAStackingContextKeepsItsDescendantsTogether(t *testing.T) {
	high := el(0, 0, 10, 10, func(s *style.Computed) {
		s.Background, s.Position, s.ZIndex, s.ZAuto = blue, style.Relative, 100, false
	})
	low := el(0, 0, 50, 50, func(s *style.Computed) {
		s.Background, s.Position, s.ZIndex, s.ZAuto = red, style.Relative, 1, false
	}, high)
	mid := el(0, 0, 50, 50, func(s *style.Computed) {
		s.Background, s.Position, s.ZIndex, s.ZAuto = green, style.Relative, 2, false
	})
	r := newRecorder()
	Paint(r, tree(el(0, 0, 100, 100, nil, low, mid)))
	assert.Equal(t, "red blue green", r.order(names), "z 100 inside z 1 stays under z 2")
}

func TestOverflowClipsChildrenToThePaddingBox(t *testing.T) {
	inner := el(0, 0, 500, 500, bg(red))
	clip := el(10, 10, 100, 50, func(s *style.Computed) {
		s.Overflow = style.Size[style.Overflow]{Width: style.Hidden, Height: style.Hidden}
		s.Border = style.All[float32](2)
		s.BorderStyle = style.All(style.BorderSolid)
		s.BorderColor = style.All(blue)
	}, inner)
	r := newRecorder()
	Paint(r, tree(clip))
	fill := r.of("rect")
	var red0 op
	for _, o := range fill {
		if o.Color == red {
			red0 = o
		}
	}
	assert.Equal(t, geom.Rect{X: 12, Y: 12, W: 96, H: 46}, red0.Clip)
	for _, o := range fill {
		if o.Color == blue {
			assert.Equal(t, geom.Rect{}, o.Clip, "the box's own border is outside its clip")
		}
	}
	assert.Equal(t, len(r.of("push")), len(r.of("pop")))
}

func TestOverflowOnOneAxisClipsOnlyThatAxis(t *testing.T) {
	clip := el(0, 0, 100, 50, func(s *style.Computed) {
		s.Overflow = style.Size[style.Overflow]{Width: style.Clip}
	}, el(0, 0, 500, 500, bg(red)))
	r := newRecorder()
	Paint(r, tree(clip))
	c := r.of("rect")[0].Clip
	assert.Equal(t, float32(0), c.X)
	assert.Equal(t, float32(100), c.W)
	assert.Greater(t, c.H, float32(1e5))
}

func TestPositionedDescendantsStayInTheirScrollersClip(t *testing.T) {
	popup := el(0, 0, 200, 200, func(s *style.Computed) {
		s.Background, s.Position, s.ZIndex, s.ZAuto = red, style.Absolute, 5, false
	})
	clip := el(10, 10, 100, 50, func(s *style.Computed) {
		s.Position = style.Relative
		s.Overflow = style.Size[style.Overflow]{Width: style.Hidden, Height: style.Hidden}
	}, popup)
	r := newRecorder()
	Paint(r, tree(el(0, 0, 300, 300, nil, clip)))
	assert.Equal(t, geom.Rect{X: 10, Y: 10, W: 100, H: 50}, r.of("rect")[0].Clip)
}

func TestScrollingShiftsTheContentButNotTheScroller(t *testing.T) {
	inner := el(0, 0, 100, 300, bg(red))
	sc := el(10, 10, 100, 100, func(s *style.Computed) {
		s.Background = white
		s.Overflow = style.Size[style.Overflow]{Width: style.Hidden, Height: style.OverflowAuto}
	}, inner)
	sc.Content, sc.Scroll = geom.V2(100, 300), geom.V2(0, 40)
	r := newRecorder()
	Paint(r, tree(sc))
	fills := r.of("rect")
	assert.Equal(t, geom.Rect{X: 10, Y: 10, W: 100, H: 100}, fills[0].Rect)
	assert.Equal(t, geom.Rect{X: 10, Y: -30, W: 100, H: 300}, fills[1].Rect)
}

func TestAScrollContainerDrawsUIsThumbOverItsContent(t *testing.T) {
	sc := el(0, 0, 100, 100, func(s *style.Computed) {
		s.Overflow = style.Size[style.Overflow]{Width: style.Hidden, Height: style.OverflowAuto}
	}, el(0, 0, 100, 400, bg(red)))
	sc.Content = geom.V2(100, 400)
	r := newRecorder()
	Paint(r, tree(sc))
	thumbs := r.of("round")
	require.Len(t, thumbs, 1)
	th := thumbs[0]
	assert.Equal(t, thumbColor, th.Color)
	assert.Equal(t, float32(barWidth), th.Rect.W)
	assert.Equal(t, float32(100-barSize+2), th.Rect.X)
	assert.InDelta(t, 100*100.0/400-8, th.Rect.H, 1e-4,
		"the thumb box is the view's share of the bar, drawn 4 in at each end")
	assert.Equal(t, "round", r.ops[len(r.ops)-1].Kind, "after the content")
	assert.Equal(t, geom.Rect{}, th.Clip, "outside the clip of what it scrolls")

	sc.Content = geom.V2(100, 80)
	r = newRecorder()
	Paint(r, tree(sc))
	assert.Empty(t, r.of("round"), "auto: no thumb when it fits")
}

func TestATransformTurnsTheBoxAboutItsOrigin(t *testing.T) {
	b := el(100, 100, 40, 20, func(s *style.Computed) {
		s.Background = red
		s.Transform = []style.TransformOp{{Kind: style.ScaleOp, A: [6]float32{2, 2}}}
	})
	r := newRecorder()
	Paint(r, tree(el(0, 0, 300, 300, nil, b, el(0, 0, 5, 5, bg(blue)))))
	fills := r.of("rect")
	assert.Equal(t, blue, fills[0].Color, "the flow first, the transformed context after")
	assert.Equal(t, geom.Identity2D(), fills[0].Xform)
	xf := fills[1].Xform
	assert.Equal(t, geom.V2(120, 110), xf.Xform(geom.V2(120, 110)), "the centre stays")
	assert.Equal(t, geom.V2(80, 90), xf.Xform(geom.V2(100, 100)), "corners go out twice as far")
	assert.Equal(t, geom.Identity2D(), r.Transform(), "put back after")
}

func TestTransformsComposeInOrderWithPercentTranslation(t *testing.T) {
	b := el(0, 0, 40, 20, func(s *style.Computed) {
		s.Background = red
		s.TransformOrigin = style.Size[style.Length]{Width: style.Px(0), Height: style.Px(0)}
		s.Transform = []style.TransformOp{
			{Kind: style.Translate, X: style.Percent(50), Y: style.Px(3)},
			{Kind: style.Rotate, A: [6]float32{geom.Pi / 2}},
		}
	})
	r := newRecorder()
	Paint(r, tree(el(0, 0, 400, 300, nil, b)))
	got := r.of("rect")[0].Xform.Xform(geom.V2(10, 0))
	assert.InDelta(t, 20, got.X, 1e-4)
	assert.InDelta(t, 13, got.Y, 1e-4)
}

func TestImagesAreDrawnByNameAndBorderImagesAsSlices(t *testing.T) {
	b := el(0, 0, 64, 32, func(s *style.Computed) {
		s.BackgroundImage = style.Image{Source: "bg.png"}
		s.BorderImage = style.Image{Source: "frame.png", Slice: style.All[float32](2)}
		s.Border = style.All[float32](2)
	})
	r := newRecorder()
	Paint(r, tree(b))
	assert.Empty(t, r.of("image", "nine"), "no images known")

	r = newRecorder()
	blank := twod.NewTexture(image.NewRGBA(image.Rect(0, 0, 8, 8)))
	Paint(r, tree(b), Images(func(string) *twod.Texture { return blank }))
	imgs := r.of("image")
	require.Greater(t, len(imgs), 32)
	for _, o := range imgs[:32] {
		assert.Equal(t, geom.Rect{W: 1, H: 1}, o.UV, "the 8×8 image tiles the 64×32 box first")
	}
	area := float32(0)
	for _, o := range imgs[32:] {
		area += o.Rect.W * o.Rect.H
	}
	assert.InDelta(t, 64*32, area, 1e-3, "then the border image's slices cover it once")
	assert.Equal(t, geom.Rect{W: 0.5, H: 0.5}, imgs[32].Rect, "its corner's outer half texel first")
	assert.Equal(t, geom.Rect{X: 0.5 / 8, Y: 0.5 / 8}, imgs[32].UV, "at that texel's centre")
}

func TestAPositionedBoxPaintsOverTheOutlinesOfTheFlowBeforeIt(t *testing.T) {
	list := el(0, 0, 100, 60, func(s *style.Computed) {
		s.OutlineWidth, s.OutlineStyle, s.OutlineColor = 2, style.BorderSolid, blue
	})
	header := el(0, 0, 100, 20, func(s *style.Computed) { s.Background, s.Position = red, style.Sticky })
	r := newRecorder()
	Paint(r, tree(el(0, 0, 200, 100, nil, list, header)))
	var kinds []string
	for _, o := range r.ops {
		kinds = append(kinds, o.Kind)
	}
	assert.Equal(t, []string{"border", "rect"}, kinds,
		"the outline in its context's flow, then the positioned header over it, as Chrome paints them")
}
