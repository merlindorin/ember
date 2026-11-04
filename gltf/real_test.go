package gltf_test

import (
	"archive/zip"
	"os"
	"path/filepath"
	"testing"

	"github.com/merlindorin/ember/gltf"
)

// TestApocalypsePack loads a real asset pack when it is on this machine
// (EMBER_TEST_GLB_ZIP or ~/Downloads/Apocalypse_Free_glb.zip).
func TestApocalypsePack(t *testing.T) {
	if os.Getenv("EMBER_REAL_PACKS") != "1" {
		t.Skip("real asset packs are opt-in: EMBER_REAL_PACKS=1")
	}
	path := os.Getenv("EMBER_TEST_GLB_ZIP")
	if path == "" {
		home, _ := os.UserHomeDir()
		path = filepath.Join(home, "Downloads", "Apocalypse_Free_glb.zip")
	}
	z, zerr := zip.OpenReader(path)
	if zerr != nil {
		t.Skip("asset pack not available:", zerr)
	}
	defer z.Close()
	for _, f := range z.File {
		if filepath.Ext(f.Name) != ".glb" {
			continue
		}
		m, err := gltf.LoadFS(z, f.Name)
		if err != nil {
			t.Fatalf("%s: %v", f.Name, err)
		}
		t.Logf("%-50s parts %2d joints %3d clips %2d bounds %v",
			f.Name, len(m.Parts), len(m.Skeleton.Joints), len(m.Clips), m.Bounds.Size())
		if len(m.Parts) == 0 {
			t.Fatalf("%s has no parts", f.Name)
		}
	}
}
