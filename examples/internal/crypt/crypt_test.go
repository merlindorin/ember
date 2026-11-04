package crypt //nolint:testpackage // it checks the unexported size of each frame

import (
	"io/fs"
	"testing"
)

func TestEveryPictureLoads(t *testing.T) {
	Floor()
	Things()
	for _, c := range []string{"elf_m", "goblin", "skelet", "imp", "orc_warrior"} {
		sheet, clips := Character(c)
		if sheet == nil || clips == nil {
			t.Fatalf("%s has no sheet", c)
		}
	}
	sprites := []string{
		"weapon_bow", "weapon_arrow", "weapon_knife", "flask_green", "flask_yellow",
		"flask_red", "flask_big_red", "flask_big_blue", "flask_big_yellow", "bomb_f0",
	}
	for _, s := range sprites {
		Sprite(s)
	}
}

func TestTheElfStandsInHisCells(t *testing.T) {
	if w, h := size("elf_m_run_anim_f0"); w != 16 || h != 28 {
		t.Fatalf("the elf is 16 by 28: %d by %d", w, h)
	}
}

func TestTheSoundsAreThere(t *testing.T) {
	for _, n := range []string{"music", "shoot", "hit", "kill", "powerup", "hurt", "gameover"} {
		if _, err := fs.Stat(Sounds(), n+".ogg"); err != nil {
			t.Errorf("%s.ogg: %v", n, err)
		}
	}
}
