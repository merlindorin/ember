package twod

import "github.com/merlindorin/ember/world"

func (s *Sprite) Add(children ...world.BasicFace) *Sprite         { add(&s.Item, children); return s }
func (l *Line) Add(children ...world.BasicFace) *Line             { add(&l.Item, children); return l }
func (p *Polygon) Add(children ...world.BasicFace) *Polygon       { add(&p.Item, children); return p }
func (t *Text) Add(children ...world.BasicFace) *Text             { add(&t.Item, children); return t }
func (t *BitmapText) Add(children ...world.BasicFace) *BitmapText { add(&t.Item, children); return t }
func (p *NinePatch) Add(children ...world.BasicFace) *NinePatch   { add(&p.Item, children); return p }
func (l *Layer) Add(children ...world.BasicFace) *Layer           { add(&l.Item, children); return l }
func (c *Camera) Add(children ...world.BasicFace) *Camera         { add(&c.Item, children); return c }
func (m *TileMap) Add(children ...world.BasicFace) *TileMap       { add(&m.Item, children); return m }
func (p *Particles) Add(children ...world.BasicFace) *Particles   { add(&p.Item, children); return p }
