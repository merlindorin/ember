//go:build linux && !wayland

package desktop

import (
	"unsafe"

	"github.com/go-gl/glfw/v3.3/glfw"
	"github.com/lantervale/ember/internal/wgpu"
)

// surfaceSource is the window's X11 window on GLFW's display. A build
// tagged wayland, as GLFW's own, takes the Wayland surface instead.
func surfaceSource(h *glfw.Window) (wgpu.SurfaceSource, error) {
	return wgpu.XlibWindow{Display: unsafe.Pointer(glfw.GetX11Display()), Window: uint64(h.GetX11Window())}, nil
}
