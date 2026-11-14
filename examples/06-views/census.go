package main

import "github.com/lantervale/ember/world"

// Census is a view: the World keeps it in sync with every burning entity,
// but never updates it.
type Census struct{ world.Group[Burning] }
