// Package cursorhint lets 2D drawing suggest a window's cursor, which the
// window shows unless the game chose one.
package cursorhint

// Show suggests cursor c, a platform.Cursor, for win, a *platform.Window;
// package platform sets it.
//
//nolint:gochecknoglobals // a hook another package fills at init, keeping the imports acyclic
var Show func(win any, c int)
