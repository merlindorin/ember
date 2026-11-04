package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
)

// node is a JSON value that keeps its keys in order, so a tileset saved
// back reads as Tiled wrote it, but for what was edited.
type node struct {
	// object holds keys and vals; array holds items; anything else is a
	// scalar, kept as its JSON text.
	object bool
	array  bool
	keys   []string
	vals   []*node
	items  []*node
	raw    string
}

// parse reads b as JSON, keeping its keys' order and its numbers' text.
func parse(b []byte) (*node, error) {
	d := json.NewDecoder(bytes.NewReader(b))
	d.UseNumber()
	n, err := parseValue(d)
	if err != nil {
		return nil, err
	}
	if _, err = d.Token(); !errors.Is(err, io.EOF) {
		return nil, fmt.Errorf("tileset: trailing data")
	}
	return n, nil
}

func parseValue(d *json.Decoder) (*node, error) {
	t, err := d.Token()
	if err != nil {
		return nil, err
	}
	switch v := t.(type) {
	case json.Delim:
		switch v {
		case '{':
			return parseObject(d)
		case '[':
			return parseArray(d)
		}
	case json.Number:
		return &node{raw: v.String()}, nil
	case nil:
		return &node{raw: "null"}, nil
	}
	return scalar(t), nil
}

func parseObject(d *json.Decoder) (*node, error) {
	n := &node{object: true}
	for d.More() {
		k, err := d.Token()
		if err != nil {
			return nil, err
		}
		val, err := parseValue(d)
		if err != nil {
			return nil, err
		}
		key, _ := k.(string) // the decoder reads an object's keys as strings
		n.keys, n.vals = append(n.keys, key), append(n.vals, val)
	}
	_, err := d.Token()
	return n, err
}

func parseArray(d *json.Decoder) (*node, error) {
	n := &node{array: true}
	for d.More() {
		val, err := parseValue(d)
		if err != nil {
			return nil, err
		}
		n.items = append(n.items, val)
	}
	_, err := d.Token()
	return n, err
}

// scalar is v as a JSON scalar node.
func scalar(v any) *node {
	var b bytes.Buffer
	e := json.NewEncoder(&b)
	e.SetEscapeHTML(false)
	_ = e.Encode(v)
	return &node{raw: strings.TrimSpace(b.String())}
}

// encode writes n as Tiled does: one key or item a line, a space of
// indent a level.
func (n *node) encode(b *bytes.Buffer, depth int) {
	pad := func(d int) { b.WriteString(strings.Repeat(" ", d)) }
	switch {
	case n.object:
		if len(n.keys) == 0 {
			b.WriteString("{}")
			return
		}
		b.WriteString("{\n")
		for i, k := range n.keys {
			pad(depth + 1)
			b.WriteString(scalar(k).raw)
			b.WriteString(": ")
			n.vals[i].encode(b, depth+1)
			if i < len(n.keys)-1 {
				b.WriteByte(',')
			}
			b.WriteByte('\n')
		}
		pad(depth)
		b.WriteByte('}')
	case n.array:
		if len(n.items) == 0 {
			b.WriteString("[]")
			return
		}
		b.WriteString("[\n")
		for i, it := range n.items {
			pad(depth + 1)
			it.encode(b, depth+1)
			if i < len(n.items)-1 {
				b.WriteByte(',')
			}
			b.WriteByte('\n')
		}
		pad(depth)
		b.WriteByte(']')
	default:
		b.WriteString(n.raw)
	}
}

// bytes is n as a file: encoded, ending with a newline when newline,
// as the file it was read from did (Tiled writes none).
func (n *node) bytes(newline bool) []byte {
	var b bytes.Buffer
	n.encode(&b, 0)
	if newline {
		b.WriteByte('\n')
	}
	return b.Bytes()
}

// get is the value of key, nil when n has none.
func (n *node) get(key string) *node {
	for i, k := range n.keys {
		if k == key {
			return n.vals[i]
		}
	}
	return nil
}

// set gives key val, in its place when n has it, else last.
func (n *node) set(key string, val *node) {
	for i, k := range n.keys {
		if k == key {
			n.vals[i] = val
			return
		}
	}
	n.keys, n.vals = append(n.keys, key), append(n.vals, val)
}

// remove drops key.
func (n *node) remove(key string) {
	for i, k := range n.keys {
		if k == key {
			n.keys = append(n.keys[:i], n.keys[i+1:]...)
			n.vals = append(n.vals[:i], n.vals[i+1:]...)
			return
		}
	}
}

// str and num read a scalar node.
func (n *node) str() string {
	if n == nil {
		return ""
	}
	var s string
	_ = json.Unmarshal([]byte(n.raw), &s)
	return s
}

func (n *node) num() float64 {
	if n == nil {
		return 0
	}
	var f float64
	_ = json.Unmarshal([]byte(n.raw), &f)
	return f
}

// clone is a deep copy of n.
func (n *node) clone() *node {
	if n == nil {
		return nil
	}
	c := &node{object: n.object, array: n.array, raw: n.raw, keys: append([]string(nil), n.keys...)}
	for _, v := range n.vals {
		c.vals = append(c.vals, v.clone())
	}
	for _, it := range n.items {
		c.items = append(c.items, it.clone())
	}
	return c
}
