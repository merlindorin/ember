package gpu //nolint:testpackage // tests the unexported blend and clamp tables

import (
	"testing"
	"unsafe"

	"github.com/merlindorin/ember/internal/draw"
	"github.com/merlindorin/ember/internal/wgpu"
	"github.com/stretchr/testify/assert"
)

func TestEachBlendModeHasItsBlendState(t *testing.T) {
	alpha := wgpu.BlendComponent{Op: wgpu.Add, Src: wgpu.One, Dst: wgpu.OneMinusSrcAlpha}
	for m, want := range map[draw.Blend]wgpu.Blend{
		draw.Alpha: {
			Color: wgpu.BlendComponent{Op: wgpu.Add, Src: wgpu.SrcAlpha, Dst: wgpu.OneMinusSrcAlpha},
			Alpha: alpha,
		},
		draw.Add: {Color: wgpu.BlendComponent{Op: wgpu.Add, Src: wgpu.SrcAlpha, Dst: wgpu.One}, Alpha: alpha},
		draw.Multiply: {
			Color: wgpu.BlendComponent{Op: wgpu.Add, Src: wgpu.Zero, Dst: wgpu.Src},
			Alpha: wgpu.BlendComponent{Op: wgpu.Add, Src: wgpu.Zero, Dst: wgpu.One},
		},
		draw.Screen:  {Color: wgpu.BlendComponent{Op: wgpu.Add, Src: wgpu.One, Dst: wgpu.OneMinusSrc}, Alpha: alpha},
		draw.Lighten: {Color: wgpu.BlendComponent{Op: wgpu.Max, Src: wgpu.One, Dst: wgpu.One}, Alpha: alpha},
		draw.Occlude: {
			Color: wgpu.BlendComponent{Op: wgpu.Max, Src: wgpu.One, Dst: wgpu.One},
			Alpha: wgpu.BlendComponent{Op: wgpu.Max, Src: wgpu.One, Dst: wgpu.One},
		},
	} {
		assert.Equal(t, want, blendOf(m), "blend %d", m)
	}
}

func TestTheUniformsLieAsUIWGSLsDrawStruct(t *testing.T) {
	var u drawUniforms
	for name, at := range map[string][2]uintptr{
		"screen":   {unsafe.Offsetof(u.Screen), 0},
		"scale":    {unsafe.Offsetof(u.Scale), 8},
		"time":     {unsafe.Offsetof(u.Time), 12},
		"hdr":      {unsafe.Offsetof(u.HDR), 16},
		"exposure": {unsafe.Offsetof(u.Exposure), 20},
		"masked":   {unsafe.Offsetof(u.Masked), 24},
		"blend":    {unsafe.Offsetof(u.Blend), 28}, "targets": {unsafe.Offsetof(u.Targets), 32},
	} {
		assert.Equal(t, at[1], at[0], name)
	}
	assert.LessOrEqual(t, unsafe.Sizeof(u), uintptr(uniformStride))
}

func TestAClipIsKeptToItsTarget(t *testing.T) {
	assert.Equal(t, 0, clampTo(-3, 100))
	assert.Equal(t, 100, clampTo(140, 100))
	assert.Equal(t, 42, clampTo(41.6, 100))
}
