package main

import (
	"os"
	"testing"

	"github.com/merlindorin/ember/twod"
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
	declared = append(twod.EngineTileProperties(), defs...)
	os.Exit(m.Run())
}

func TestWithoutPropsTheToolKnowsTheEnginesPropertiesOnly(t *testing.T) {
	engine := twod.EngineTileProperties()
	if len(engineDeclared) != len(engine) {
		t.Fatalf("the engine's %d properties, no game's: %d", len(engine), len(engineDeclared))
	}
	for i, d := range engine {
		if engineDeclared[i].Name != d.Name {
			t.Fatalf("property %d is %s, not the engine's %s", i, engineDeclared[i].Name, d.Name)
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
