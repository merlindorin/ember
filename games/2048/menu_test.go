package main

import (
	"testing"

	"github.com/lantervale/ember/audio"
	"github.com/lantervale/ember/games/internal/save"
	"github.com/lantervale/ember/geom"
	"github.com/lantervale/ember/input"
	"github.com/lantervale/ember/ui"
	"github.com/lantervale/ember/ui/align"
	"github.com/lantervale/ember/world"
)

func nodes(g *Game) []*ui.Node {
	var all []*ui.Node
	g.hud.Body().Walk(func(n *ui.Node) bool {
		all = append(all, n)
		return true
	})
	return all
}

// choose clicks a menu button by its text in whatever dialog shows it.
func choose(t *testing.T, g *Game, w *world.World, text string) {
	t.Helper()
	for _, n := range nodes(g) {
		if n.Text() != text || !n.HasClass("item") {
			continue
		}
		if r, ok := g.hud.BoxOf(n); ok && r.W > 0 {
			in := g.in
			in.SetMouse(r.Center())
			w.RunFrames(1)
			in.SetMouseButton(input.MouseLeft, true)
			w.RunFrames(1)
			in.SetMouseButton(input.MouseLeft, false)
			w.RunFrames(2)
			return
		}
	}
	t.Fatalf("no %q button showing", text)
}

func TestEscapeOpensTheMenuAndTheGameWaits(t *testing.T) {
	g, w, press := play(t)
	press(input.KeyEscape)
	if !g.menu.Open() {
		t.Fatal("Escape opens the menu")
	}
	if faults := align.Check(g.hud); len(faults) > 0 {
		t.Fatalf("the menu's layout: %v", faults)
	}
	g.board.Cells = [4][4]int{{0, 2}}
	press(input.KeyLeft)
	if g.board.Cells[0][1] != 2 {
		t.Fatal("the board waits while the menu is open")
	}
	choose(t, g, w, "Resume")
	press(input.KeyLeft)
	if g.menu.Open() || g.board.Cells[0][0] != 2 {
		t.Fatalf("Resume gives the arrows back: open %v, %v", g.menu.Open(), g.board.Cells)
	}
}

func TestTheLeaderboardShowsTheScoresKept(t *testing.T) {
	g, w, press := play(t)
	g.board.Cells = [4][4]int{{2, 4, 2, 4}, {4, 2, 4, 2}, {32, 4, 2, 4}, {8, 16, 8, 0}}
	g.board.Score = 1234
	press(input.KeyRight)
	g.hud.Close(g.over, "")
	w.RunFrames(1)
	if len(g.data.Scores) != 1 || g.data.Scores[0].Score != 1234 {
		t.Fatalf("the game over is on the board: %+v", g.data.Scores)
	}
	if kept, err := save.Load("2048"); err != nil || len(kept.Scores) != 1 {
		t.Fatalf("and in the file: %+v, %v", kept, err)
	}
	press(input.KeyEscape)
	choose(t, g, w, "Leaderboard")
	found := false
	for _, n := range nodes(g) {
		found = found || n.HasClass("points") && n.Text() == "1234"
	}
	if !found {
		t.Fatal("the leaderboard lists 1234")
	}
	choose(t, g, w, "Back")
	if !g.menu.Open() {
		t.Fatal("Back returns to the menu")
	}
}

func TestTheSoundSlidersChangeTheMixerAndAreKept(t *testing.T) {
	g, w, press := play(t)
	press(input.KeyEscape)
	choose(t, g, w, "Music & sound")
	slider := g.hud.ByID("music")
	r, ok := g.hud.BoxOf(slider)
	if !ok || r.W == 0 {
		t.Fatalf("the music slider shows: %v", r)
	}
	if g.hud.Value(slider) != "60" {
		t.Fatalf("it starts at the saved level: %q", g.hud.Value(slider))
	}
	in := g.in
	in.SetMouse(geom.V2(r.X+r.W*0.25, r.Y+r.H/2))
	w.RunFrames(1)
	in.SetMouseButton(input.MouseLeft, true)
	w.RunFrames(1)
	in.SetMouseButton(input.MouseLeft, false)
	w.RunFrames(2)
	if g.data.Music > 0.35 || g.data.Music < 0.15 || g.player.Mixer.Bus("music").Volume() != g.data.Music {
		t.Fatalf("a click a quarter along sets about 25%%: data %v, bus %v",
			g.data.Music, g.player.Mixer.Bus("music").Volume())
	}
	choose(t, g, w, "Music: on")
	if !g.data.MusicOff {
		t.Fatal("the switch turns the music off")
	}
	if kept, _ := save.Load("2048"); kept.Music != g.data.Music || !kept.MusicOff {
		t.Fatalf("the settings are kept: %+v", kept)
	}
	if faults := align.Check(g.hud); len(faults) > 0 {
		t.Fatalf("the page's layout: %v", faults)
	}
}

func TestQuitEndsTheGame(t *testing.T) {
	g, w, press := play(t)
	press(input.KeyEscape)
	choose(t, g, w, "Quit")
	if !w.ShouldQuit() {
		t.Fatal("Quit stops the World")
	}
}

func TestAMoveIsHeard(t *testing.T) {
	g, w, press := play(t)
	rec := audio.NewRecorder(g.player.Mixer)
	g.board.Cells = [4][4]int{{2, 2}}
	rec.Reset()
	press(input.KeyLeft)
	w.RunFrames(3)
	if rec.Peak(0, rec.Frames()) == 0 {
		t.Fatal("the slide and the merge sound")
	}
}
