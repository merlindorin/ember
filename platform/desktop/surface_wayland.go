//go:build linux && wayland

package desktop

import (
	"unsafe"

	"github.com/go-gl/glfw/v3.3/glfw"
	"github.com/lantervale/ember/internal/wgpu"
)

// surfaceSource is the window's Wayland surface on GLFW's display.
func surfaceSource(h *glfw.Window) (wgpu.SurfaceSource, error) {
	return wgpu.WaylandSurface{
		Display: unsafe.Pointer(glfw.GetWaylandDisplay()),
		Surface: unsafe.Pointer(h.GetWaylandWindow()),
	}, nil
}
