// Package assets lets export and test pages reach into a Document's images, as fonts.Of lets
// it read its faces.
package assets

import (
	"github.com/lantervale/ember/geom"
	"github.com/lantervale/ember/ui/internal/box"
	"github.com/lantervale/ember/ui/internal/dom"
)

// Of is a ui.Document's image by url, as css resolves it: a scheme url
// such as ember:select-arrow, or a path in its stylesheets' file systems.
var Of func(doc any, url string) ([]byte, bool) //nolint:gochecknoglobals // a hook ui sets

// Styled computes a ui.Document's styles without laying it out, for
// export to read them.
var Styled func(doc any) //nolint:gochecknoglobals // a hook ui sets

// Pin holds pseudo-classes on a node of a ui.Document whatever the
// pointer and the focus do, for test pages' still pictures.
var Pin func(doc any, n *dom.Node, s dom.State) //nolint:gochecknoglobals // a hook ui sets

// Modal reports whether a dialog of a ui.Document is open modal and its
// layer, higher for the one shown modal later, for export to open it so in
// Chrome.
var Modal func(doc any, n *dom.Node) (layer uint64, ok bool) //nolint:gochecknoglobals // a hook ui sets

// Tree is a ui.Document's last laid-out box tree, for the kit's
// alignment test to read boxes and baselines from.
var Tree func(doc any) *box.Tree //nolint:gochecknoglobals // a hook ui sets

// Viewport is the size a ui.Document was last laid out at, the window
// its boxes may be cut by.
var Viewport func(doc any) geom.Vec2 //nolint:gochecknoglobals // a hook ui sets

// Path is a node's automation path in a ui.Document, "" for one
// automation doesn't record, for tests to name what they report.
var Path func(doc any, n *dom.Node) string //nolint:gochecknoglobals // a hook ui sets
