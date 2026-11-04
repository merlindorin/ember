package desktop

/*
#cgo CFLAGS: -x objective-c
#cgo LDFLAGS: -framework Cocoa -framework QuartzCore
#import <Cocoa/Cocoa.h>
#import <QuartzCore/CAMetalLayer.h>

static void *ember_metal_layer(void *window, double scale) {
	NSWindow *w = (__bridge NSWindow *)window;
	NSView *v = [w contentView];
	CAMetalLayer *l = [CAMetalLayer layer];
	l.contentsScale = scale;
	[v setWantsLayer:YES];
	[v setLayer:l];
	return (__bridge void *)l;
}

// The C side ends here.
*/
import "C"

import (
	"github.com/go-gl/glfw/v3.3/glfw"
	"github.com/merlindorin/ember/internal/wgpu"
)

// surfaceSource puts a CAMetalLayer on the window's view for WebGPU to
// draw into, at the window's content scale.
func surfaceSource(h *glfw.Window) (wgpu.SurfaceSource, error) {
	scale, _ := h.GetContentScale()
	return wgpu.MetalLayer{Layer: C.ember_metal_layer(h.GetCocoaWindow(), C.double(scale))}, nil
}
