//go:build darwin

package desktop

/*
#cgo CFLAGS: -x objective-c
#cgo LDFLAGS: -framework Cocoa
#import <Cocoa/Cocoa.h>
#import <objc/runtime.h>
#include <stdint.h>

extern void emberTouch(uintptr_t window, uint64_t key, double x, double y, int phase);
extern void emberGesture(uintptr_t window, int kind, double a, double b);

static void emberTouches(id self, SEL _cmd, NSEvent *event) {
	NSView *view = (NSView *)self;
	uintptr_t window = (uintptr_t)[view window];
	for (NSTouch *t in [event touchesMatchingPhase:NSTouchPhaseAny inView:view]) {
		NSPoint p = [t normalizedPosition];
		emberTouch(window, (uint64_t)[(id)[t identity] hash], p.x, 1 - p.y, (int)[t phase]);
	}
}

static void emberMagnify(id self, SEL _cmd, NSEvent *event) {
	emberGesture((uintptr_t)[(NSView *)self window], 0, [event magnification], 0);
}

static void emberRotate(id self, SEL _cmd, NSEvent *event) {
	emberGesture((uintptr_t)[(NSView *)self window], 1, [event rotation], 0);
}

static void emberSwipe(id self, SEL _cmd, NSEvent *event) {
	emberGesture((uintptr_t)[(NSView *)self window], 2, [event deltaX], [event deltaY]);
}

static void emberListen(void *nswindow) {
	NSView *view = [(NSWindow *)nswindow contentView];
	view.allowedTouchTypes = NSTouchTypeMaskIndirect;
	view.wantsRestingTouches = YES;
	Class c = object_getClass(view);
	const char *types = "v@:@";
	class_addMethod(c, @selector(touchesBeganWithEvent:), (IMP)emberTouches, types);
	class_addMethod(c, @selector(touchesMovedWithEvent:), (IMP)emberTouches, types);
	class_addMethod(c, @selector(touchesEndedWithEvent:), (IMP)emberTouches, types);
	class_addMethod(c, @selector(touchesCancelledWithEvent:), (IMP)emberTouches, types);
	class_addMethod(c, @selector(magnifyWithEvent:), (IMP)emberMagnify, types);
	class_addMethod(c, @selector(rotateWithEvent:), (IMP)emberRotate, types);
	class_addMethod(c, @selector(swipeWithEvent:), (IMP)emberSwipe, types);
}
*/
import "C"

import (
	"github.com/go-gl/glfw/v3.3/glfw"
	"github.com/lantervale/ember/input"
)

// touched is each window's fingers, by its NSWindow; AppKit calls on the
// main thread, as the run loop does.
var touched = map[uintptr]*fingers{} //nolint:gochecknoglobals // AppKit calls back with no state of ours

// listenTouches has h's content view report trackpad touches and gestures
// into in.
func listenTouches(h *glfw.Window, in *input.State) {
	w := h.GetCocoaWindow()
	touched[uintptr(w)] = newFingers(in)
	C.emberListen(w)
}

func forgetTouches(h *glfw.Window) { delete(touched, uintptr(h.GetCocoaWindow())) }

//export emberTouch
func emberTouch(window C.uintptr_t, key C.uint64_t, x, y C.double, phase C.int) {
	if f := touched[uintptr(window)]; f != nil {
		f.touch(uint64(key), float64(x), float64(y), int(phase))
	}
}

//export emberGesture
func emberGesture(window C.uintptr_t, kind C.int, a, b C.double) {
	if f := touched[uintptr(window)]; f != nil {
		f.gesture(int(kind), float64(a), float64(b))
	}
}
