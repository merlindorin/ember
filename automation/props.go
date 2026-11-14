package automation

import (
	"fmt"
	"math"
	"reflect"
	"sort"
	"strconv"
	"strings"

	"github.com/lantervale/ember/world"
)

// Paths address live values:
//
//	entity/42                          the whole entity
//	entity/42/Transform/Position       a component field (embedded components use their type name)
//	entity/42/Position/X               promoted fields work too
//	time/scale                         the World's clock (fields in any case)
//	stats/fps                          the World's frame metrics
//	resource/Weather/Wind              a game resource field (short or full type name)
//	entity/42/Inventory/Slots/3/Count  slices take indices, maps take keys
func resolve(w *world.World, path string) (reflect.Value, error) {
	segs := strings.Split(strings.Trim(path, "/"), "/")
	if len(segs) < 2 {
		return reflect.Value{}, fmt.Errorf("automation: path %q must start with entity/<id> or resource/<type>", path)
	}
	var v reflect.Value
	rest := segs[2:]
	switch segs[0] {
	case "time":
		v, rest = reflect.ValueOf(w.Time()), segs[1:]
	case "stats":
		v, rest = reflect.ValueOf(w.Stats()), segs[1:]
	case "entity":
		id, err := strconv.ParseUint(segs[1], 10, 64)
		if err != nil {
			return reflect.Value{}, fmt.Errorf("automation: bad entity id %q", segs[1])
		}
		e, ok := w.Entity(id)
		if !ok {
			return reflect.Value{}, fmt.Errorf("automation: no entity %d", id)
		}
		v = reflect.ValueOf(e)
	case "resource":
		r, ok := findResource(w, segs[1])
		if !ok {
			return reflect.Value{}, fmt.Errorf("automation: no resource %q", segs[1])
		}
		v = reflect.ValueOf(r)
	default:
		return reflect.Value{}, fmt.Errorf("automation: path %q must start with entity/, resource/, time/ or stats/", path)
	}
	for _, seg := range rest {
		next, err := step(v, seg)
		if err != nil {
			return reflect.Value{}, fmt.Errorf("automation: %s: %w", path, err)
		}
		v = next
	}
	return v, nil
}

func findResource(w *world.World, name string) (any, bool) {
	for _, r := range w.Resources() {
		full := r.Type.String()
		if full == name || shortType(r.Type) == name || full[strings.LastIndex(full, ".")+1:] == name {
			return r.Value, true
		}
	}
	return nil, false
}

func step(v reflect.Value, seg string) (reflect.Value, error) {
	for v.Kind() == reflect.Pointer || v.Kind() == reflect.Interface {
		if v.IsNil() {
			return reflect.Value{}, fmt.Errorf("nil before %q", seg)
		}
		v = v.Elem()
	}
	switch v.Kind() {
	case reflect.Struct:
		f, ok := v.Type().FieldByName(seg)
		if !ok {
			f, ok = v.Type().FieldByNameFunc(func(name string) bool {
				return strings.EqualFold(name, strings.ReplaceAll(seg, "_", ""))
			})
		}
		if !ok || !f.IsExported() {
			return reflect.Value{}, fmt.Errorf("no exported field %q in %s", seg, v.Type())
		}
		return v.FieldByIndex(f.Index), nil
	case reflect.Slice, reflect.Array:
		i, err := strconv.Atoi(seg)
		if err != nil || i < 0 || i >= v.Len() {
			return reflect.Value{}, fmt.Errorf("index %q out of range [0,%d)", seg, v.Len())
		}
		return v.Index(i), nil
	case reflect.Map:
		if v.Type().Key().Kind() != reflect.String {
			return reflect.Value{}, fmt.Errorf("only string-keyed maps are addressable")
		}
		mv := v.MapIndex(reflect.ValueOf(seg).Convert(v.Type().Key()))
		if !mv.IsValid() {
			return reflect.Value{}, fmt.Errorf("no key %q", seg)
		}
		return mv, nil
	case reflect.Invalid, reflect.Bool, reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr,
		reflect.Float32, reflect.Float64, reflect.Complex64, reflect.Complex128, reflect.Chan, reflect.Func,
		reflect.Interface, reflect.Pointer, reflect.String, reflect.UnsafePointer:
	}
	return reflect.Value{}, fmt.Errorf("cannot descend into %s with %q", v.Type(), seg)
}

