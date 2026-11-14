// Fixed step: a slow and a fast machine each run one second. Update runs once
// per frame, FixedUpdate sixty times a second on both.
//
//	go run ./examples/07-fixed-step
package main

import (
	"fmt"
	"os"

	"github.com/lantervale/ember/world"
)

func main() {
	oneSecondAt(30)
	oneSecondAt(120)
}

func oneSecondAt(fps int) {
	steps, frames := &Steps{}, &Frames{}

	w := world.New().AddSystems(steps, frames)
	for range fps {
		w.Frame(1 / float32(fps))
	}

	fmt.Fprintf(os.Stdout, "%d fps: %d frames, %d fixed steps\n", fps, frames.count, steps.count)
}
