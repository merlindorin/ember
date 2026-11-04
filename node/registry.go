package node

import (
	"encoding/json"
	"fmt"
	"path"
	"reflect"
	"sort"
	"strings"
	"sync"
	"unicode"
	"unsafe"
)

// Type is a registered node class, or a behaviour: a Go type extending the
// class Base. Both are named by their Go package and type, node3d.Camera or
// game.Spawner, which Register fills in when Name is empty.
type Type struct {
	Name string
	Base string
	New  func() Noded
}

var (
	registryMu sync.RWMutex              //nolint:gochecknoglobals // guards the process-wide type registry
	registry   = map[string]Type{}       //nolint:gochecknoglobals // the types Register adds, by name
	byGoType   = map[reflect.Type]Type{} //nolint:gochecknoglobals // the types Register adds, by Go type
)

// Register adds a class or a behaviour; registering a name twice panics.
func Register(t Type) {
	if t.Name == "" {
		t.Name = typeName(reflect.TypeOf(t.New()))
	}
	registryMu.Lock()
	defer registryMu.Unlock()
	if _, dup := registry[t.Name]; dup {
		panic("node: " + t.Name + " registered twice")
	}
	registry[t.Name] = t
	byGoType[reflect.TypeOf(t.New())] = t
}

// ClassOf returns the class of a registered node and its behaviour, the
// type registered on top of a base class (empty when it is a class of its
// own); false when e's type was never registered.
func ClassOf(e Noded) (string, string, bool) {
	registryMu.RLock()
	defer registryMu.RUnlock()
	t, ok := byGoType[reflect.TypeOf(e)]
	switch {
	case !ok:
		return "", "", false
	case t.Base != "":
		return t.Base, t.Name, true
	}
	return t.Name, "", true
}

func Lookup(name string) (Type, bool) {
	registryMu.RLock()
	defer registryMu.RUnlock()
	t, ok := registry[name]
	return t, ok
}

// Types lists the registered classes and behaviours by name.
func Types() []Type {
	registryMu.RLock()
	defer registryMu.RUnlock()
	out := make([]Type, 0, len(registry))
	for _, t := range registry {
		out = append(out, t)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}

// Create makes a node of class, or of behaviour when it is set, which must
// extend class.
func Create(class, behaviour string) (Noded, error) {
	name := class
	if behaviour != "" {
		b, ok := Lookup(behaviour)
		if !ok {
			return nil, fmt.Errorf("node: no behaviour %s is registered", behaviour)
		}
		if b.Base != class {
			return nil, fmt.Errorf("node: behaviour %s extends %s, not %s", behaviour, b.Base, class)
		}
		name = behaviour
	}
	t, ok := Lookup(name)
	if !ok {
		return nil, fmt.Errorf("node: no class %s is registered", class)
	}
	return t.New(), nil
}

type propertyField struct {
	name  string
	index []int
}

var propertyFields sync.Map //nolint:gochecknoglobals // cache of each type's property fields

func propertiesOf(t reflect.Type) []propertyField {
	if f, ok := propertyFields.Load(t); ok {
		fields, _ := f.([]propertyField)
		return fields
	}
	var out []propertyField
	var scan func(reflect.Type, []int)
	scan = func(t reflect.Type, at []int) {
		for i := range t.NumField() {
			f := t.Field(i)
			index := append(append([]int(nil), at...), i)
			if f.Tag.Get("node") == "prop" {
				out = append(out, propertyField{name: snake(f.Name), index: index})
				continue
			}
			if !f.Anonymous {
				continue
			}
			switch {
			case f.Type.Kind() == reflect.Struct:
				scan(f.Type, index)
			case f.Type.Kind() == reflect.Pointer && f.Type.Elem().Kind() == reflect.Struct:
				scan(f.Type.Elem(), index)
			}
		}
	}
	scan(t, nil)
	propertyFields.Store(t, out)
	return out
}

func property(e Noded, name string) (reflect.Value, error) {
	v := reflect.ValueOf(e).Elem()
	for _, f := range propertiesOf(v.Type()) {
		if f.name == name {
			if fv, err := v.FieldByIndexErr(f.index); err == nil {
				if !fv.CanSet() {
					fv = reflect.NewAt(fv.Type(), unsafe.Pointer(fv.UnsafeAddr())).Elem()
				}
				return fv, nil
			}
		}
	}
	return reflect.Value{}, fmt.Errorf("node: %s has no property %s", e.GetSceneNode().Name(), name)
}

// Properties lists e's properties: the fields tagged `node:"prop"` in e and
// the structs it embeds, exported or not, in field order, each named after
// its field in snake_case (WaitTime is wait_time, sortByY is sort_by_y).
func Properties(e Noded) []string {
	var out []string
	for _, f := range propertiesOf(reflect.ValueOf(e).Elem().Type()) {
		out = append(out, f.name)
	}
	return out
}

// GetProperty reads a property as JSON, resources inline.
func GetProperty(e Noded, name string) (json.RawMessage, error) { return GetPropertyWith(e, name, nil) }

// SetProperty sets a property from JSON, resources inline.
func SetProperty(e Noded, name string, value json.RawMessage) error {
	return SetPropertyWith(e, name, value, nil)
}

// GetPropertyWith reads a property as a scene file stores it: resources
// are added to res and referred to by ID.
func GetPropertyWith(e Noded, name string, res *Resources) (json.RawMessage, error) {
	fv, err := property(e, name)
	if err != nil {
		return nil, err
	}
	return getValue(fv, res)
}

// SetPropertyWith sets a property as a scene file stores it, resolving
// resource references in res.
func SetPropertyWith(e Noded, name string, value json.RawMessage, res *Resources) error {
	fv, err := property(e, name)
	if err != nil {
		return err
	}
	if err = setValue(fv, value, res); err != nil {
		return fmt.Errorf("node: %s.%s: %w", e.GetSceneNode().Name(), name, err)
	}
	return nil
}

// snake turns a Go field name into a scene file name: WaitTime is
// wait_time, FOV is fov and ZIndex is z_index.
func snake(name string) string {
	var b strings.Builder
	rs := []rune(name)
	for i, r := range rs {
		if unicode.IsUpper(r) && i > 0 &&
			(unicode.IsLower(rs[i-1]) || unicode.IsDigit(rs[i-1]) || i+1 < len(rs) && unicode.IsLower(rs[i+1])) {
			b.WriteByte('_')
		}
		b.WriteRune(unicode.ToLower(r))
	}
	return b.String()
}

func typeName(t reflect.Type) string {
	for t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	return path.Base(t.PkgPath()) + "." + t.Name()
}
