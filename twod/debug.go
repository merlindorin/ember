package twod

import (
	"fmt"

	"github.com/merlindorin/ember/color"
	"github.com/merlindorin/ember/geom"
	"github.com/merlindorin/ember/input"
	"github.com/merlindorin/ember/internal/draw"
	"github.com/merlindorin/ember/internal/slot"
	"github.com/merlindorin/ember/internal/wm"
	"github.com/merlindorin/ember/render"
	"github.com/merlindorin/ember/world"
)

// debugOverlay is the resource behind the F3 profiler: frame graph, entity
// and draw counts, the slowest systems, and the costs recorded in the
// World's Stats, such as twod's collect and draw and the GPU's time per
// pass. F4 toggles gizmos.
type debugOverlay struct {
	Visible bool

	gizmos world.Group[render.GizmoSet]
}

// ShowProfiler shows or hides w's F3 profiler, as F3 toggles it.
func ShowProfiler(w *world.World, on bool) {
	if ws := slot.Get[windows](w); ws != nil {
		ws.overlay.Visible = on
	}
}

// ProfilerShown is whether w's F3 profiler shows.
func ProfilerShown(w *world.World) bool {
	ws := slot.Get[windows](w)
	return ws != nil && ws.overlay.Visible
}

func (d *debugOverlay) update(w *world.World) {
	in := wm.Input(w)
	if in.Pressed(input.KeyF3) {
		d.Visible = !d.Visible
	}
	if in.Pressed(input.KeyF4) {
		for g := range d.gizmos.All() {
			g.GetGizmos().Enabled = !g.GetGizmos().Enabled
		}
	}
	if d.Visible {
		win := wm.Primary(w)
		ld := draw.LightDebugOf(w, win)
		switch {
		case in.Pressed(input.KeyF5):
			ld.View = (ld.View + 1) % (draw.LightViewShadows + 1)
			draw.SetLightDebug(w, win, ld)
		case in.Pressed(input.KeyF6):
			ld.Outlines = !ld.Outlines
			draw.SetLightDebug(w, win, ld)
		}
		d.draw(w)
	} else if draw.DebugPanel(w) != (geom.Rect{}) {
		draw.SetDebugPanel(w, geom.Rect{})
	}
}

func (d *debugOverlay) draw(w *world.World) {
	u := CanvasFor(w, nil)
	st := w.Stats()
	slowest, recorded := st.Slowest(8), st.Recorded()
	head := append(debugHead(st), debugLights(w)...)
	costs := debugCosts(slowest, recorded, st.Counted())
	r := geom.Rect{X: 12, Y: 12, W: 300, H: 16 + 17*float32(len(head)+len(costs)) + 54}
	if draw.DebugPanel(w) != r {
		draw.SetDebugPanel(w, r)
	}
	u.RoundRect(r, 6, color.RGBA(0, 0, 0, 0.75))
	y := r.Y + 8
	line := func(format string, args ...any) {
		u.Text(
			geom.V2(r.X+10, y),
			fmt.Sprintf(format, args...),
			Typeface(Mono()),
			FontSize(13),
			Color(color.RGB(0.75, 0.9, 0.75)),
		)
		y += 17
	}
	for _, l := range head {
		line("%s", l)
	}

	graph := geom.Rect{X: r.X + 10, Y: y + 2, W: r.W - 20, H: 44}
	u.Rect(graph, color.RGBA(1, 1, 1, 0.05))
	hist := st.FrameHistory()
	bw := graph.W / float32(len(hist))
	for i, v := range hist {
		h := geom.Clamp(v/33.3, 0, 1) * graph.H
		c := color.RGB(0.3, 0.9, 0.4)
		if v > 17.5 {
			c = color.RGB(1, 0.7, 0.2)
		}
		if v > 34 {
			c = color.RGB(1, 0.25, 0.2)
		}
		u.Rect(geom.Rect{X: graph.X + float32(i)*bw, Y: graph.Y + graph.H - h, W: max(bw-0.5, 0.5), H: h}, c)
	}
	u.Rect(geom.Rect{X: graph.X, Y: graph.Y + graph.H/2, W: graph.W, H: 1}, color.RGBA(1, 1, 1, 0.2))
	y = graph.Y + graph.H + 8
	for _, l := range costs {
		line("%s", l)
	}
}

// debugHead is the overlay's lines over its graph: the frame, the work in
// it apart from waiting for the display, and its counts.
func debugHead(st *world.Stats) []string {
	return []string{
		fmt.Sprintf("FPS %5.0f   frame %5.2fms", st.FPS, ms(st.FrameTime)),
		fmt.Sprintf("work %5.2fms  (~%.0f fps possible)", ms(st.Work()), 1e3/max(ms(st.Work()), 1e-3)),
		fmt.Sprintf("tick %5.2fms  render %5.2fms", ms(st.TickTime), ms(st.RenderTime)),
		fmt.Sprintf("entities %d  draws %d  tris %dk", st.Entities(), st.DrawCalls, st.Triangles/1000),
		fmt.Sprintf(
			"2D draws %d  binds %d  verts %dk  passes %d",
			st.Draws2D,
			st.Binds2D,
			st.Vertices2D/1000,
			st.Passes2D,
		),
		"F4 gizmos",
	}
}

// debugLights is the overlay's lines on the primary window's lighting: its
// lights, those casting shadows, against how many it draws, and its
// occluders; and the keys of its debug views.
func debugLights(w *world.World) []string {
	li := draw.LightingFor(w, wm.Primary(w))
	if li == nil {
		return nil
	}
	cast := 0
	for _, l := range li.Lights {
		if !l.XRay {
			cast++
		}
	}
	occluders := 0
	if li.Occluders != nil {
		occluders = len(li.Occluders.Vertices) / 4
	}
	static := 0
	if li.Static != nil {
		static = li.Static.Count
	}
	ld := draw.LightDebugOf(w, wm.Primary(w))
	outlines := "off"
	if ld.Outlines {
		outlines = "on"
	}
	budget := ""
	if li.Budget > 0 {
		budget = fmt.Sprintf(", budget %d (%d fading)", li.Budget, li.Fading)
	}
	if n := len(li.Plan.Pictures); n > 0 {
		budget += fmt.Sprintf(", pictures %d of %d", n, draw.MaxPictures)
		if f := len(li.Plan.Fallen); f > 0 {
			budget += fmt.Sprintf(" (%d fallen back)", f)
		}
	}
	return []string{
		fmt.Sprintf(
			"lights %d (%d cast) of %d%s, %d static  occluders %d",
			len(li.Lights),
			cast,
			draw.MaxShadowLights,
			budget,
			static,
			occluders,
		),
		fmt.Sprintf("F5 view: %s  F6 outlines: %s", ld.View, outlines),
	}
}

// debugCosts is the overlay's lines under its graph: the slowest systems,
// then the recorded costs, then the counters, if any.
func debugCosts(slowest, recorded []*world.SystemStat, counted []*world.Counter) []string {
	var out []string
	for _, s := range slowest {
		out = append(out, fmt.Sprintf("%-24.24s %6.3fms", s.Name, ms(s.Time)))
	}
	out = append(out, "recorded")
	for _, s := range recorded {
		out = append(out, fmt.Sprintf("%-24.24s %6.3fms", s.Name, ms(s.Time)))
	}
	if len(counted) > 0 {
		out = append(out, "counted")
	}
	for _, c := range counted {
		out = append(out, fmt.Sprintf("%-24.24s %8d", c.Name, c.N))
	}
	return out
}

func ms[T ~int64](d T) float64 { return float64(d) / 1e6 }
