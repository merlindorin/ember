package geom

// Transform2D is a 2D affine transform: the X and Y axes and the
// origin, mapping p to X*p.x + Y*p.y + Origin.
type Transform2D struct {
	X      Vec2 `json:"X"`
	Y      Vec2 `json:"Y"`
	Origin Vec2 `json:"Origin"`
}

func Identity2D() Transform2D { return Transform2D{X: V2(1, 0), Y: V2(0, 1)} }

// Transform2DFrom builds a transform as a 2D node does from its
// position, rotation, skew and scale.
func Transform2DFrom(position Vec2, rotation, skew float32, scale Vec2) Transform2D {
	return Transform2D{
		X:      V2(Cos(rotation)*scale.X, Sin(rotation)*scale.X),
		Y:      V2(-Sin(rotation+skew)*scale.Y, Cos(rotation+skew)*scale.Y),
		Origin: position,
	}
}

func (t Transform2D) Xform(p Vec2) Vec2 { return t.X.Scale(p.X).Add(t.Y.Scale(p.Y)).Add(t.Origin) }

// BasisXform transforms a direction, ignoring the origin.
func (t Transform2D) BasisXform(v Vec2) Vec2 { return t.X.Scale(v.X).Add(t.Y.Scale(v.Y)) }

// Mul returns t applied after o.
func (t Transform2D) Mul(o Transform2D) Transform2D {
	return Transform2D{X: t.BasisXform(o.X), Y: t.BasisXform(o.Y), Origin: t.Xform(o.Origin)}
}

func (t Transform2D) Inverse() Transform2D {
	det := t.X.X*t.Y.Y - t.X.Y*t.Y.X
	if det == 0 {
		return Identity2D()
	}
	inv := Transform2D{X: V2(t.Y.Y/det, -t.X.Y/det), Y: V2(-t.Y.X/det, t.X.X/det)}
	inv.Origin = inv.BasisXform(t.Origin).Scale(-1)
	return inv
}

func (t Transform2D) Rotation() float32 { return Atan2(t.X.Y, t.X.X) }

func (t Transform2D) Scale() Vec2 {
	det := t.X.X*t.Y.Y - t.X.Y*t.Y.X
	sy := t.Y.Len()
	if det < 0 {
		sy = -sy
	}
	return V2(t.X.Len(), sy)
}
