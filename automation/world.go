package automation

import (
	"reflect"
	"strings"

	"github.com/merlindorin/ember/world"
)

// EntityInfo summarizes an entity for listings.
type EntityInfo struct {
	ID         uint64
	Type       string
	Name       string
	Components []string
	Parent     uint64
	Children   []uint64
}

// Describe summarizes one entity.
func Describe(e world.Identifier) EntityInfo {
	info := EntityInfo{ID: e.ID(), Type: TypeName(e), Name: DisplayName(e)}
	t := reflect.TypeOf(e)
	for t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	if t.Kind() == reflect.Struct {
		for i := 0; i < t.NumField(); i++ {
			if f := t.Field(i); f.Anonymous {
				info.Components = append(info.Components, shortType(f.Type))
			}
		}
	}
	if b, ok := e.(world.BasicFace); ok {
		basic := b.GetBasicEntity()
		if p := basic.Parent(); p != nil {
			info.Parent = p.ID()
		}
		for _, c := range basic.Children() {
			info.Children = append(info.Children, c.ID())
		}
	}
	return info
}

// EntityFilter selects entities; empty fields match everything.
type EntityFilter struct {
	Type      string
	Component string
	Name      string
	Limit     int
}

// Entities lists matching entities and the total number of matches.
func Entities(w *world.World, f EntityFilter) ([]EntityInfo, int) {
	if f.Limit <= 0 {
		f.Limit = 500
	}
	var out []EntityInfo
	total := 0
	for _, e := range w.Entities() {
		info := Describe(e)
		if f.Type != "" && !strings.Contains(strings.ToLower(info.Type), strings.ToLower(f.Type)) {
			continue
		}
		if f.Name != "" && !strings.Contains(strings.ToLower(info.Name), strings.ToLower(f.Name)) {
			continue
		}
		if f.Component != "" && !hasComponent(info.Components, f.Component) {
			continue
		}
		total++
		if len(out) < f.Limit {
			out = append(out, info)
		}
	}
	return out, total
}

func hasComponent(components []string, want string) bool {
	for _, c := range components {
		if c == want || strings.HasSuffix(c, "."+want) {
			return true
		}
	}
	return false
}

// TypeName is the entity's Go type without the pointer star.
func TypeName(v any) string {
	return strings.TrimPrefix(reflect.TypeOf(v).String(), "*")
}

// DisplayName finds a human name: a Title() method, then a Name or Kind
// string field, including promoted fields of embedded components.
func DisplayName(e any) string {
	if n, ok := e.(interface{ Title() string }); ok && n.Title() != "" {
		return n.Title()
	}
	v := reflect.ValueOf(e)
	for v.Kind() == reflect.Pointer && !v.IsNil() {
		v = v.Elem()
	}
	if v.Kind() != reflect.Struct {
		return ""
	}
	for _, field := range []string{"Name", "Kind"} {
		if f := v.FieldByName(field); f.IsValid() && f.Kind() == reflect.String && f.String() != "" {
			return f.String()
		}
	}
	return ""
}

// Resources lists resource type names.
func Resources(w *world.World) []string {
	var out []string
	for _, r := range w.Resources() {
		name := shortType(r.Type)
		if !strings.Contains(name, "eventQueue") {
			out = append(out, name)
		}
	}
	return out
}
