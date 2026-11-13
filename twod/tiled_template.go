package twod

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"path"
)

// tiledTemplate is a Tiled template file (.tj): an object and, for a tile
// object, the tileset its tile is from.
type tiledTemplate struct {
	Object  json.RawMessage `json:"object"`
	Tileset *struct {
		FirstGID int    `json:"firstgid"`
		Source   string `json:"source"`
	} `json:"tileset"`
}

// templateGap is how far apart the global ids of tilesets a template brings
// start: more tiles than any tileset holds.
const templateGap = 1 << 20

// applyTemplates lays each object of m made from a template over the
// template's object, read once a file, its tile renumbered into the map's
// tilesets and the template's tileset added to them when the map doesn't
// name it.
func (t *Tiled) applyTemplates(fsys fs.FS, m *tiledMap) error {
	read := map[string]tiledTemplate{}
	return eachLayer(m.Layers, func(l *tiledLayer) error {
		for i, o := range l.Objects {
			if o.Template == "" {
				continue
			}
			made, err := t.fromTemplate(fsys, m, o, read)
			if err != nil {
				return fmt.Errorf("layer %s: object %d (%s): %w", l.Name, o.ID, o.Name, err)
			}
			l.Objects[i] = made
		}
		return nil
	})
}

// fromTemplate is object o, made from a template: the template's object,
// o's own values over it, a property at a time.
func (t *Tiled) fromTemplate(
	fsys fs.FS, m *tiledMap, o tiledObject, read map[string]tiledTemplate,
) (tiledObject, error) {
	file := path.Join(t.dir, o.Template)
	if path.Ext(file) != ".tj" {
		return tiledObject{}, fmt.Errorf("template %s: only templates saved as JSON (.tj) are read", o.Template)
	}
	tpl, ok := read[file]
	if !ok {
		if err := readJSON(fsys, file, &tpl); err != nil {
			return tiledObject{}, err
		}
		read[file] = tpl
	}
	var base, own map[string]any
	if err := json.Unmarshal(tpl.Object, &base); err != nil {
		return tiledObject{}, fmt.Errorf("template %s: %w", o.Template, err)
	}
	if err := json.Unmarshal(o.raw, &own); err != nil {
		return tiledObject{}, err
	}
	if err := fileProperties(base["properties"]); err != nil {
		return tiledObject{}, fmt.Errorf("template %s: %w", o.Template, err)
	}
	_, ownGID := own["gid"]
	for key, v := range own {
		switch key {
		case "template":
		case "properties":
			base[key] = overProperties(base[key], v)
		default:
			base[key] = v
		}
	}
	data, err := json.Marshal(base)
	if err != nil {
		return tiledObject{}, err
	}
	var made tiledObject
	if err = json.Unmarshal(data, &made); err != nil {
		return tiledObject{}, err
	}
	if made.GID != 0 && !ownGID {
		if made.GID, err = t.templateGID(m, made.GID, tpl, path.Dir(file)); err != nil {
			return tiledObject{}, fmt.Errorf("template %s: %w", o.Template, err)
		}
	}
	return made, nil
}

// templateGID is gid, a tile of template tpl in folder dir, renumbered as
// the map numbers its tilesets: by the map's tileset of the same file, else
// by the template's tileset added to the map's.
func (t *Tiled) templateGID(m *tiledMap, gid uint32, tpl tiledTemplate, dir string) (uint32, error) {
	if tpl.Tileset == nil || tpl.Tileset.Source == "" {
		return 0, errors.New("its tile names no tileset")
	}
	file := path.Join(dir, tpl.Tileset.Source)
	first, last := -1, 0
	for _, ts := range m.Tilesets {
		last = max(last, ts.FirstGID)
		same := ts.abs == file
		if ts.abs == "" && ts.Source != "" {
			same = path.Join(t.dir, ts.Source) == file
		}
		if same {
			first = ts.FirstGID
		}
	}
	if first < 0 {
		first = last + templateGap
		m.Tilesets = append(m.Tilesets, tiledTileset{FirstGID: first, Source: tpl.Tileset.Source, abs: file})
	}
	flags := gid & tiledFlags
	//nolint:gosec // a gid is under the flags' bits, the gaps between tilesets far below them
	return uint32(first+int(gid&^tiledFlags)-tpl.Tileset.FirstGID) | flags, nil
}

// overProperties are a template's properties with an object's own laid
// over them by name, the object's new ones after.
func overProperties(template, own any) any {
	base, _ := template.([]any)
	mine, _ := own.([]any)
	out := append([]any(nil), base...)
	for _, p := range mine {
		name := propertyName(p)
		replaced := false
		for i, q := range out {
			if propertyName(q) == name {
				out[i], replaced = p, true
			}
		}
		if !replaced {
			out = append(out, p)
		}
	}
	return out
}

func propertyName(p any) string {
	m, _ := p.(map[string]any)
	name, _ := m["name"].(string)
	return name
}

// fileProperties refuses a property naming a file in a template: its path
// would be read from the map's folder rather than the template's.
func fileProperties(list any) error {
	props, _ := list.([]any)
	for _, p := range props {
		if m, _ := p.(map[string]any); m["type"] == "file" {
			return fmt.Errorf("property %s: a file property in a template is not supported", propertyName(p))
		}
	}
	return nil
}
