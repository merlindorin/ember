// Package geom provides float32 linear algebra tailored for real-time 3D:
// vectors, quaternions, column-major 4x4 matrices, rays and bounding boxes.
// All types are plain values; every operation returns a new value.
package geom

import "math"

const (
	Pi      = float32(math.Pi)
	Tau     = 2 * Pi
	Deg2Rad = Pi / 180
	Rad2Deg = 180 / Pi
	Epsilon = 1e-6
)

type Vec2 struct {
	X float32 `json:"X"`
	Y float32 `json:"Y"`
}

type Vec3 struct {
	X float32 `json:"X"`
	Y float32 `json:"Y"`
	Z float32 `json:"Z"`
}

type Vec4 struct {
	X float32 `json:"X"`
	Y float32 `json:"Y"`
	Z float32 `json:"Z"`
	W float32 `json:"W"`
}

func V2(x, y float32) Vec2       { return Vec2{x, y} }
func V3(x, y, z float32) Vec3    { return Vec3{x, y, z} }
func V4(x, y, z, w float32) Vec4 { return Vec4{x, y, z, w} }

//nolint:gochecknoglobals // exported direction values, part of the API (Vec3 cannot be a const)
var (
	Zero3    = Vec3{}
	One3     = Vec3{1, 1, 1}
	Up       = Vec3{0, 1, 0}
	Down     = Vec3{0, -1, 0}
	Right    = Vec3{1, 0, 0}
	Left     = Vec3{-1, 0, 0}
	Forward  = Vec3{0, 0, -1}
	Backward = Vec3{0, 0, 1}
)

func (a Vec2) Add(b Vec2) Vec2      { return Vec2{a.X + b.X, a.Y + b.Y} }
func (a Vec2) Sub(b Vec2) Vec2      { return Vec2{a.X - b.X, a.Y - b.Y} }
func (a Vec2) Scale(s float32) Vec2 { return Vec2{a.X * s, a.Y * s} }
func (a Vec2) Mul(b Vec2) Vec2      { return Vec2{a.X * b.X, a.Y * b.Y} }
func (a Vec2) Dot(b Vec2) float32   { return a.X*b.X + a.Y*b.Y }
func (a Vec2) Len() float32         { return Sqrt(a.Dot(a)) }
func (a Vec2) LenSq() float32       { return a.Dot(a) }
func (a Vec2) Dist(b Vec2) float32  { return a.Sub(b).Len() }

// Angle is the direction of a in radians, from +X towards +Y, in (-π, π].
func (a Vec2) Angle() float32 { return Atan2(a.Y, a.X) }

// FromAngle is the unit vector at angle radians from +X towards +Y.
func FromAngle(angle float32) Vec2 { return Vec2{Cos(angle), Sin(angle)} }
func (a Vec2) Lerp(b Vec2, t float32) Vec2 {
	return Vec2{Lerp(a.X, b.X, t), Lerp(a.Y, b.Y, t)}
}
func (a Vec2) Normalize() Vec2 {
	l := a.Len()
	if l < Epsilon {
		return Vec2{}
	}
	return a.Scale(1 / l)
}

func (a Vec3) Add(b Vec3) Vec3      { return Vec3{a.X + b.X, a.Y + b.Y, a.Z + b.Z} }
func (a Vec3) Sub(b Vec3) Vec3      { return Vec3{a.X - b.X, a.Y - b.Y, a.Z - b.Z} }
func (a Vec3) Scale(s float32) Vec3 { return Vec3{a.X * s, a.Y * s, a.Z * s} }
func (a Vec3) Mul(b Vec3) Vec3      { return Vec3{a.X * b.X, a.Y * b.Y, a.Z * b.Z} }
func (a Vec3) Neg() Vec3            { return Vec3{-a.X, -a.Y, -a.Z} }
func (a Vec3) Dot(b Vec3) float32   { return a.X*b.X + a.Y*b.Y + a.Z*b.Z }
func (a Vec3) Cross(b Vec3) Vec3 {
	return Vec3{a.Y*b.Z - a.Z*b.Y, a.Z*b.X - a.X*b.Z, a.X*b.Y - a.Y*b.X}
}
func (a Vec3) Len() float32        { return Sqrt(a.Dot(a)) }
func (a Vec3) LenSq() float32      { return a.Dot(a) }
func (a Vec3) Dist(b Vec3) float32 { return a.Sub(b).Len() }
func (a Vec3) Normalize() Vec3 {
	l := a.Len()
	if l < Epsilon {
		return Vec3{}
	}
	return a.Scale(1 / l)
}
func (a Vec3) Lerp(b Vec3, t float32) Vec3 {
	return Vec3{Lerp(a.X, b.X, t), Lerp(a.Y, b.Y, t), Lerp(a.Z, b.Z, t)}
}
func (a Vec3) Min(b Vec3) Vec3 { return Vec3{min(a.X, b.X), min(a.Y, b.Y), min(a.Z, b.Z)} }
func (a Vec3) Max(b Vec3) Vec3 { return Vec3{max(a.X, b.X), max(a.Y, b.Y), max(a.Z, b.Z)} }

// XZ projects the vector onto the ground plane.
func (a Vec3) XZ() Vec2 { return Vec2{a.X, a.Z} }

// WithY returns a copy of the vector with its Y component replaced.
func (a Vec3) WithY(y float32) Vec3 { return Vec3{a.X, y, a.Z} }

// ClampLen limits the vector magnitude to maxLen.
func (a Vec3) ClampLen(maxLen float32) Vec3 {
	if l := a.Len(); l > maxLen && l > Epsilon {
		return a.Scale(maxLen / l)
	}
	return a
}

func (a Vec3) ApproxEq(b Vec3, eps float32) bool {
	return Abs(a.X-b.X) <= eps && Abs(a.Y-b.Y) <= eps && Abs(a.Z-b.Z) <= eps
}

func (a Vec3) Vec4(w float32) Vec4 { return Vec4{a.X, a.Y, a.Z, w} }

func (a Vec4) Vec3() Vec3           { return Vec3{a.X, a.Y, a.Z} }
func (a Vec4) Add(b Vec4) Vec4      { return Vec4{a.X + b.X, a.Y + b.Y, a.Z + b.Z, a.W + b.W} }
func (a Vec4) Scale(s float32) Vec4 { return Vec4{a.X * s, a.Y * s, a.Z * s, a.W * s} }
func (a Vec4) Dot(b Vec4) float32   { return a.X*b.X + a.Y*b.Y + a.Z*b.Z + a.W*b.W }

// FromXZ lifts a ground-plane vector to 3D at height y.
func FromXZ(v Vec2, y float32) Vec3 { return Vec3{v.X, y, v.Y} }
