//go:build darwin

package desktop

/*
#cgo CFLAGS: -x objective-c
#cgo LDFLAGS: -framework Cocoa
#import <Cocoa/Cocoa.h>
#import <objc/runtime.h>
#include <stdint.h>
#include <stdlib.h>

extern void emberText(uintptr_t window, char *text);

// emberInsertText replaces GLFW's insertText:, which hands a character to
// the game only when the current event holds no Command key: the emoji
// picker, opened with Control+Command+Space, then inserted nothing.
static void emberInsertText(id self, SEL _cmd, id string, NSRange replacement) {
	NSString *s = [string isKindOfClass:[NSAttributedString class]] ? [string string] : string;
	NSEvent *e = [NSApp currentEvent];
	if (e != nil && [e type] == NSEventTypeKeyDown && ([e modifierFlags] & NSEventModifierFlagCommand)) {
		return;
	}
	emberText((uintptr_t)[(NSView *)self window], (char *)[s UTF8String]);
}

static void emberListenText(void *nswindow) {
	NSView *view = [(NSWindow *)nswindow contentView];
	Method m = class_getInstanceMethod(object_getClass(view), @selector(insertText:replacementRange:));
	if (m != NULL) {
		method_setImplementation(m, (IMP)emberInsertText);
	}
}

static void emberInsert(void *nswindow, char *text) {
	NSView *view = [(NSWindow *)nswindow contentView];
	[(id<NSTextInputClient>)view insertText:[NSString stringWithUTF8String:text]
		replacementRange:NSMakeRange(NSNotFound, 0)];
}
*/
import "C"

import (
	"unsafe"

	"github.com/go-gl/glfw/v3.3/glfw"
	"github.com/lantervale/ember/input"
)

// typed is each window's input, by its NSWindow, for inserted text.
var typed = map[uintptr]*input.State{} //nolint:gochecknoglobals // AppKit calls back with no state of ours

// listenText has h's content view hand whole inserted strings, from typing,
// the emoji picker or the Character Viewer, to in.
func listenText(h *glfw.Window, in *input.State) {
	w := h.GetCocoaWindow()
	typed[uintptr(w)] = in
	C.emberListenText(w)
}

func forgetText(h *glfw.Window) { delete(typed, uintptr(h.GetCocoaWindow())) }

// insert sends text to h as the emoji picker does, for tests.
func insert(h *glfw.Window, text string) {
	cs := C.CString(text)
	defer C.free(unsafe.Pointer(cs))
	C.emberInsert(h.GetCocoaWindow(), cs)
}

//export emberText
func emberText(window C.uintptr_t, text *C.char) {
	if in := typed[uintptr(window)]; in != nil {
		addText(in, C.GoString(text))
	}
}
