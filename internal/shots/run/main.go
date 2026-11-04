// Command run makes the guide's pictures from shots.Shots: each package
// built once, each shot run hidden and captured, only those whose key
// changed, Jobs at a time. task docs:shots runs it from ember's root.
//
//	go run ./internal/shots/run [-j 4] [-force] [-only 19-lights]
package main

import (
	"flag"
	"fmt"
	"os"
	"runtime"

	"github.com/merlindorin/ember/internal/shots"
)

func main() {
	o := shots.Options{Root: ".", Docs: "docs", Work: "dist/shots", Log: os.Stdout}
	flag.IntVar(&o.Jobs, "j", max(runtime.NumCPU()/2, 1), "shots at once")
	flag.BoolVar(&o.Force, "force", false, "remake every shot, whatever its key")
	flag.StringVar(&o.Only, "only", "", "make only the shots whose path contains this")
	flag.Parse()
	if err := shots.Run(shots.Shots, o); err != nil {
		fmt.Fprintln(os.Stderr, "shots:", err)
		os.Exit(1)
	}
}
