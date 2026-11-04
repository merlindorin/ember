package geom

import "math"

// Quat is a unit quaternion representing a rotation.
type Quat struct {
	X float32 `json:"X"`
	Y float32 `json:"Y"`
	Z float32 `json:"Z"`
	W float32 `json:"W"`
}

func QuatIdent() Quat { return Quat{W: 1} }

func QuatAxisAngle(axis Vec3, angle float32) Quat {
	axis = axis.Normalize()
	s, c := Sin(angle/2), Cos(angle/2)
	return Quat{axis.X * s, axis.Y * s, axis.Z * s, c}
}

// QuatEuler builds a rotation applying yaw (Y), then pitch (X), then roll (Z)
// in the object's local frame.
func QuatEuler(pitch, yaw, roll float32) Quat {
	return QuatAxisAngle(Up, yaw).Mul(QuatAxisAngle(Right, pitch)).Mul(QuatAxisAngle(Backward, roll))
}

// QuatYaw is a rotation around the world up axis.
func QuatYaw(yaw float32) Quat { return QuatAxisAngle(Up, yaw) }

// QuatLookRotation returns the rotation that points -Z toward dir.
func QuatLookRotation(dir, up Vec3) Quat {
	return Mat4LookAt(Zero3, dir, up).Inverse().Quat()
}

func (a Quat) Mul(b Quat) Quat {
	return Quat{
		a.W*b.X + a.X*b.W + a.Y*b.Z - a.Z*b.Y,
		a.W*b.Y - a.X*b.Z + a.Y*b.W + a.Z*b.X,
		a.W*b.Z + a.X*b.Y - a.Y*b.X + a.Z*b.W,
		a.W*b.W - a.X*b.X - a.Y*b.Y - a.Z*b.Z,
	}
}

func (a Quat) Conjugate() Quat { return Quat{-a.X, -a.Y, -a.Z, a.W} }

func (a Quat) Dot(b Quat) float32 { return a.X*b.X + a.Y*b.Y + a.Z*b.Z + a.W*b.W }

func (a Quat) Normalize() Quat {
	l := Sqrt(a.Dot(a))
	if l < Epsilon {
		return QuatIdent()
	}
	return Quat{a.X / l, a.Y / l, a.Z / l, a.W / l}
}

// Rotate applies the rotation to v.
func (a Quat) Rotate(v Vec3) Vec3 {
	u := Vec3{a.X, a.Y, a.Z}
	t := u.Cross(v).Scale(2)
	return v.Add(t.Scale(a.W)).Add(u.Cross(t))
}

// Slerp interpolates along the shortest arc.
func (a Quat) Slerp(b Quat, t float32) Quat {
	d := a.Dot(b)
	if d < 0 {
		b = Quat{-b.X, -b.Y, -b.Z, -b.W}
		d = -d
	}
	if d > 0.9995 {
		return Quat{Lerp(a.X, b.X, t), Lerp(a.Y, b.Y, t), Lerp(a.Z, b.Z, t), Lerp(a.W, b.W, t)}.Normalize()
	}
	theta := Acos(d)
	s := Sin(theta)
	wa, wb := Sin((1-t)*theta)/s, Sin(t*theta)/s
	return Quat{a.X*wa + b.X*wb, a.Y*wa + b.Y*wb, a.Z*wa + b.Z*wb, a.W*wa + b.W*wb}
}

func (a Quat) Mat4() Mat4 {
	x, y, z, w := a.X, a.Y, a.Z, a.W
	xx, yy, zz := x*x, y*y, z*z
	xy, xz, yz := x*y, x*z, y*z
	wx, wy, wz := w*x, w*y, w*z
	return Mat4{
		1 - 2*(yy+zz), 2 * (xy + wz), 2 * (xz - wy), 0,
		2 * (xy - wz), 1 - 2*(xx+zz), 2 * (yz + wx), 0,
		2 * (xz + wy), 2 * (yz - wx), 1 - 2*(xx+yy), 0,
		0, 0, 0, 1,
	}
}

func (a Quat) ApproxEq(b Quat, eps float32) bool {
	return Abs(a.Dot(b)) >= 1-eps
}

// Euler returns the pitch, yaw and roll that QuatEuler would combine into
// this rotation.
func (a Quat) Euler() (float32, float32, float32) {
	m := a.Mat4()
	pitch := float32(math.Asin(float64(Clamp(-m[9], -1, 1))))
	yaw := Atan2(m[8], m[10])
	roll := Atan2(m[1], m[5])
	return pitch, yaw, roll
}
