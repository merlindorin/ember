package desktop

/*
#include <windows.h>
static void *ember_module(void) { return (void *)GetModuleHandleW(NULL); }
*/
import "C"

import (
	"unsafe"

	"github.com/go-gl/glfw/v3.3/glfw"
	"github.com/lantervale/ember/internal/wgpu"
)

// surfaceSource is the window's HWND and the program's module.
func surfaceSource(h *glfw.Window) (wgpu.SurfaceSource, error) {
	return wgpu.WindowsHWND{Instance: C.ember_module(), HWND: unsafe.Pointer(h.GetWin32Window())}, nil
}
