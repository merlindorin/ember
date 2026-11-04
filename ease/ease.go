// Package ease shapes motion over time: easing functions that bend a
// progress from 0 to 1, and tweens, entities that drive a value with one
// over a duration.
//
//	y := ease.OutBounce(t)
//	slide := ease.NewTween(1.5, func(t float32) { box.SetPosition(from.Lerp(to, t)) },
//		ease.Easing(ease.OutBack), ease.PingPong())
package ease

import "math"

// Func bends progress t, from 0 to 1, into an eased one: 0 at 0 and 1 at
// 1, overshooting in between for Back, Elastic and Bounce.
type Func func(t float32) float32

const (
	back    = 1.70158
	backIO  = back * 1.525
	elastic = 2 * math.Pi / 3
	elastIO = 2 * math.Pi / 4.5
)

func Linear(t float32) float32 { return t }

func InQuad(t float32) float32    { return t * t }
func OutQuad(t float32) float32   { return 1 - (1-t)*(1-t) }
func InOutQuad(t float32) float32 { return inOut(InQuad, t) }

func InCubic(t float32) float32    { return t * t * t }
func OutCubic(t float32) float32   { return out(InCubic, t) }
func InOutCubic(t float32) float32 { return inOut(InCubic, t) }

func InSine(t float32) float32    { return 1 - cos(t*math.Pi/2) }
func OutSine(t float32) float32   { return sin(t * math.Pi / 2) }
func InOutSine(t float32) float32 { return -(cos(math.Pi*t) - 1) / 2 }

func InExpo(t float32) float32    { return inExpo(t) }
func OutExpo(t float32) float32   { return out(inExpo, t) }
func InOutExpo(t float32) float32 { return inOut(inExpo, t) }

func InBack(t float32) float32  { return (back+1)*t*t*t - back*t*t }
func OutBack(t float32) float32 { return out(InBack, t) }
func InOutBack(t float32) float32 {
	if t < 0.5 {
		return (4 * t * t * ((backIO+1)*2*t - backIO)) / 2
	}
	u := 2*t - 2
	return (u*u*((backIO+1)*u+backIO) + 2) / 2
}

func InElastic(t float32) float32  { return inElastic(t) }
func OutElastic(t float32) float32 { return out(inElastic, t) }
func InOutElastic(t float32) float32 {
	switch {
	case t <= 0:
		return 0
	case t >= 1:
		return 1
	case t < 0.5:
		return -(exp2(20*t-10) * sin((20*t-11.125)*elastIO)) / 2
	}
	return exp2(-20*t+10)*sin((20*t-11.125)*elastIO)/2 + 1
}

func InBounce(t float32) float32    { return out(outBounce, t) }
func OutBounce(t float32) float32   { return outBounce(t) }
func InOutBounce(t float32) float32 { return inOut(InBounce, t) }

func inExpo(t float32) float32 {
	if t <= 0 {
		return 0
	}
	return exp2(10*t - 10)
}

func inElastic(t float32) float32 {
	switch {
	case t <= 0:
		return 0
	case t >= 1:
		return 1
	}
	return -exp2(10*t-10) * sin((10*t-10.75)*elastic)
}

func outBounce(t float32) float32 {
	const n, d = 7.5625, 2.75
	switch {
	case t < 1/d:
		return n * t * t
	case t < 2/d:
		t -= 1.5 / d
		return n*t*t + 0.75
	case t < 2.5/d:
		t -= 2.25 / d
		return n*t*t + 0.9375
	}
	t -= 2.625 / d
	return n*t*t + 0.984375
}

// out is an In easing mirrored into its Out at t: fast at first, slow at
// the end.
func out(in Func, t float32) float32 { return 1 - in(1-t) }

// inOut runs an In easing over the first half and its Out over the second.
func inOut(in Func, t float32) float32 {
	if t < 0.5 {
		return in(2*t) / 2
	}
	return 1 - in(2-2*t)/2
}

func sin(x float32) float32  { return float32(math.Sin(float64(x))) }
func cos(x float32) float32  { return float32(math.Cos(float64(x))) }
func exp2(y float32) float32 { return float32(math.Exp2(float64(y))) }
