package tiles

import (
	"github.com/merlindorin/ember/collide"
	"github.com/merlindorin/ember/nav"
)

// What a Layer and an Area are to collide and nav.
var (
	_ collide.SolidAppender = (*Layer)(nil)
	_ nav.Grid              = (*Layer)(nil)
	_ nav.Grid              = (*Area)(nil)
	_ nav.Roomy             = (*Area)(nil)
)
