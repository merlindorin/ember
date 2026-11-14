//go:build gpu

package gpu //nolint:testpackage // white-box: drives the renderer's unexported passes

import (
	"testing"

	"github.com/lantervale/ember/golden"
)

// edges3D is the share of a 3D scene's pixels that may be over the levels,
// where multisampled silhouette edges resolve with another sample pattern.
const edges3D = 0.006

//nolint:gochecknoglobals // the goldens' list, shared by the native and browser tests
var goldenScenes = []golden.Scene{
	{Name: "15-themes", Pkg: "examples/15-themes", Frame: 30},
	{Name: "19-sprites", Pkg: "examples/19-sprites", Frame: 30},
	{Name: "21-lines-and-shapes", Pkg: "examples/21-lines-and-shapes", Frame: 30},
	{Name: "31-blend-modes", Pkg: "examples/31-blend-modes", Frame: 30},
	{
		Name:  "33-2d-shader",
		Pkg:   "examples/33-2d-shader",
		Frame: 45,
		Env:   []string{"EMBER_CAPTURE_MOUSE=320,192@0;160,250@20;left:10-12;left:22-24"},
	},
	{Name: "44-text", Pkg: "examples/44-text", Frame: 30},
	{Name: "01-hello-cube", Pkg: "examples/01-hello-cube", Frame: 30, Edges: edges3D},
	{Name: "09-window", Pkg: "examples/09-window", Frame: 30, Edges: edges3D},
	{Name: "10-multi-window", Pkg: "examples/10-multi-window", Frame: 30, Edges: edges3D},
	{Name: "look3d", Pkg: "examples/internal/look3d", Frame: 30, Edges: edges3D},
	{Name: "lights2d", Pkg: "examples/internal/lights2d", Frame: 90},
	{Name: "look3d-post", Pkg: "examples/internal/look3d", Frame: 30, Env: []string{"LOOK3D=post"}, Edges: edges3D},
}

// TestGolden checks the renderer against reference pictures in
// testdata/golden through package golden: a few 2D examples, the 3D ones
// and the internal scenes, each captured hidden at a fixed frame with scripted input
// and halved to points. The references were captured through OpenGL
// before it was removed (decision 0041). It needs the wgpu-native library
// (task wgpu) and a GPU:
//
//	go test -tags gpu ./gpu -run Golden [-update]
//
// A 2D scene passes when no channel differs by more than one level; a 3D
// scene may have up to 0.6% of its pixels over that. Gizmo lines aren't
// covered: OpenGL drivers and WebGPU rasterize them a pixel apart, and
// bloom spreads that (look3d draws them only with LOOK3D=lines).
func TestGolden(t *testing.T) {
	golden.Run(t, "..", "testdata/golden", goldenScenes)
}
