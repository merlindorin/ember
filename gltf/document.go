package gltf

import (
	"encoding/json"
	"sort"

	"github.com/lantervale/ember/anim"
	"github.com/lantervale/ember/geom"
	"github.com/lantervale/ember/render"
)

// document mirrors the subset of the glTF 2.0 JSON schema the loader uses.
type document struct {
	Scene       int           `json:"scene"`
	Scenes      []sceneDef    `json:"scenes"`
	Nodes       []node        `json:"nodes"`
	Meshes      []meshDef     `json:"meshes"`
	Skins       []skinDef     `json:"skins"`
	Animations  []animation   `json:"animations"`
	Materials   []materialDef `json:"materials"`
	Textures    []textureDef  `json:"textures"`
	Images      []imageDef    `json:"images"`
	Samplers    []sampler     `json:"samplers"`
	Accessors   []accessor    `json:"accessors"`
	BufferViews []bufferView  `json:"bufferViews"`
	Buffers     []buffer      `json:"buffers"`
	Cameras     []cameraDef   `json:"cameras"`

	ExtensionsRequired []string                   `json:"extensionsRequired"`
	Extensions         map[string]json.RawMessage `json:"extensions"`
	Extras             json.RawMessage            `json:"extras"`
}

type sceneDef struct {
	Name  string `json:"name"`
	Nodes []int  `json:"nodes"`
}

type node struct {
	Name        string    `json:"name"`
	Children    []int     `json:"children"`
	Mesh        *int      `json:"mesh"`
	Skin        *int      `json:"skin"`
	Camera      *int      `json:"camera"`
	Matrix      []float32 `json:"matrix"`
	Translation []float32 `json:"translation"`
	Rotation    []float32 `json:"rotation"`
	Scale       []float32 `json:"scale"`

	Extensions map[string]json.RawMessage `json:"extensions"`
	Extras     json.RawMessage            `json:"extras"`
}

func (n *node) trs() anim.TRS {
	t := anim.TRS{R: geom.QuatIdent(), S: geom.V3(1, 1, 1)}
	if len(n.Matrix) == 16 {
		var m geom.Mat4
		copy(m[:], n.Matrix)
		sx := geom.V3(m[0], m[1], m[2]).Len()
		sy := geom.V3(m[4], m[5], m[6]).Len()
		sz := geom.V3(m[8], m[9], m[10]).Len()
		t.T, t.S = m.Translation(), geom.V3(sx, sy, sz)
		if sx > 0 && sy > 0 && sz > 0 {
			for c := range 3 {
				m[c] /= sx
				m[4+c] /= sy
				m[8+c] /= sz
			}
			t.R = m.Quat().Normalize()
		}
		return t
	}
	if v := n.Translation; len(v) == 3 {
		t.T = geom.V3(v[0], v[1], v[2])
	}
	if v := n.Rotation; len(v) == 4 {
		t.R = geom.Quat{X: v[0], Y: v[1], Z: v[2], W: v[3]}.Normalize()
	}
	if v := n.Scale; len(v) == 3 {
		t.S = geom.V3(v[0], v[1], v[2])
	}
	return t
}

// rootNodes returns the nodes nobody lists as a child, for files without
// scenes.
func (d *document) rootNodes() []int {
	child := make([]bool, len(d.Nodes))
	for _, n := range d.Nodes {
		for _, c := range n.Children {
			if c >= 0 && c < len(child) {
				child[c] = true
			}
		}
	}
	var roots []int
	for i, c := range child {
		if !c {
			roots = append(roots, i)
		}
	}
	return roots
}

type meshDef struct {
	Name       string      `json:"name"`
	Primitives []primitive `json:"primitives"`
}

type primitive struct {
	Attributes map[string]int `json:"attributes"`
	Indices    *int           `json:"indices"`
	Material   *int           `json:"material"`
	Mode       *int           `json:"mode"`
}

type skinDef struct {
	InverseBindMatrices *int  `json:"inverseBindMatrices"`
	Joints              []int `json:"joints"`
}

type animation struct {
	Name     string        `json:"name"`
	Channels []animChannel `json:"channels"`
	Samplers []animSampler `json:"samplers"`
}

type animChannel struct {
	Sampler int `json:"sampler"`
	Target  struct {
		Node *int   `json:"node"`
		Path string `json:"path"`
	} `json:"target"`
}

type animSampler struct {
	Input         int    `json:"input"`
	Output        int    `json:"output"`
	Interpolation string `json:"interpolation"`
}

