package geom

// Mat4 is a column-major 4x4 matrix, directly uploadable to the GPU.
// Element (row r, column c) lives at index c*4+r.
type Mat4 [16]float32

func Ident() Mat4 {
	return Mat4{1, 0, 0, 0, 0, 1, 0, 0, 0, 0, 1, 0, 0, 0, 0, 1}
}

func (m Mat4) At(row, col int) float32 { return m[col*4+row] }

func (m Mat4) Mul(b Mat4) Mat4 {
	var r Mat4
	for c := 0; c < 4; c++ {
		for row := 0; row < 4; row++ {
			r[c*4+row] = m[row]*b[c*4] + m[4+row]*b[c*4+1] + m[8+row]*b[c*4+2] + m[12+row]*b[c*4+3]
		}
	}
	return r
}

func (m Mat4) MulVec4(v Vec4) Vec4 {
	return Vec4{
		m[0]*v.X + m[4]*v.Y + m[8]*v.Z + m[12]*v.W,
		m[1]*v.X + m[5]*v.Y + m[9]*v.Z + m[13]*v.W,
		m[2]*v.X + m[6]*v.Y + m[10]*v.Z + m[14]*v.W,
		m[3]*v.X + m[7]*v.Y + m[11]*v.Z + m[15]*v.W,
	}
}

// MulPoint transforms a point, applying the perspective divide.
func (m Mat4) MulPoint(p Vec3) Vec3 {
	v := m.MulVec4(p.Vec4(1))
	if v.W != 0 && v.W != 1 {
		return Vec3{v.X / v.W, v.Y / v.W, v.Z / v.W}
	}
	return v.Vec3()
}

// MulDir transforms a direction, ignoring translation.
func (m Mat4) MulDir(d Vec3) Vec3 { return m.MulVec4(d.Vec4(0)).Vec3() }

func (m Mat4) Translation() Vec3 { return Vec3{m[12], m[13], m[14]} }

func (m Mat4) Transpose() Mat4 {
	var r Mat4
	for c := 0; c < 4; c++ {
		for row := 0; row < 4; row++ {
			r[row*4+c] = m[c*4+row]
		}
	}
	return r
}

func Translate(t Vec3) Mat4 {
	m := Ident()
	m[12], m[13], m[14] = t.X, t.Y, t.Z
	return m
}

func Scale(s Vec3) Mat4 {
	return Mat4{s.X, 0, 0, 0, 0, s.Y, 0, 0, 0, 0, s.Z, 0, 0, 0, 0, 1}
}

// TRS composes translation * rotation * scale.
func TRS(t Vec3, r Quat, s Vec3) Mat4 {
	m := r.Mat4()
	for i := 0; i < 4; i++ {
		m[i] *= s.X
		m[4+i] *= s.Y
		m[8+i] *= s.Z
	}
	m[12], m[13], m[14] = t.X, t.Y, t.Z
	return m
}

// Perspective builds a projection with -1..1 depth, as OpenGL has it (the
// renderer maps it to WebGPU's 0..1). fovY is in radians.
func Perspective(fovY, aspect, near, far float32) Mat4 {
	f := 1 / Tan(fovY/2)
	var m Mat4
	m[0] = f / aspect
	m[5] = f
	m[10] = (far + near) / (near - far)
	m[11] = -1
	m[14] = 2 * far * near / (near - far)
	return m
}

func Ortho(left, right, bottom, top, near, far float32) Mat4 {
	m := Ident()
	m[0] = 2 / (right - left)
	m[5] = 2 / (top - bottom)
	m[10] = -2 / (far - near)
	m[12] = -(right + left) / (right - left)
	m[13] = -(top + bottom) / (top - bottom)
	m[14] = -(far + near) / (far - near)
	return m
}

// Mat4LookAt builds a view matrix for an eye looking at target.
func Mat4LookAt(eye, target, up Vec3) Mat4 {
	f := target.Sub(eye).Normalize()
	s := f.Cross(up).Normalize()
	if s.LenSq() < Epsilon {
		s = f.Cross(Backward).Normalize()
	}
	u := s.Cross(f)
	return Mat4{
		s.X, u.X, -f.X, 0,
		s.Y, u.Y, -f.Y, 0,
		s.Z, u.Z, -f.Z, 0,
		-s.Dot(eye), -u.Dot(eye), f.Dot(eye), 1,
	}
}

// Quat extracts the rotation of an orthonormal (unscaled) matrix.
func (m Mat4) Quat() Quat {
	m00, m11, m22 := m[0], m[5], m[10]
	tr := m00 + m11 + m22
	var q Quat
	switch {
	case tr > 0:
		s := Sqrt(tr+1) * 2
		q = Quat{(m[6] - m[9]) / s, (m[8] - m[2]) / s, (m[1] - m[4]) / s, s / 4}
	case m00 > m11 && m00 > m22:
		s := Sqrt(1+m00-m11-m22) * 2
		q = Quat{s / 4, (m[4] + m[1]) / s, (m[8] + m[2]) / s, (m[6] - m[9]) / s}
	case m11 > m22:
		s := Sqrt(1+m11-m00-m22) * 2
		q = Quat{(m[4] + m[1]) / s, s / 4, (m[9] + m[6]) / s, (m[8] - m[2]) / s}
	default:
		s := Sqrt(1+m22-m00-m11) * 2
		q = Quat{(m[8] + m[2]) / s, (m[9] + m[6]) / s, s / 4, (m[1] - m[4]) / s}
	}
	return q.Normalize()
}

