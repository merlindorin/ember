package server_test

import (
	"os"
	"os/exec"
	"strings"
	"testing"

	"github.com/merlindorin/ember/automation"
	"github.com/merlindorin/ember/geom"
	"github.com/merlindorin/ember/scene"
	"github.com/merlindorin/ember/server"
	"github.com/merlindorin/ember/world"
)

func TestServerStaysPureGo(t *testing.T) {
	cmd := exec.Command("go", "list", "-deps", ".")
	cmd.Env = append(os.Environ(), "CGO_ENABLED=0")
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("go list with CGO_ENABLED=0: %v\n%s", err, out)
	}
	for _, dep := range strings.Fields(string(out)) {
		for _, banned := range []string{
			"/gpu", "/internal/wgpu", "/platform/desktop", "/internal/audiodevice", "go-gl/", "ebitengine/oto",
		} {
			if strings.Contains(dep, banned) {
				t.Errorf("server depends on %s", dep)
			}
		}
		if dep == "C" {
			t.Error("server depends on cgo")
		}
	}
}

type counter struct{ steps int }

func (c *counter) Update(f world.Frame) {
	c.steps++
	if c.steps == 3 {
		f.Quit()
	}
}
func (*counter) Remove(world.Identifier) {}
func (*counter) Phase() world.Phase      { return world.FixedUpdate }
func (*counter) Priority() int           { return 0 }
func (*counter) Name() string            { return "test.counter" }

func TestRunTicksUntilQuit(t *testing.T) {
	c := &counter{}
	w := server.New().AddSystems(c)
	w.Time().FixedDelta = 1.0 / 240
	if err := w.Run(); err != nil {
		t.Fatal(err)
	}
	if c.steps != 3 {
		t.Fatalf("fixed steps = %d, want 3", c.steps)
	}
}

type body struct {
	*world.BasicEntity
	scene.Transform
}

func TestWorldPositionsFollowTransforms(t *testing.T) {
	parent := &body{BasicEntity: world.NewBasic(), Transform: scene.At(10, 0, 0)}
	child := &body{BasicEntity: world.NewBasic(), Transform: scene.At(0, 2, 0)}
	parent.AppendChild(child)

	server.New().AddEntities(parent).RunFrames(1)

	if got, want := child.WorldPosition(), geom.V3(10, 2, 0); got != want {
		t.Fatalf("child world position = %v, want %v", got, want)
	}
}

func TestAutomationIsTheGamesOwn(t *testing.T) {
	auto := automation.New(automation.Addr("127.0.0.1:0"))
	server.New(server.Automation(auto))
	defer auto.Stop()
	if auto.Addr() == "" {
		t.Fatal("the World never built the automation it was given")
	}
}
