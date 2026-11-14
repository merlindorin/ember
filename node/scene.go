package node

import (
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"regexp"
	"strings"

	"github.com/lantervale/ember/gltf"
)

const (
	nodeExtension  = "EMBER_node"
	sceneExtension = "EMBER_scene"
)

// Scene is a scene file: a self-contained glTF model whose nodes carry
// EMBER_node and whose document carries EMBER_scene. Files without them,
// such as Blender exports, load with empty IDs, which Encode assigns.
type Scene struct {
	ID       string
	Inherits string
	// Root is the scene's root when no glTF node holds it, such as the
	// Skeleton of an animated file.
	Root        *SceneNode
	Model       *gltf.Model
	Nodes       []SceneNode
	Connections []Connection
	Instances   []Instance
	// Resources are the resources the scene's properties refer to by ID.
	Resources map[string]ResourceData
}

// SceneNode is the EMBER_node data of a skeleton joint, at the same index.
type SceneNode struct {
	ID          string                     `json:"id"`
	Name        string                     `json:"name,omitempty"`
	Type        string                     `json:"type,omitempty"`
	Groups      []string                   `json:"groups,omitempty"`
	ProcessMode ProcessMode                `json:"process_mode,omitempty"`
	Behaviour   string                     `json:"behaviour,omitempty"`
	Properties  map[string]json.RawMessage `json:"properties,omitempty"`
	InstanceOf  string                     `json:"instance_of,omitempty"`
}

// Connection is a persisted signal connection between node IDs.
type Connection struct {
	Signal string `json:"signal"`
	From   string `json:"from"`
	To     string `json:"to"`
	Method string `json:"method"`
	Flags  Flags  `json:"flags,omitempty"`
}

// Instance places the scene with ID Scene at the node with ID Node.
type Instance struct {
	Node             string     `json:"node"`
	Scene            string     `json:"scene"`
	Overrides        []Override `json:"overrides,omitempty"`
	EditableChildren bool       `json:"editable_children,omitempty"`
}

// Override sets properties of a node inside an instance; Path is kept for
// readers only.
type Override struct {
	Node       string                     `json:"node"`
	Path       string                     `json:"path,omitempty"`
	Properties map[string]json.RawMessage `json:"properties"`
}

type sceneData struct {
	ID          string                  `json:"id"`
	Inherits    string                  `json:"inherits,omitempty"`
	Root        *SceneNode              `json:"root,omitempty"`
	Resources   map[string]ResourceData `json:"resources,omitempty"`
	Connections []Connection            `json:"connections,omitempty"`
	Instances   []Instance              `json:"instances,omitempty"`
}

var idPattern = regexp.MustCompile(`^[0-9a-f]{16}$`)

// Decode reads a scene from a .glb, failing on malformed or duplicate IDs
// and on connections to nodes the scene does not have.
func Decode(data []byte) (*Scene, error) {
	m, err := gltf.Decode(data)
	if err != nil {
		return nil, err
	}
	s := &Scene{Model: m, Nodes: make([]SceneNode, len(m.Skeleton.Joints))}
	if raw, ok := m.Extensions[sceneExtension]; ok {
		var d sceneData
		if err = json.Unmarshal(raw, &d); err != nil {
			return nil, fmt.Errorf("node: %s: %w", sceneExtension, err)
		}
		s.ID, s.Inherits, s.Root = d.ID, d.Inherits, d.Root
		s.Connections, s.Instances, s.Resources = d.Connections, d.Instances, d.Resources
		delete(m.Extensions, sceneExtension)
	}
	for i := range s.Nodes {
		if i >= len(m.Nodes) {
			break
		}
		raw, ok := m.Nodes[i].Extensions[nodeExtension]
		if !ok {
			continue
		}
		if err = json.Unmarshal(raw, &s.Nodes[i]); err != nil {
			return nil, fmt.Errorf("node: %s on %s: %w", nodeExtension, s.path(i), err)
		}
		delete(m.Nodes[i].Extensions, nodeExtension)
	}
	return s, s.check(true)
}

// Encode writes the scene as a .glb, first giving an ID to the scene and to
// every node without one.
func Encode(s *Scene) ([]byte, error) {
	if s.ID == "" {
		s.ID = NewID()
	}
	for len(s.Nodes) < len(s.Model.Skeleton.Joints) {
		s.Nodes = append(s.Nodes, SceneNode{})
	}
	for i := range s.Nodes {
		if s.Nodes[i].ID == "" {
			s.Nodes[i].ID = NewID()
		}
	}
	if s.Root != nil && s.Root.ID == "" {
		s.Root.ID = NewID()
	}
	if err := s.check(false); err != nil {
		return nil, err
	}
	return encode(s)
}

