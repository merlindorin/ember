package draw

import "slices"

// MaxPictures is how many light pictures a window draws besides its own:
// one for each set of occluder groups left out and lights left out that
// its lit drawings need. Past it, a drawing reads the window's picture.
const MaxPictures = 4

// Picture is a light picture a window draws besides its own: its light
// without the shadows of the occluder groups Left, and without the
// lights Excluded, by index in Lighting.Lights.
type Picture struct {
	Left     uint8
	Excluded []int
}

// PicturePlan is the light pictures a window draws besides its own, and
// which each key reads: Of[key] is a picture's index plus one, 0 for the
// window's picture. Fallen are the keys drawn that needed a picture past
// MaxPictures, and read the window's.
type PicturePlan struct {
	Pictures []Picture
	Of       [MaxKeys]int8
	Fallen   []int
}

// Plan works out the light pictures for the keys drawn, of lit drawings
// of classes, under lights: a key leaves out its groups, and every light
// Affecting layers none of which its class is under.
func (p *PicturePlan) Plan(lights []Light, classes [][]any, keys Keys) {
	p.Pictures, p.Fallen, p.Of = p.Pictures[:0], p.Fallen[:0], [MaxKeys]int8{}
	var excluded []int
	for key := range MaxKeys {
		if !keys.Has(key) {
			continue
		}
		left, class := uint8(key&AllGroups), key>>OccluderGroups
		var chain []any
		if class > 0 && class <= len(classes) {
			chain = classes[class-1]
		}
		excluded = excluded[:0]
		for i, l := range lights {
			if len(l.Affects) > 0 &&
				!slices.ContainsFunc(l.Affects, func(a any) bool { return slices.Contains(chain, a) }) {
				excluded = append(excluded, i)
			}
		}
		if left == 0 && len(excluded) == 0 {
			continue
		}
		at := slices.IndexFunc(
			p.Pictures,
			func(q Picture) bool { return q.Left == left && slices.Equal(q.Excluded, excluded) },
		)
		if at < 0 {
			if len(p.Pictures) == MaxPictures {
				p.Fallen = append(p.Fallen, key)
				continue
			}
			at = len(p.Pictures)
			p.Pictures = append(p.Pictures, Picture{Left: left, Excluded: slices.Clone(excluded)})
		}
		p.Of[key] = int8(at + 1) //nolint:gosec // at < MaxPictures
	}
}
