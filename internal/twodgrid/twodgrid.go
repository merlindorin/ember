// Package twodgrid turns off twod's grids of placements, so a test can
// compare what twod draws with them and without.
package twodgrid

// Off makes every layer visit all its children, as before grids.
var Off bool //nolint:gochecknoglobals // a process-wide switch a benchmark test flips
