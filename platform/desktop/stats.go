package desktop

import (
	"fmt"
	"io"
	"os"
	"runtime/metrics"
	"runtime/pprof"
	"slices"
	"time"

	"github.com/merlindorin/ember/world"
)

// statsWarmup is how many frames EMBER_STATS skips before it averages.
const statsWarmup = 60

// A frame over spikeAt is reported with what it spent, up to spikesShown
// of them.
const (
	spikeAt     = 16 * time.Millisecond
	spikesShown = 30
)

// startProfile starts writing a CPU profile to path, for EMBER_CPUPROFILE.
func startProfile(path string) error {
	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("desktop: cpu profile: %w", err)
	}
	return pprof.StartCPUProfile(f)
}

// runStats averages a World's frame metrics over a run, after a warm-up,
// for EMBER_STATS: the frame, the tick and the render, each system and
// each cost recorded in the World's Stats, such as the GPU's passes.
type runStats struct {
	frames                int
	frame, tick, render   time.Duration
	worst                 time.Duration
	draws, binds, entries int
	entities              int
	costs                 map[string]time.Duration
	order                 []string
	spikes                int
	gc                    []metrics.Sample
}

// displayWait is how long the last frame waited for the primary window's
// texture, which under vsync is most of a refresh and no work of the
// game's.
func displayWait(w *world.World) time.Duration {
	for _, c := range w.Stats().Recorded() {
		if c.Name == "acquire" {
			return c.Spent
		}
	}
	return 0
}

// spike reports frame when it took over spikeAt besides waiting for the
// display: how long, the garbage
// collections it ran, and what it spent most on.
func (r *runStats) spike(out io.Writer, w *world.World, frame int, took time.Duration) {
	if r.gc == nil {
		r.gc = []metrics.Sample{{Name: "/gc/cycles/total:gc-cycles"}}
		metrics.Read(r.gc)
	}
	before := r.gc[0].Value.Uint64()
	metrics.Read(r.gc)
	took -= displayWait(w)
	if frame <= statsWarmup || took < spikeAt || r.spikes >= spikesShown {
		return
	}
	r.spikes++
	type cost struct {
		name string
		d    time.Duration
	}
	var costs []cost
	for _, c := range w.Timings() {
		costs = append(costs, cost{c.Name, c.Spent})
	}
	for _, c := range w.Stats().Recorded() {
		costs = append(costs, cost{c.Name, c.Spent})
	}
	slices.SortFunc(costs, func(a, b cost) int { return int(b.d - a.d) })
	fmt.Fprintf(
		out,
		"spike: frame %d took %.2f ms, %d GC cycles:",
		frame,
		float64(took)/1e6,
		r.gc[0].Value.Uint64()-before,
	)
	for _, c := range costs[:min(6, len(costs))] {
		fmt.Fprintf(out, " %s %.2f", c.name, float64(c.d)/1e6)
	}
	fmt.Fprintln(out)
}

// add takes one frame's metrics, dt its wall time; the first statsWarmup
// are skipped.
func (r *runStats) add(w *world.World, frame int, dt time.Duration) {
	if frame <= statsWarmup {
		return
	}
	r.worst = max(r.worst, dt)
	s := w.Stats()
	r.frames++
	r.frame += s.FrameTime
	r.tick += s.TickTime
	r.render += s.RenderTime
	r.draws, r.binds, r.entries, r.entities = s.Draws2D, s.Binds2D, s.Vertices2D, s.Entities()
	if r.costs == nil {
		r.costs = map[string]time.Duration{}
	}
	for _, c := range w.Timings() {
		r.cost(c.Name+" ("+c.Phase.String()+")", c.Time)
	}
	for _, c := range s.Recorded() {
		r.cost(c.Name, c.Time)
	}
}

func (r *runStats) cost(name string, d time.Duration) {
	if _, ok := r.costs[name]; !ok {
		r.order = append(r.order, name)
	}
	r.costs[name] += d
}

// print writes the averages, the costliest first, down to limit of them.
func (r *runStats) print(out io.Writer, limit int) {
	if r.frames == 0 {
		fmt.Fprintln(out, "stats: no frame past the warm-up")
		return
	}
	n := time.Duration(r.frames)
	ms := func(d time.Duration) float64 { return float64(d/n) / 1e6 }
	work := ms(r.tick) + ms(r.render)
	fmt.Fprintf(
		out,
		"stats over %d frames, %d entities: frame %.2f ms (%.0f fps), worst %.2f ms, "+
			"work %.2f ms (~%.0f fps possible): tick %.2f ms, render %.2f ms; "+
			"2D draws %d, binds %d, vertices %d\n",
		r.frames,
		r.entities,
		ms(r.frame),
		1e3/max(ms(r.frame), 1e-3),
		float64(r.worst)/1e6,
		work,
		1e3/max(work, 1e-3),
		ms(r.tick),
		ms(r.render),
		r.draws,
		r.binds,
		r.entries,
	)
	names := slices.Clone(r.order)
	slices.SortStableFunc(names, func(a, b string) int { return int(r.costs[b] - r.costs[a]) })
	for _, name := range names[:min(limit, len(names))] {
		fmt.Fprintf(out, "  %-36s %6.2f ms\n", name, ms(r.costs[name]))
	}
}
