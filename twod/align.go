package twod

import "github.com/lantervale/ember/node"

// Align is where text sits across the width it is given: a ui label's
// text in its rectangle.
type Align int

const (
	AlignLeft Align = iota
	AlignCenter
	AlignRight
)

//nolint:gochecknoglobals // word table
var alignWords = node.Words[Align]{"left", "center", "right"}

func (a Align) MarshalText() ([]byte, error)  { return alignWords.Text(a) }
func (a *Align) UnmarshalText(b []byte) error { return alignWords.Parse(a, b) }

// Name names an entity: its node's name, part of the automation IDs of
// what it holds.
func Name(id string) Option { return func(e Entity) { e.GetSceneNode().SetName(id) } }
