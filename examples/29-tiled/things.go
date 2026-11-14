package main

import (
	"fmt"

	"github.com/lantervale/ember/collide"
	"github.com/lantervale/ember/geom"
	"github.com/lantervale/ember/twod"
)

// spawn turns the level's objects into the hero, at the start, and solid
// pillars, tile objects drawn as Tiled draws them, standing in things,
// which sorts them by y.
func spawn(things *twod.Layer, level *twod.Tiled, objects []twod.TiledObject) (*twod.Sprite, error) {
	var hero *twod.Sprite
	for _, o := range objects {
		switch o.Type {
		case "start":
			var err error
			if hero, err = newHero(geom.V2(o.Rect.X, o.Rect.Y)); err != nil {
				return nil, err
			}
			things.Add(hero)
		case "pillar":
			pillar := level.Sprite(o)
			pillar.Add(collide.NewBody(collide.Box(20, 12), collide.Offset(0, -6)))
			things.Add(pillar)
		}
	}
	if hero == nil {
		return nil, fmt.Errorf("the level has no start")
	}
	return hero, nil
}
