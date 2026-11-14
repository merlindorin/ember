package main

import (
	"github.com/lantervale/ember/ui"
	"github.com/lantervale/ember/ui/kit"
)

// button is a default, a primary and a destructive button, and one disabled.
func button(*gallery) *ui.Node {
	return kit.Row(kit.Gap(12),
		kit.Button("Cancel"),
		kit.Button("Save", kit.Primary()),
		kit.Button("Delete", kit.Destructive()),
		kit.Button("Export", ui.Disabled()),
	)
}
