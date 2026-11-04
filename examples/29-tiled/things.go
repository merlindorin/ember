package main

import (
	"fmt"

	"github.com/merlindorin/ember/collide"
	"github.com/merlindorin/ember/geom"
	"github.com/merlindorin/ember/twod"
)

// spawn turns the level's objects into the hero, at the start, and solid
// pillars, standing at their points in things, which sorts them by y.
func spawn(things *twod.Layer, objects []twod.TiledObject) (*twod.Sprite, error) {
	tall, err := twod.LoadTexture(files, "objects.png", twod.Pixelated())
	if err != nil {
		return nil, err
	}
	var hero *twod.Sprite
	for _, o := range objects {
		at := geom.V2(o.Rect.X, o.Rect.Y)
		switch o.Type {
		case "start":
			if hero, err = newHero(at); err != nil {
				return nil, err
			}
			things.Add(hero)
		case "pillar":
			pillar := twod.NewSprite(tall,
				twod.Region(geom.Rect{X: 16, W: 16, H: 32}),
				twod.At(at.X, at.Y),
				twod.Scale(2, 2),
				twod.Pivot(0.5, 1),
			)
			pillar.Add(collide.NewBody(collide.Box(20, 12), collide.Offset(0, -6)))
			things.Add(pillar)
		}
	}
	if hero == nil {
		return nil, fmt.Errorf("the level has no start")
	}
	return hero, nil
}
