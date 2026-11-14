package gltf

import (
	"fmt"

	"github.com/lantervale/ember/anim"
	"github.com/lantervale/ember/render"
)

// Without returns the model minus the joints drop marks, which must be
// whole subtrees, with every joint index renumbered: parents, raw node
// data, cameras, parts, skins and clip channels. Parts on dropped joints
// and channels animating them go too; a skin using one is an error. The
// model itself is left as is.
func (m *Model) Without(drop []bool) (*Model, error) {
	joints := m.Skeleton.Joints
	dropped := func(i int) bool { return i >= 0 && i < len(drop) && drop[i] }
	index := make([]int, len(joints))
	out := *m
	out.Skeleton = &anim.Skeleton{}
	out.Nodes, out.Parts, out.Clips, out.Cameras = nil, nil, nil, nil
	for i, j := range joints {
		if dropped(i) {
			index[i] = -1
			continue
		}
		if dropped(j.Parent) {
			return nil, fmt.Errorf("gltf: %s is kept under a joint that is dropped", j.Name)
		}
		index[i] = len(out.Skeleton.Joints)
		if j.Parent >= 0 {
			j.Parent = index[j.Parent]
		}
		out.Skeleton.Joints = append(out.Skeleton.Joints, j)
		if i < len(m.Nodes) {
			out.Nodes = append(out.Nodes, m.Nodes[i])
		}
	}
	for i, c := range m.Cameras {
		if !dropped(i) && i < len(index) {
			if out.Cameras == nil {
				out.Cameras = map[int]render.Camera{}
			}
			out.Cameras[index[i]] = c
		}
	}
	skins := map[*anim.Skin]*anim.Skin{}
	for _, p := range m.Parts {
		if dropped(p.Joint) {
			continue
		}
		c := *p
		if p.Joint >= 0 && p.Joint < len(index) {
			c.Joint = index[p.Joint]
		}
		if p.Skin != nil {
			s, ok := skins[p.Skin]
			if !ok {
				var err error
				if s, err = reskin(p, index, dropped, joints); err != nil {
					return nil, err
				}
				skins[p.Skin] = s
			}
			c.Skin = s
		}
		out.Parts = append(out.Parts, &c)
	}
	for _, clip := range m.Clips {
		c := *clip
		c.Channels = nil
		for _, ch := range clip.Channels {
			if !dropped(ch.Joint) {
				ch.Joint = index[ch.Joint]
				c.Channels = append(c.Channels, ch)
			}
		}
		out.Clips = append(out.Clips, &c)
	}
	return &out, nil
}

// reskin returns a copy of a part's skin on the renumbered joints.
func reskin(p *Part, index []int, dropped func(int) bool, joints []anim.Joint) (*anim.Skin, error) {
	s := &anim.Skin{InverseBind: p.Skin.InverseBind}
	for _, j := range p.Skin.Joints {
		if dropped(j) {
			return nil, fmt.Errorf("gltf: part %s is skinned to %s, which is dropped", p.Name, joints[j].Name)
		}
		s.Joints = append(s.Joints, index[j])
	}
	return s, nil
}