// Get reads the value at path as plain data (maps, lists, numbers…).
func Get(w *world.World, path string, depth int) (any, string, error) {
	v, err := resolve(w, path)
	if err != nil {
		return nil, "", err
	}
	if depth <= 0 {
		depth = 6
	}
	return Encode(v, depth), v.Type().String(), nil
}

// Set writes plain data into the exported value at path.
func Set(w *world.World, path string, value any) (any, error) {
	v, err := resolve(w, path)
	if err != nil {
		return nil, err
	}
	if !v.CanSet() {
		return nil, fmt.Errorf("automation: %s is not settable", path)
	}
	if err = Decode(v, value); err != nil {
		return nil, fmt.Errorf("automation: %s: %w", path, err)
	}
	return Encode(v, 6), nil
}

const maxItems = 64

// Encode converts any Go value into plain data: structs become maps of
// their exported fields (embedded components keyed by type name), long
// slices are truncated, functions and channels are dropped.
func Encode(v reflect.Value, depth int) any {
	return encode(v, depth, map[uintptr]bool{})
}

func encode(v reflect.Value, depth int, seen map[uintptr]bool) any {
	if !v.IsValid() {
		return nil
	}
	switch v.Kind() {
	case reflect.Pointer:
		return encodePointer(v, depth, seen)
	case reflect.Interface:
		if v.IsNil() {
			return nil
		}
		return encode(v.Elem(), depth, seen)
	case reflect.Bool:
		return v.Bool()
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return float64(v.Int())
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		return float64(v.Uint())
	case reflect.Float32, reflect.Float64:
		f := v.Float()
		if math.IsNaN(f) || math.IsInf(f, 0) {
			return fmt.Sprint(f)
		}
		return f
	case reflect.String:
		return v.String()
	case reflect.Func, reflect.Chan, reflect.UnsafePointer, reflect.Complex64, reflect.Complex128:
		return nil
	case reflect.Struct:
		if depth <= 0 {
			return elided(v)
		}
		return encodeStruct(v, depth, seen)
	case reflect.Slice, reflect.Array:
		if depth <= 0 {
			return elided(v)
		}
		return encodeList(v, depth, seen)
	case reflect.Map:
		if depth <= 0 {
			return elided(v)
		}
		return encodeMap(v, depth, seen)
	case reflect.Invalid:
		return nil
	}
	return v.Type().String()
}

func elided(v reflect.Value) string { return "(" + v.Type().String() + ")" }

func encodePointer(v reflect.Value, depth int, seen map[uintptr]bool) any {
	if v.IsNil() {
		return nil
	}
	if seen[v.Pointer()] {
		return "(cycle)"
	}
	seen[v.Pointer()] = true
	defer delete(seen, v.Pointer())
	if v.Type().Elem() == reflect.TypeFor[world.BasicEntity]() {
		return encodeBasic(v)
	}
	return encode(v.Elem(), depth, seen)
}

func encodeBasic(v reflect.Value) any {
	b, ok := v.Interface().(*world.BasicEntity)
	if !ok {
		return nil
	}
	out := map[string]any{"ID": float64(b.ID()), "Children": float64(len(b.Children()))}
	if p := b.Parent(); p != nil {
		out["Parent"] = float64(p.ID())
	}
	return out
}

func encodeStruct(v reflect.Value, depth int, seen map[uintptr]bool) any {
	out := map[string]any{}
	t := v.Type()
	for i := 0; i < t.NumField(); i++ {
		f := t.Field(i)
		if !f.IsExported() {
			continue
		}
		fv := v.Field(i)
		if k := fv.Kind(); k == reflect.Func || k == reflect.Chan {
			continue
		}
		out[f.Name] = encode(fv, depth-1, seen)
	}
	return out
}

func encodeList(v reflect.Value, depth int, seen map[uintptr]bool) any {
	n := v.Len()
	items := make([]any, 0, min(n, maxItems))
	for i := 0; i < min(n, maxItems); i++ {
		items = append(items, encode(v.Index(i), depth-1, seen))
	}
	if n > maxItems {
		return map[string]any{"Len": float64(n), "Head": items}
	}
	return items
}

