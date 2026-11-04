// Package layout lays out ui's box tree: block, inline text, flex and grid formatting
// contexts with intrinsic sizes, overflow and positioning, as CSS defines them.
//
// The algorithms are ported from Taffy (github.com/DioxusLabs/taffy), MIT licensed; its
// notice is in LICENSE-taffy, and testdata/taffy holds its fixtures, generated from Chrome.
// The package is pure Go and never imports twod: text measures through an interface.
package layout