func encode(s *Scene) ([]byte, error) {
	m := s.Model
	for len(m.Nodes) < len(m.Skeleton.Joints) {
		m.Nodes = append(m.Nodes, gltf.Raw{})
	}
	if m.Extensions == nil {
		m.Extensions = map[string]json.RawMessage{}
	}
	scene, err := json.Marshal(sceneData{
		ID: s.ID, Inherits: s.Inherits, Root: s.Root,
		Resources: s.Resources, Connections: s.Connections, Instances: s.Instances,
	})
	if err != nil {
		return nil, err
	}
	m.Extensions[sceneExtension] = scene
	defer delete(m.Extensions, sceneExtension)
	for i, n := range s.Nodes {
		if i >= len(m.Nodes) {
			break
		}
		var raw []byte
		raw, err = json.Marshal(n)
		if err != nil {
			return nil, fmt.Errorf("node: %s: %w", s.path(i), err)
		}
		if m.Nodes[i].Extensions == nil {
			m.Nodes[i].Extensions = map[string]json.RawMessage{}
		}
		m.Nodes[i].Extensions[nodeExtension] = raw
		defer delete(m.Nodes[i].Extensions, nodeExtension)
	}
	return gltf.Encode(m)
}

func (s *Scene) check(loading bool) error {
	if s.ID != "" && !idPattern.MatchString(s.ID) {
		return fmt.Errorf("node: scene ID %q is not 16 hex characters", s.ID)
	}
	seen := map[string]int{}
	if r := s.Root; r != nil && (r.ID != "" || !loading) {
		if !idPattern.MatchString(r.ID) {
			return fmt.Errorf("node: root %s has ID %q, not 16 hex characters", r.Name, r.ID)
		}
		seen[r.ID] = -1
	}
	for i, n := range s.Nodes {
		if n.ID == "" && loading {
			continue
		}
		if !idPattern.MatchString(n.ID) {
			return fmt.Errorf("node: %s has ID %q, not 16 hex characters", s.path(i), n.ID)
		}
		if j, dup := seen[n.ID]; dup {
			return fmt.Errorf("node: ID %s is on both %s and %s", n.ID, s.path(j), s.path(i))
		}
		seen[n.ID] = i
	}
	for _, c := range s.Connections {
		for _, id := range []string{c.From, c.To} {
			if _, ok := seen[id]; !ok {
				return fmt.Errorf(
					"node: connection %s → %s.%s names node %s, which the scene does not have",
					c.Signal,
					c.To,
					c.Method,
					id,
				)
			}
		}
	}
	return nil
}

// path is a joint's path from the scene's root, for error messages; -1 is
// the scene's Root.
func (s *Scene) path(i int) string {
	if i < 0 && s.Root != nil {
		return s.Root.Name
	}
	joints := s.Model.Skeleton.Joints
	var parts []string
	for j := i; j >= 0 && j < len(joints); j = joints[j].Parent {
		parts = append(parts, joints[j].Name)
	}
	for l, r := 0, len(parts)-1; l < r; l, r = l+1, r-1 {
		parts[l], parts[r] = parts[r], parts[l]
	}
	return strings.Join(parts, "/")
}

// ReadID reads a scene's ID from the start of a .glb (or from a .gltf)
// without decoding it, "" for files without EMBER_scene.
func ReadID(r io.Reader) (string, error) {
	var head [12]byte
	n, err := io.ReadFull(r, head[:])
	var js []byte
	switch {
	case n == 12 && binary.LittleEndian.Uint32(head[:]) == 0x46546C67:
		var chunk [8]byte
		if _, err = io.ReadFull(r, chunk[:]); err != nil {
			return "", fmt.Errorf("node: glb header: %w", err)
		}
		js = make([]byte, binary.LittleEndian.Uint32(chunk[:]))
		if _, err = io.ReadFull(r, js); err != nil {
			return "", fmt.Errorf("node: glb JSON chunk: %w", err)
		}
	case err != nil && !errors.Is(err, io.ErrUnexpectedEOF):
		return "", err
	default:
		var rest []byte
		rest, err = io.ReadAll(r)
		if err != nil {
			return "", err
		}
		js = append(head[:n], rest...)
	}
	var doc struct {
		Extensions struct {
			Scene struct {
				ID string `json:"id"`
			} `json:"EMBER_scene"`
		} `json:"extensions"`
	}
	if err = json.Unmarshal(js, &doc); err != nil {
		return "", fmt.Errorf("node: %w", err)
	}
	return doc.Extensions.Scene.ID, nil
}