type materialDef struct {
	Name           string    `json:"name"`
	AlphaMode      string    `json:"alphaMode"`
	DoubleSided    bool      `json:"doubleSided"`
	EmissiveFactor []float32 `json:"emissiveFactor"`
	PBR            struct {
		BaseColorFactor  []float32   `json:"baseColorFactor"`
		BaseColorTexture *textureRef `json:"baseColorTexture"`
		RoughnessFactor  *float32    `json:"roughnessFactor"`
	} `json:"pbrMetallicRoughness"`
	Extensions struct {
		EmissiveStrength *struct {
			EmissiveStrength float32 `json:"emissiveStrength"`
		} `json:"KHR_materials_emissive_strength"`
		Unlit *struct{} `json:"KHR_materials_unlit"`
	} `json:"extensions"`
}

type textureRef struct {
	Index int `json:"index"`
}

type textureDef struct {
	Source  *int `json:"source"`
	Sampler *int `json:"sampler"`
}

type imageDef struct {
	Name       string `json:"name"`
	URI        string `json:"uri"`
	MimeType   string `json:"mimeType"`
	BufferView *int   `json:"bufferView"`
}

type sampler struct {
	MagFilter int `json:"magFilter"`
	MinFilter int `json:"minFilter"`
	WrapS     int `json:"wrapS"`
	WrapT     int `json:"wrapT"`
}

type accessor struct {
	BufferView    *int   `json:"bufferView"`
	ByteOffset    int    `json:"byteOffset"`
	ComponentType int    `json:"componentType"`
	Normalized    bool   `json:"normalized"`
	Count         int    `json:"count"`
	Type          string `json:"type"`
}

// Accessor types.
const (
	typeScalar = "SCALAR"
	typeVec2   = "VEC2"
	typeVec3   = "VEC3"
	typeVec4   = "VEC4"
	typeMat2   = "MAT2"
	typeMat3   = "MAT3"
	typeMat4   = "MAT4"
)

func (a accessor) components() int {
	switch a.Type {
	case typeScalar:
		return 1
	case typeVec2:
		return 2
	case typeVec3:
		return 3
	case typeVec4, typeMat2:
		return 4
	case typeMat3:
		return 9
	case typeMat4:
		return 16
	}
	return 0
}

type bufferView struct {
	Buffer     int `json:"buffer"`
	ByteOffset int `json:"byteOffset"`
	ByteLength int `json:"byteLength"`
	ByteStride int `json:"byteStride"`
}

type buffer struct {
	URI        string `json:"uri"`
	ByteLength int    `json:"byteLength"`
}

func sortParts(parts []*Part) {
	sort.SliceStable(parts, func(i, j int) bool { return parts[i].Name < parts[j].Name })
}

type cameraDef struct {
	Type        string `json:"type"`
	Perspective *struct {
		YFov  float32 `json:"yfov"`
		ZNear float32 `json:"znear"`
		ZFar  float32 `json:"zfar,omitempty"`
	} `json:"perspective,omitempty"`
	Orthographic *struct {
		XMag  float32 `json:"xmag"`
		YMag  float32 `json:"ymag"`
		ZNear float32 `json:"znear"`
		ZFar  float32 `json:"zfar"`
	} `json:"orthographic,omitempty"`
}

func (c cameraDef) camera() render.Camera {
	switch {
	case c.Perspective != nil:
		return render.Camera{FOV: c.Perspective.YFov, Near: c.Perspective.ZNear, Far: c.Perspective.ZFar}
	case c.Orthographic != nil:
		return render.Camera{OrthoHeight: 2 * c.Orthographic.YMag, Near: c.Orthographic.ZNear, Far: c.Orthographic.ZFar}
	}
	return render.Camera{}
}

func cameraOf(c render.Camera) cameraDef {
	var d cameraDef
	if c.OrthoHeight > 0 {
		d.Type = "orthographic"
		h := c.OrthoHeight / 2
		d.Orthographic = &struct {
			XMag  float32 `json:"xmag"`
			YMag  float32 `json:"ymag"`
			ZNear float32 `json:"znear"`
			ZFar  float32 `json:"zfar"`
		}{h, h, c.Near, c.Far}
		return d
	}
	d.Type = "perspective"
	d.Perspective = &struct {
		YFov  float32 `json:"yfov"`
		ZNear float32 `json:"znear"`
		ZFar  float32 `json:"zfar,omitempty"`
	}{c.FOV, c.Near, c.Far}
	return d
}
