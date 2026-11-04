package node

import (
	"bytes"
	"encoding/json"
	"fmt"
	"reflect"
	"sync"
)

// Resource is shared data that node properties point to, such as a Theme.
// Scene files store each once, in EMBER_scene.resources, and properties
// refer to it as {"resource": "<id>"}.
type Resource interface{ ResourceType() string }

var (
	resourceMu    sync.RWMutex                   //nolint:gochecknoglobals // guards the process-wide resource registry
	resourceTypes = map[string]func() Resource{} //nolint:gochecknoglobals // the types RegisterResource adds
)

// RegisterResource makes a resource type loadable from scene files.
func RegisterResource(typ string, newResource func() Resource) {
	resourceMu.Lock()
	defer resourceMu.Unlock()
	resourceTypes[typ] = newResource
}

// ResourceData is a resource as a scene file stores it.
type ResourceData struct {
	Type string          `json:"type"`
	Data json.RawMessage `json:"data"`
}

// Resources is the resource table of a scene being saved or loaded.
type Resources struct {
	ByID map[string]Resource
	ids  map[Resource]string
}

func (r *Resources) add(v Resource) string {
	if id, ok := r.ids[v]; ok {
		return id
	}
	if r.ByID == nil {
		r.ByID, r.ids = map[string]Resource{}, map[Resource]string{}
	}
	id := NewID()
	r.ByID[id], r.ids[v] = v, id
	return id
}

// Data encodes the table for a scene file.
func (r *Resources) Data() map[string]ResourceData {
	if len(r.ByID) == 0 {
		return nil
	}
	out := map[string]ResourceData{}
	for id, v := range r.ByID {
		b, _ := json.Marshal(v)
		out[id] = ResourceData{Type: v.ResourceType(), Data: b}
	}
	return out
}

// LoadResources decodes a scene file's resources, failing on unknown types.
func LoadResources(data map[string]ResourceData) (*Resources, error) {
	r := &Resources{}
	for id, d := range data {
		resourceMu.RLock()
		mk, ok := resourceTypes[d.Type]
		resourceMu.RUnlock()
		if !ok {
			return nil, fmt.Errorf("node: resource %s has unknown type %s", id, d.Type)
		}
		v := mk()
		if err := json.Unmarshal(d.Data, v); err != nil {
			return nil, fmt.Errorf("node: resource %s: %w", id, err)
		}
		if r.ByID == nil {
			r.ByID, r.ids = map[string]Resource{}, map[Resource]string{}
		}
		r.ByID[id], r.ids[v] = v, id
	}
	return r, nil
}

type reference struct {
	Resource string `json:"resource"`
}

func getValue(fv reflect.Value, res *Resources) (json.RawMessage, error) {
	if res != nil && fv.Type().Implements(reflect.TypeFor[Resource]()) && !fv.IsNil() {
		r, _ := fv.Interface().(Resource)
		return json.Marshal(reference{res.add(r)})
	}
	return json.Marshal(fv.Interface())
}

func setValue(fv reflect.Value, raw json.RawMessage, res *Resources) error {
	if fv.Type().Implements(reflect.TypeFor[Resource]()) && fv.Kind() == reflect.Pointer {
		return setResource(fv, raw, res)
	}
	return json.Unmarshal(raw, fv.Addr().Interface())
}

// setResource sets a resource property from a reference into res, or from
// the resource's data inline.
func setResource(fv reflect.Value, raw json.RawMessage, res *Resources) error {
	var ref reference
	if bytes.Contains(raw, []byte(`"resource"`)) && json.Unmarshal(raw, &ref) == nil && ref.Resource != "" {
		v, ok := res.get(ref.Resource)
		if !ok {
			return fmt.Errorf("no resource %s", ref.Resource)
		}
		rv := reflect.ValueOf(v)
		if !rv.Type().AssignableTo(fv.Type()) {
			return fmt.Errorf("resource %s is a %s", ref.Resource, v.ResourceType())
		}
		fv.Set(rv)
		return nil
	}
	fresh := reflect.New(fv.Type().Elem())
	if string(bytes.TrimSpace(raw)) == "null" {
		fv.Set(reflect.Zero(fv.Type()))
		return nil
	}
	if err := json.Unmarshal(raw, fresh.Interface()); err != nil {
		return err
	}
	fv.Set(fresh)
	return nil
}

func (r *Resources) get(id string) (Resource, bool) {
	if r == nil {
		return nil, false
	}
	v, ok := r.ByID[id]
	return v, ok
}
