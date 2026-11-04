package wgpu

/*
#include <windows.h>
#include <stdlib.h>
#include "procs.h"

static void *ember_sym(void *lib, const char *name) { return (void *)GetProcAddress((HMODULE)lib, name); }
static void *ember_open(const char *path) { return (void *)LoadLibraryA(path); }
static unsigned long ember_last(void) { return GetLastError(); }
static const char *ember_load_all(void *lib) { return ember_wgpu_load(ember_sym, lib); }
*/
import "C"

import (
	"errors"
	"fmt"
	"unsafe"
)

func open(path string) error {
	p := C.CString(path)
	defer C.free(unsafe.Pointer(p))
	lib := C.ember_open(p)
	if lib == nil {
		return fmt.Errorf("LoadLibrary %s: error %d", path, C.ember_last())
	}
	if missing := C.ember_load_all(lib); missing != nil {
		return errors.New("no " + C.GoString(missing) + " in " + path)
	}
	return nil
}
