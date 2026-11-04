//go:build darwin || linux

package wgpu

/*
#cgo linux LDFLAGS: -ldl
#include <dlfcn.h>
#include <stdlib.h>
#include "procs.h"

static void *ember_sym(void *lib, const char *name) { return dlsym(lib, name); }
static void *ember_open(const char *path) { return dlopen(path, RTLD_NOW | RTLD_LOCAL); }
static const char *ember_dlerror(void) { return dlerror(); }
static const char *ember_load_all(void *lib) { return ember_wgpu_load(ember_sym, lib); }
*/
import "C"

import (
	"errors"
	"unsafe"
)

func open(path string) error {
	p := C.CString(path)
	defer C.free(unsafe.Pointer(p))
	lib := C.ember_open(p)
	if lib == nil {
		return errors.New(C.GoString(C.ember_dlerror()))
	}
	if missing := C.ember_load_all(lib); missing != nil {
		return errors.New("no " + C.GoString(missing) + " in " + path)
	}
	return nil
}