func encodeMap(v reflect.Value, depth int, seen map[uintptr]bool) any {
	if v.Type().Key().Kind() != reflect.String {
		return map[string]any{"Len": float64(v.Len())}
	}
	out := map[string]any{}
	keys := v.MapKeys()
	sort.Slice(keys, func(i, j int) bool { return keys[i].String() < keys[j].String() })
	for i, k := range keys {
		if i >= maxItems {
			out["…"] = float64(len(keys) - maxItems)
			break
		}
		out[k.String()] = encode(v.MapIndex(k), depth-1, seen)
	}
	return out
}

// Decode writes plain data (as produced by Encode or JSON) into dst.
// Structs are updated field by field, so partial maps leave other fields
// untouched.
func Decode(dst reflect.Value, src any) error {
	if src == nil {
		dst.Set(reflect.Zero(dst.Type()))
		return nil
	}
	switch dst.Kind() {
	case reflect.Pointer:
		if dst.IsNil() {
			dst.Set(reflect.New(dst.Type().Elem()))
		}
		return Decode(dst.Elem(), src)
	case reflect.Bool:
		b, ok := src.(bool)
		if !ok {
			return fmt.Errorf("want bool, got %T", src)
		}
		dst.SetBool(b)
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		f, ok := src.(float64)
		if !ok {
			return fmt.Errorf("want number, got %T", src)
		}
		dst.SetInt(int64(math.Round(f)))
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		f, ok := src.(float64)
		if !ok || f < 0 {
			return fmt.Errorf("want non-negative number, got %v", src)
		}
		dst.SetUint(uint64(math.Round(f)))
	case reflect.Float32, reflect.Float64:
		f, ok := src.(float64)
		if !ok {
			return fmt.Errorf("want number, got %T", src)
		}
		dst.SetFloat(f)
	case reflect.String:
		s, ok := src.(string)
		if !ok {
			return fmt.Errorf("want string, got %T", src)
		}
		dst.SetString(s)
	case reflect.Struct:
		return decodeStruct(dst, src)
	case reflect.Slice:
		return decodeSlice(dst, src)
	case reflect.Array:
		return decodeArray(dst, src)
	case reflect.Invalid, reflect.Complex64, reflect.Complex128, reflect.Chan, reflect.Func,
		reflect.Interface, reflect.Map, reflect.UnsafePointer:
		return fmt.Errorf("cannot set %s", dst.Type())
	}
	return nil
}

func decodeStruct(dst reflect.Value, src any) error {
	m, ok := src.(map[string]any)
	if !ok {
		return fmt.Errorf("want object for %s, got %T", dst.Type(), src)
	}
	for k, val := range m {
		f, found := dst.Type().FieldByName(k)
		if !found || !f.IsExported() {
			return fmt.Errorf("no exported field %q in %s", k, dst.Type())
		}
		if err := Decode(dst.FieldByIndex(f.Index), val); err != nil {
			return fmt.Errorf("%s: %w", k, err)
		}
	}
	return nil
}

func decodeSlice(dst reflect.Value, src any) error {
	list, ok := src.([]any)
	if !ok {
		return fmt.Errorf("want list, got %T", src)
	}
	s := reflect.MakeSlice(dst.Type(), len(list), len(list))
	for i, item := range list {
		if err := Decode(s.Index(i), item); err != nil {
			return fmt.Errorf("[%d]: %w", i, err)
		}
	}
	dst.Set(s)
	return nil
}

func decodeArray(dst reflect.Value, src any) error {
	list, ok := src.([]any)
	if !ok || len(list) != dst.Len() {
		return fmt.Errorf("want list of %d, got %v", dst.Len(), src)
	}
	for i, item := range list {
		if err := Decode(dst.Index(i), item); err != nil {
			return fmt.Errorf("[%d]: %w", i, err)
		}
	}
	return nil
}

func shortType(t reflect.Type) string {
	s := strings.TrimPrefix(t.String(), "*")
	if i := strings.LastIndex(s, "/"); i >= 0 {
		s = s[i+1:]
	}
	return s
}

func reflectValue(v any) reflect.Value { return reflect.ValueOf(v) }