// Inverse computes the general inverse. Singular matrices yield the identity.
func (m Mat4) Inverse() Mat4 {
	var inv Mat4
	inv[0] = m[5]*m[10]*m[15] - m[5]*m[11]*m[14] - m[9]*m[6]*m[15] + m[9]*m[7]*m[14] + m[13]*m[6]*m[11] - m[13]*m[7]*m[10]
	inv[4] = -m[4]*m[10]*m[15] + m[4]*m[11]*m[14] + m[8]*m[6]*m[15] - m[8]*m[7]*m[14] - m[12]*m[6]*m[11] + m[12]*m[7]*m[10]
	inv[8] = m[4]*m[9]*m[15] - m[4]*m[11]*m[13] - m[8]*m[5]*m[15] + m[8]*m[7]*m[13] + m[12]*m[5]*m[11] - m[12]*m[7]*m[9]
	inv[12] = -m[4]*m[9]*m[14] + m[4]*m[10]*m[13] + m[8]*m[5]*m[14] - m[8]*m[6]*m[13] - m[12]*m[5]*m[10] + m[12]*m[6]*m[9]
	inv[1] = -m[1]*m[10]*m[15] + m[1]*m[11]*m[14] + m[9]*m[2]*m[15] - m[9]*m[3]*m[14] - m[13]*m[2]*m[11] + m[13]*m[3]*m[10]
	inv[5] = m[0]*m[10]*m[15] - m[0]*m[11]*m[14] - m[8]*m[2]*m[15] + m[8]*m[3]*m[14] + m[12]*m[2]*m[11] - m[12]*m[3]*m[10]
	inv[9] = -m[0]*m[9]*m[15] + m[0]*m[11]*m[13] + m[8]*m[1]*m[15] - m[8]*m[3]*m[13] - m[12]*m[1]*m[11] + m[12]*m[3]*m[9]
	inv[13] = m[0]*m[9]*m[14] - m[0]*m[10]*m[13] - m[8]*m[1]*m[14] + m[8]*m[2]*m[13] + m[12]*m[1]*m[10] - m[12]*m[2]*m[9]
	inv[2] = m[1]*m[6]*m[15] - m[1]*m[7]*m[14] - m[5]*m[2]*m[15] + m[5]*m[3]*m[14] + m[13]*m[2]*m[7] - m[13]*m[3]*m[6]
	inv[6] = -m[0]*m[6]*m[15] + m[0]*m[7]*m[14] + m[4]*m[2]*m[15] - m[4]*m[3]*m[14] - m[12]*m[2]*m[7] + m[12]*m[3]*m[6]
	inv[10] = m[0]*m[5]*m[15] - m[0]*m[7]*m[13] - m[4]*m[1]*m[15] + m[4]*m[3]*m[13] + m[12]*m[1]*m[7] - m[12]*m[3]*m[5]
	inv[14] = -m[0]*m[5]*m[14] + m[0]*m[6]*m[13] + m[4]*m[1]*m[14] - m[4]*m[2]*m[13] - m[12]*m[1]*m[6] + m[12]*m[2]*m[5]
	inv[3] = -m[1]*m[6]*m[11] + m[1]*m[7]*m[10] + m[5]*m[2]*m[11] - m[5]*m[3]*m[10] - m[9]*m[2]*m[7] + m[9]*m[3]*m[6]
	inv[7] = m[0]*m[6]*m[11] - m[0]*m[7]*m[10] - m[4]*m[2]*m[11] + m[4]*m[3]*m[10] + m[8]*m[2]*m[7] - m[8]*m[3]*m[6]
	inv[11] = -m[0]*m[5]*m[11] + m[0]*m[7]*m[9] + m[4]*m[1]*m[11] - m[4]*m[3]*m[9] - m[8]*m[1]*m[7] + m[8]*m[3]*m[5]
	inv[15] = m[0]*m[5]*m[10] - m[0]*m[6]*m[9] - m[4]*m[1]*m[10] + m[4]*m[2]*m[9] + m[8]*m[1]*m[6] - m[8]*m[2]*m[5]

	det := m[0]*inv[0] + m[1]*inv[4] + m[2]*inv[8] + m[3]*inv[12]
	if Abs(det) < 1e-12 {
		return Ident()
	}
	inv1 := 1 / det
	for i := range inv {
		inv[i] *= inv1
	}
	return inv
}

// NormalMatrix returns the inverse-transpose used to transform normals.
func (m Mat4) NormalMatrix() Mat4 {
	n := m
	n[12], n[13], n[14] = 0, 0, 0
	return n.Inverse().Transpose()
}

func (m Mat4) ApproxEq(b Mat4, eps float32) bool {
	for i := range m {
		if Abs(m[i]-b[i]) > eps {
			return false
		}
	}
	return true
}
