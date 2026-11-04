package geom

import "math"

func Sqrt(v float32) float32 { return float32(math.Sqrt(float64(v))) }
func Sin(v float32) float32  { return float32(math.Sin(float64(v))) }
func Cos(v float32) float32  { return float32(math.Cos(float64(v))) }
func Tan(v float32) float32  { return float32(math.Tan(float64(v))) }
func Atan2(y, x float32) float32 {
	return float32(math.Atan2(float64(y), float64(x)))
}
func Acos(v float32) float32   { return float32(math.Acos(float64(Clamp(v, -1, 1)))) }
func Pow(b, e float32) float32 { return float32(math.Pow(float64(b), float64(e))) }
func Floor(v float32) float32  { return float32(math.Floor(float64(v))) }
func Exp(v float32) float32    { return float32(math.Exp(float64(v))) }

func Abs(v float32) float32 {
	if v < 0 {
		return -v
	}
	return v
}

func Clamp(v, lo, hi float32) float32 { return max(lo, min(hi, v)) }

func Clamp01(v float32) float32 { return Clamp(v, 0, 1) }

func Lerp(a, b, t float32) float32 { return a + (b-a)*t }

// InvLerp returns where v sits between a and b, unclamped.
func InvLerp(a, b, v float32) float32 {
	if a == b {
		return 0
	}
	return (v - a) / (b - a)
}

func SmoothStep(e0, e1, x float32) float32 {
	t := Clamp01((x - e0) / (e1 - e0))
	return t * t * (3 - 2*t)
}

// Damp is a frame-rate independent exponential approach of a toward b.
// lambda controls the speed: larger values converge faster.
func Damp(a, b, lambda, dt float32) float32 {
	return Lerp(a, b, 1-Exp(-lambda*dt))
}

func DampVec3(a, b Vec3, lambda, dt float32) Vec3 {
	return a.Lerp(b, 1-Exp(-lambda*dt))
}

// WrapAngle maps an angle into [-Pi, Pi).
func WrapAngle(a float32) float32 {
	a = float32(math.Mod(float64(a+Pi), float64(Tau)))
	if a < 0 {
		a += Tau
	}
	return a - Pi
}

// DampAngle smoothly rotates angle a toward b along the shortest arc.
func DampAngle(a, b, lambda, dt float32) float32 {
	return a + WrapAngle(b-a)*(1-Exp(-lambda*dt))
}
