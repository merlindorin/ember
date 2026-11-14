// Time: the World's clock. Scale slows simulated time down or pauses it,
// and Alpha says how far a frame sits between two fixed steps, so a moving
// thing is drawn smoothly between them.
//
//	go run ./examples/08-time
package main

import (
	"fmt"
	"os"

	"github.com/lantervale/ember/world"
)

func main() {
	ball := &Ball{}
	w := world.New().AddSystems(ball)

	oneSecond(w, ball, "normal speed", 1)
	oneSecond(w, ball, "half speed", 0.5)
	oneSecond(w, ball, "paused", 0)

	w.Time().Scale = 1
	for range 5 {
		w.Frame(1.0 / 144)
		alpha := w.Time().Alpha
		fmt.Fprintf(os.Stdout, "a frame at 144 fps: alpha %.2f, ball drawn at %.2f\n", alpha, ball.Drawn(alpha))
	}
}

// oneSecond runs sixty frames of a sixtieth of a second at scale.
func oneSecond(w *world.World, ball *Ball, name string, scale float32) {
	w.Time().Scale = scale
	start := w.Time().Elapsed
	for range 60 {
		w.Frame(1.0 / 60)
	}
	fmt.Fprintf(os.Stdout, "%s: %.2f s simulated, ball at %.0f\n", name, w.Time().Elapsed-start, ball.x)
}
