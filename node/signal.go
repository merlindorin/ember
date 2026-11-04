package node

import (
	"fmt"
	"reflect"
	"slices"
	"sync"
	"unsafe"
)

// Flags change how a connection calls: later in the frame, saved with
// the scene, or once.
type Flags uint8

const (
	Deferred Flags = 1
	Persist  Flags = 2
	OneShot  Flags = 4
)

var flagWords = FlagWords[Flags]{ //nolint:gochecknoglobals // word table
	None: "none", Bits: []string{"deferred", "persist", "one_shot"},
}

func (v Flags) MarshalText() ([]byte, error)  { return flagWords.Text(v) }
func (v *Flags) UnmarshalText(b []byte) error { return flagWords.Parse(v, b) }

// Signal is a node's typed signal. Its name is the field's
// `signal:"name"` tag, or an exported field's name in snake_case: an
// unexported field is a signal only when tagged, so a type can keep its
// signals private and still connect in scenes.
type Signal[T any] struct {
	node  *Node
	conns []*connection[T]
}

type connection[T any] struct {
	fn     func(T)
	flags  Flags
	target *Node
	method string
}

// Connect calls fn on every Emit until the returned function disconnects it.
func (s *Signal[T]) Connect(fn func(T), flags Flags) func() {
	c := &connection[T]{fn: fn, flags: flags}
	s.conns = append(s.conns, c)
	return func() { s.drop(c) }
}

func (s *Signal[T]) drop(c *connection[T]) {
	s.conns = slices.DeleteFunc(s.conns, func(x *connection[T]) bool { return x == c })
}

// Emit calls the connections in order. Deferred ones run at the end of the
// frame when the target, or else the emitting node, is in a tree, and right
// away otherwise.
func (s *Signal[T]) Emit(v T) {
	for _, c := range slices.Clone(s.conns) {
		if c.flags&OneShot != 0 {
			s.drop(c)
		}
		if c.flags&Deferred == 0 {
			c.fn(v)
			continue
		}
		host := c.target
		if host == nil {
			host = s.node
		}
		if host != nil && host.tree != nil {
			host.tree.deferCall(func() { c.fn(v) })
		} else {
			c.fn(v)
		}
	}
}

func (s *Signal[T]) bind(n *Node) { s.node = n }

func (s *Signal[T]) connectMethod(m method, target *Node, flags Flags) error {
	arg := reflect.TypeFor[T]()
	var fn func(T)
	switch t := m.Type(); {
	case t.NumIn() == 1 && t.In(0) == arg:
		f, ok := m.Interface().(func(T))
		if !ok {
			panic(fmt.Sprintf("node: %s is %s, not func(%s)", m.name, t, arg))
		}
		fn = f
	case t.NumIn() == 0 && arg == reflect.TypeFor[struct{}]():
		call, ok := m.Interface().(func())
		if !ok {
			panic(fmt.Sprintf("node: %s is %s, not func()", m.name, t))
		}
		fn = func(T) { call() }
	default:
		return fmt.Errorf("takes %s, the signal passes %s", t, arg)
	}
	s.conns = append(s.conns, &connection[T]{fn: fn, flags: flags, target: target, method: m.name})
	return nil
}

func (s *Signal[T]) persisted(signal string) []MethodConnection {
	var out []MethodConnection
	for _, c := range s.conns {
		if c.flags&Persist != 0 && c.method != "" {
			out = append(out, MethodConnection{Signal: signal, Target: c.target, Method: c.method, Flags: c.flags})
		}
	}
	return out
}

type signaler interface {
	bind(*Node)
	connectMethod(m method, target *Node, flags Flags) error
	persisted(signal string) []MethodConnection
}

type method struct {
	reflect.Value
	name string
}

// MethodConnection is a connection to a method by name, as scenes save it.
type MethodConnection struct {
	Signal string
	Target *Node
	Method string
	Flags  Flags
}

// Connections lists e's persisted method connections, by signal in field
// order.
func Connections(e Noded) []MethodConnection {
	all := signals(e)
	var out []MethodConnection
	for _, name := range SignalNames(e) {
		out = append(out, all[name].persisted(name)...)
	}
	return out
}

type signalField struct {
	name  string
	index []int
}

var signalFields sync.Map //nolint:gochecknoglobals // cache of each type's signal fields

func fieldsOf(t reflect.Type) []signalField {
	if f, ok := signalFields.Load(t); ok {
		fields, _ := f.([]signalField)
		return fields
	}
	var out []signalField
	var scan func(reflect.Type, []int)
	scan = func(t reflect.Type, at []int) {
		for i := range t.NumField() {
			f := t.Field(i)
			index := append(slices.Clone(at), i)
			switch {
			case reflect.PointerTo(f.Type).Implements(reflect.TypeFor[signaler]()):
				name := f.Tag.Get("signal")
				if name == "" {
					if !f.IsExported() {
						continue
					}
					name = snake(f.Name)
				}
				out = append(out, signalField{name: name, index: index})
			case f.Anonymous && f.Type.Kind() == reflect.Struct:
				scan(f.Type, index)
			case f.Anonymous && f.Type.Kind() == reflect.Pointer && f.Type.Elem().Kind() == reflect.Struct:
				scan(f.Type.Elem(), index)
			}
		}
	}
	scan(t, nil)
	signalFields.Store(t, out)
	return out
}

func signals(e Noded) map[string]signaler {
	v := reflect.ValueOf(e)
	if v.Kind() != reflect.Pointer || v.Elem().Kind() != reflect.Struct {
		return nil
	}
	out := map[string]signaler{}
	for _, f := range fieldsOf(v.Elem().Type()) {
		if fv, err := v.Elem().FieldByIndexErr(f.index); err == nil {
			s, _ := reflect.NewAt(fv.Type(), unsafe.Pointer(fv.UnsafeAddr())).Interface().(signaler)
			out[f.name] = s
		}
	}
	return out
}

func bindSignals(n *Node) {
	for _, s := range signals(n.self) {
		s.bind(n)
	}
}

// SignalNames lists e's signals by name, in field order.
func SignalNames(e Noded) []string {
	var out []string
	for _, f := range fieldsOf(reflect.Indirect(reflect.ValueOf(e)).Type()) {
		out = append(out, f.name)
	}
	return out
}

// ConnectMethod connects from's signal, by name, to the method of to
// named method, checking that the method takes the signal's value (or
// nothing, for a struct{} signal).
func ConnectMethod(from Noded, signal string, to Noded, name string, flags Flags) error {
	s, ok := signals(from)[signal]
	if !ok {
		return fmt.Errorf("node: %s has no signal %s", from.GetSceneNode().Name(), signal)
	}
	m := reflect.ValueOf(to).MethodByName(name)
	if !m.IsValid() {
		return fmt.Errorf("node: %s has no method %s", to.GetSceneNode().Name(), name)
	}
	if err := s.connectMethod(method{m, name}, to.GetSceneNode(), flags); err != nil {
		return fmt.Errorf("node: %s.%s %w (%s)", to.GetSceneNode().Name(), name, err, signal)
	}
	return nil
}
