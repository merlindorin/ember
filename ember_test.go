package ember_test

import (
	"testing"

	"github.com/lantervale/ember"
	"github.com/lantervale/ember/automation"
	"github.com/lantervale/ember/geom"
	"github.com/lantervale/ember/input"
	"github.com/lantervale/ember/internal/slot"
	"github.com/lantervale/ember/internal/wm"
	"github.com/lantervale/ember/platform"
	"github.com/lantervale/ember/world"
)

func TestTheFirstWindowAddedIsThePrimary(t *testing.T) {
	main := platform.NewWindow(platform.Title("ember · hello cube"), platform.Size(800, 600))
	tool := platform.NewWindow(platform.Title("map"))
	a := ember.New().AddEntities(main, tool)
	if wm.Primary(a) != main {
		t.Fatal("the game's first window is the primary one")
	}
	a.RunFrames(2)
	if wm.Primary(a) != main {
		t.Fatal("no default window is added when the game has one")
	}
}

func TestAWorldRunsFramesWithoutAWindow(_ *testing.T) {
	ember.New().RunFrames(5)
}

func TestWithHeadlessRunsOnTheTicker(t *testing.T) {
	w := ember.New(ember.Headless()).On(world.PostUpdate, "stop", func(f world.Frame) {
		if f.FixedFrame >= 3 {
			f.Quit()
		}
	})
	if err := w.Run(); err != nil || w.Time().FixedFrame < 3 {
		t.Fatalf("headless run: %v", err)
	}
}

func TestTheGamesAutomationRecordsTheWorld(t *testing.T) {
	auto := automation.New()
	a := ember.New(ember.Automation(auto))
	if slot.Get[automation.Registry](a) != auto.Registry {
		t.Fatal("widgets record into the game's automation")
	}
	if slot.Get[automation.Registry](ember.New()) == nil {
		t.Fatal("New makes one when the game passes none")
	}
}

func TestEveryWindowsInputBeginsItsFrameAsTheWorldRuns(t *testing.T) {
	main, tool := platform.NewWindow(), platform.NewWindow()
	var presses int
	var typed string
	var wheel float32
	w := ember.New().AddEntities(main, tool).On(world.Update, "read", func(world.Frame) {
		if tool.Input().Pressed(input.MouseLeft) {
			presses++
		}
		typed += string(tool.Input().Text())
		wheel += main.Input().Wheel().Y
	})
	tool.Input().SetMouseButton(input.MouseLeft, true)
	tool.Input().AddText('h')
	tool.Input().AddText('i')
	main.Input().AddWheel(geom.V2(0, -2))
	w.RunFrames(3)
	if presses != 1 || typed != "hi" || wheel != -2 {
		t.Fatalf("one press, typed once, turned once: %d %q %v", presses, typed, wheel)
	}
	tool.Input().SetMouseButton(input.MouseLeft, false)
	w.RunFrames(1)
	tool.Input().SetMouseButton(input.MouseLeft, true)
	w.Tick(1.0 / 60)
	if presses != 2 {
		t.Fatalf("Tick begins the frame too: %d presses", presses)
	}
}

func TestPadsConnectingAndGoingAreWorldEvents(t *testing.T) {
	win := platform.NewWindow()
	var joined world.Reader[input.GamepadConnected]
	var gone world.Reader[input.GamepadDisconnected]
	var names []string
	var ids []int
	w := ember.New().AddEntities(win).On(world.Update, "pads", func(f world.Frame) {
		for e := range joined.Read(f.World) {
			names = append(names, e.Name)
		}
		for e := range gone.Read(f.World) {
			ids = append(ids, e.ID)
		}
	})
	win.Input().ConnectGamepad(2, "PS5 Controller")
	w.RunFrames(3)
	win.Input().DisconnectGamepad(2)
	w.RunFrames(3)
	if len(names) != 1 || names[0] != "PS5 Controller" || len(ids) != 1 || ids[0] != 2 {
		t.Fatalf("one connection, one disconnection: %v %v", names, ids)
	}
}

func TestTouchesAreWorldEvents(t *testing.T) {
	win := platform.NewWindow()
	var started world.Reader[input.TouchStarted]
	var moved world.Reader[input.TouchMoved]
	var ended world.Reader[input.TouchEnded]
	var seen []string
	w := ember.New().AddEntities(win).On(world.Update, "touches", func(f world.Frame) {
		for e := range started.Read(f.World) {
			seen = append(seen, "started "+e.Touch.Source.String())
		}
		for range moved.Read(f.World) {
			seen = append(seen, "moved")
		}
		for range ended.Read(f.World) {
			seen = append(seen, "ended")
		}
	})
	in := win.Input()
	in.AddTouch(1, input.Trackpad, geom.V2(0.5, 0.5))
	w.RunFrames(1)
	in.MoveTouch(1, geom.V2(0.6, 0.5))
	w.RunFrames(2)
	in.EndTouch(1)
	w.RunFrames(2)
	want := []string{"started trackpad", "moved", "ended"}
	if len(seen) != 3 || seen[0] != want[0] || seen[1] != want[1] || seen[2] != want[2] {
		t.Fatalf("each change once: %v", seen)
	}
}
