package anim

import "github.com/lantervale/ember/geom"

// Retarget maps a clip made for one skeleton onto another whose joints
// have the same names (a shared rig family, an animation library).
// Rotations transfer as changes from each skeleton's rest pose, so rest
// orientations may differ. Translations transfer where the two bones have
// the same length; on the root of the motion (the hips) they are scaled
// by the ratio of the two hips' heights so a smaller character takes
// smaller steps. Channels of joints the target lacks are dropped.
func Retarget(c *Clip, from, to *Skeleton) *Clip {
	out := &Clip{Name: c.Name, Start: c.Start, Duration: c.Duration}
	fromRest, toRest := from.Globals(from.RestPose(), nil), to.Globals(to.RestPose(), nil)
	for _, ch := range c.Channels {
		if ch.Joint < 0 || ch.Joint >= len(from.Joints) {
			continue
		}
		j := to.Find(from.Joints[ch.Joint].Name)
		if j < 0 {
			continue
		}
		fr, tr := from.Joints[ch.Joint].Rest, to.Joints[j].Rest
		n := Channel{
			Joint: j, Path: ch.Path, Interpolation: ch.Interpolation,
			Times: ch.Times, Values: make([]geom.Vec4, len(ch.Values)),
		}
		switch ch.Path {
		case Rotation:
			delta := tr.R.Mul(fr.R.Conjugate())
			for i, v := range ch.Values {
				q := delta.Mul(geom.Quat(v))
				n.Values[i] = geom.V4(q.X, q.Y, q.Z, q.W)
			}
		case Translation:
			root := from.Joints[ch.Joint].Parent < 0 || fr.T.Len() > 0.3*span(from)
			k := float32(1)
			switch {
			case root:
				if hf, ht := fromRest[ch.Joint].Translation().Y, toRest[j].Translation().Y; hf > 1e-3 && ht > 1e-3 {
					k = ht / hf
				} else if sf, st := span(from), span(to); sf > 0 && st > 0 {
					k = st / sf
				}
			case !sameLength(fr.T.Len(), tr.T.Len()):
				continue
			}
			for i, v := range ch.Values {
				offset := tr.T
				if n.Interpolation == CubicSpline && i%3 != 1 {
					offset = geom.Vec3{}
				}
				fo := fr.T
				if n.Interpolation == CubicSpline && i%3 != 1 {
					fo = geom.Vec3{}
				}
				n.Values[i] = offset.Add(v.Vec3().Sub(fo).Scale(k)).Vec4(0)
			}
		case Scale: // scale keys are copied as they are
			copy(n.Values, ch.Values)
		default:
			copy(n.Values, ch.Values)
		}
		out.Channels = append(out.Channels, n)
	}
	return out
}

func sameLength(a, b float32) bool {
	if a < 1e-4 && b < 1e-4 {
		return true
	}
	return geom.Abs(a-b) <= 0.1*max(a, b)
}

// span is how far a skeleton's joints spread vertically at rest.
func span(s *Skeleton) float32 {
	g := s.Globals(s.RestPose(), nil)
	if len(g) == 0 {
		return 0
	}
	lo, hi := g[0].Translation().Y, g[0].Translation().Y
	for _, m := range g {
		y := m.Translation().Y
		lo, hi = min(lo, y), max(hi, y)
	}
	return hi - lo
}
