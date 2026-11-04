package nav

import "github.com/merlindorin/ember/geom"

// Route is a way to walk, planned again when the grid changes or the way
// ahead closes.
type Route struct {
	m       *Map
	to      geom.Vec2
	points  []geom.Vec2
	version uint64
	reaches bool
}

// Route plans the way from from to to.
func (m *Map) Route(from, to geom.Vec2) *Route {
	r := &Route{m: m, to: to}
	r.plan(from)
	return r
}

func (r *Route) plan(from geom.Vec2) {
	r.points, r.reaches = r.m.Find(from, r.to)
	if v, ok := r.m.grid.(versioned); ok {
		r.version = v.Version()
	}
}

// To is where the route goes.
func (r *Route) To() geom.Vec2 { return r.to }

// Reaches reports whether the route gets to To, rather than as near as it
// can.
func (r *Route) Reaches() bool { return r.reaches }

// Waypoints is what is left to walk.
func (r *Route) Waypoints() []geom.Vec2 { return r.points }

// Done reports whether the route has been walked.
func (r *Route) Done() bool { return len(r.points) == 0 }

// Toward is the unit direction from at to the next waypoint, moving on to
// the following one once at is within reach of it; zero once walked. It
// plans again first when the grid changed or the way to the next waypoint
// is blocked.
func (r *Route) Toward(at geom.Vec2, reach float32) geom.Vec2 {
	for len(r.points) > 0 && at.Dist(r.points[0]) <= reach {
		r.points = r.points[1:]
	}
	if v, ok := r.m.grid.(versioned); ok && v.Version() != r.version {
		r.plan(at)
	} else if len(r.points) > 0 && !r.m.Clear(at, r.points[0]) {
		r.plan(at)
	}
	for len(r.points) > 0 && at.Dist(r.points[0]) <= reach {
		r.points = r.points[1:]
	}
	if len(r.points) == 0 {
		return geom.Vec2{}
	}
	return r.points[0].Sub(at).Normalize()
}
