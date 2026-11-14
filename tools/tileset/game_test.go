package main

import (
	"os"
	"testing"

	"github.com/lantervale/ember/collide/tiles"
	"github.com/lantervale/ember/twod"
	"github.com/lantervale/ember/twod/light"
)

// engineDeclared is what the tool declares with no -props: the engine's
// properties alone.
//
//nolint:gochecknoglobals // the engine's alone, kept before TestMain adds the game's
var engineDeclared = declared

// The fixture's tilesets are a game's, so the tests run as the game runs
// the tool: with its properties, testdata/props.json, through -props.
func TestMain(m *testing.M) {
	defs, err := readProperties("testdata/props.json")
	if err != nil {
		panic(err)
	}
	declared = append(engineTileProperties(), defs...)
	os.Exit(m.Run())
}

func TestWithoutPropsTheToolKnowsTheEnginesPropertiesOnly(t *testing.T) {
	engine := engineTileProperties()
	if len(engineDeclared) != len(engine) {
		t.Fatalf("the engine's %d properties, no game's: %d", len(engine), len(engineDeclared))
	}
	for i, d := range engine {
		if engineDeclared[i].Name != d.Name {
			t.Fatalf("property %d is %s, not the engine's %s", i, engineDeclared[i].Name, d.Name)
		}
	}
}

func TestTheEnginesPropertiesAreTwodsCollideTilesAndLightsEachOnce(t *testing.T) {
	names := map[string]int{}
	for _, d := range engineTileProperties() {
		names[d.Name]++
	}
	for _, d := range twod.Declared(tiles.Properties(), light.TileProperties()) {
		if names[d.Name] != 1 {
			t.Fatalf("%s is declared %d times, once wanted", d.Name, names[d.Name])
		}
	}
	for name, n := range names {
		if n != 1 {
			t.Fatalf("%s is declared %d times, once wanted", name, n)
		}
	}
}

func TestTheOpenDialogStartsWhereItWasThenInDirThenHere(t *testing.T) {
	dir := t.TempDir()
	defer func(d string) { startDir = d }(startDir)
	startDir = ""
	if got := (state{}).startFolder(); got == dir {
		t.Fatal("no -dir: here")
	}
	startDir = dir
	if got := (state{}).startFolder(); got != dir {
		t.Fatalf("-dir before any tileset is opened: %s", got)
	}
	last := t.TempDir()
	if got := (state{Folder: last}).startFolder(); got != last {
		t.Fatalf("then the folder it was last in: %s", got)
	}
}
