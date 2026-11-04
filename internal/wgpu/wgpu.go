// Package wgpu is ember's own binding to wgpu-native, the WebGPU
// implementation the renderer draws through (decision 0041). It covers only
// the calls the renderer makes. cgo reads every struct from the vendored
// webgpu.h and wgpu.h, so nothing is laid out by hand; the library itself
// is opened at run time, so building needs only the header, and only a
// running game needs the library (task wgpu fetches it).
package wgpu

/*
#cgo CFLAGS: -I${SRCDIR}/include
#include <stdlib.h>
#include "procs.h"
#include "callbacks.h"
*/
import "C"

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"unsafe"
)

// LibraryEnv names the wgpu-native library to load, ahead of the places
// Load looks.
const LibraryEnv = "EMBER_WGPU_LIB"

// FallbackEnv, set to 1, makes RequestAdapter ask for wgpu's fallback
// adapter, one drawing on the CPU: WARP through Direct3D 12 on Windows,
// lavapipe through Vulkan on Linux. Machines without a GPU, such as CI
// runners, draw through it.
const FallbackEnv = "EMBER_WGPU_FALLBACK"

// OptionalEnv, set to 1, makes the GPU tests skip, saying why, when no
// adapter is found, instead of failing: a CI runner whose GPU is out of
// reach.
const OptionalEnv = "EMBER_GPU_OPTIONAL"

//nolint:gochecknoglobals // the library opens once per process
var loaded struct {
	once sync.Once
	err  error
	path string
}

// Load opens wgpu-native once: from EMBER_WGPU_LIB, else beside the
// executable, else, in a macOS app, from its Contents/Frameworks, else
// from internal/wgpu/lib in the source tree, where task wgpu puts it (not
// for a build made with -trimpath, which knows no source tree).
func Load() error {
	loaded.once.Do(func() {
		var tried []string
		for _, path := range candidates() {
			if _, err := os.Stat(path); err != nil {
				tried = append(tried, path)
				continue
			}
			if err := open(path); err != nil {
				loaded.err = fmt.Errorf("wgpu: %w", err)
				return
			}
			loaded.path = path
			return
		}
		loaded.err = fmt.Errorf(
			"wgpu: no %s; run task wgpu, or set %s (looked in %v)",
			libraryName(),
			LibraryEnv,
			tried,
		)
	})
	return loaded.err
}

// Library is the path Load opened.
func Library() string { return loaded.path }

// goosDarwin is macOS's GOOS; the backends are wgpu's, by name.
const (
	goosDarwin    = "darwin"
	backendOpenGL = "OpenGL"
	backendVulkan = "Vulkan"
)

func libraryName() string { return libraryFor(runtime.GOOS) }

func libraryFor(goos string) string {
	switch goos {
	case "windows":
		return "wgpu_native.dll"
	case goosDarwin:
		return "libwgpu_native.dylib"
	}
	return "libwgpu_native.so"
}

func candidates() []string {
	exe, _ := os.Executable()
	return candidatesFor(runtime.GOOS, exe)
}

// candidatesFor are where Load looks on goos for a game whose executable
// is exe, empty when unknown.
func candidatesFor(goos, exe string) []string {
	var out []string
	if p := os.Getenv(LibraryEnv); p != "" {
		out = append(out, p)
	}
	if exe != "" {
		out = append(out, filepath.Join(filepath.Dir(exe), libraryFor(goos)))
		if goos == goosDarwin {
			out = append(out, filepath.Join(filepath.Dir(exe), "..", "Frameworks", libraryFor(goos)))
		}
	}
	if _, file, _, ok := runtime.Caller(0); ok && filepath.IsAbs(file) {
		out = append(out, filepath.Join(filepath.Dir(file), "lib", runtime.GOOS+"-"+runtime.GOARCH, libraryName()))
	}
	return out
}

// Version is wgpu-native's version, as major.minor.patch.build.
func Version() string {
	v := uint32(C.e_wgpuGetVersion())
	return fmt.Sprintf("%d.%d.%d.%d", v>>24, v>>16&0xff, v>>8&0xff, v&0xff)
}

// arena holds the C memory one call's descriptors take, freed together.
type arena struct{ ptrs []unsafe.Pointer }

func alloc[T any](a *arena) *T {
	var zero T
	p := C.calloc(1, C.size_t(unsafe.Sizeof(zero)))
	a.ptrs = append(a.ptrs, p)
	return (*T)(p)
}

func allocN[T any](a *arena, n int) []T {
	if n == 0 {
		return nil
	}
	var zero T
	p := C.calloc(C.size_t(n), C.size_t(unsafe.Sizeof(zero)))
	a.ptrs = append(a.ptrs, p)
	return unsafe.Slice((*T)(p), n)
}

func (a *arena) str(s string) C.WGPUStringView {
	if s == "" {
		return C.WGPUStringView{data: nil, length: C.SIZE_MAX}
	}
	p := C.CString(s)
	a.ptrs = append(a.ptrs, unsafe.Pointer(p))
	return C.WGPUStringView{data: p, length: C.size_t(len(s))}
}

func (a *arena) free() {
	for _, p := range a.ptrs {
		C.free(p)
	}
	a.ptrs = nil
}

func goString(v C.WGPUStringView) string {
	if v.data == nil {
		return ""
	}
	if v.length == C.SIZE_MAX {
		return C.GoString(v.data)
	}
	return C.GoStringN(v.data, C.int(v.length))
}

// Instance is the entry to WebGPU: it makes surfaces and finds adapters.
type Instance struct{ h C.WGPUInstance }

// NewInstance loads the library if need be and makes an instance.
func NewInstance() (*Instance, error) {
	if err := Load(); err != nil {
		return nil, err
	}
	h := C.e_wgpuCreateInstance(nil)
	if h == nil {
		return nil, errors.New("wgpu: no instance")
	}
	return &Instance{h: h}, nil
}

// wait processes the instance's events until w's callback has run.
func (i *Instance) wait(w *C.ember_wait) {
	for w.done == 0 {
		C.e_wgpuInstanceProcessEvents(i.h)
	}
}

func newWait() *C.ember_wait {
	return (*C.ember_wait)(C.calloc(1, C.size_t(unsafe.Sizeof(C.ember_wait{}))))
}

func freeWait(w *C.ember_wait) { C.free(unsafe.Pointer(w)) }

// Adapter is a GPU and the backend that drives it.
type Adapter struct {
	h    C.WGPUAdapter
	inst *Instance
}

// RequestAdapter finds the high-performance adapter that can present to
// compatible, which may be nil; the fallback one when FallbackEnv is 1.
func (i *Instance) RequestAdapter(compatible *Surface) (*Adapter, error) {
	var a arena
	defer a.free()
	opts := alloc[C.WGPURequestAdapterOptions](&a)
	opts.powerPreference = C.WGPUPowerPreference_HighPerformance
	if os.Getenv(FallbackEnv) == "1" {
		opts.forceFallbackAdapter = 1
	}
	if compatible != nil {
		opts.compatibleSurface = compatible.h
	}
	w := newWait()
	defer freeWait(w)
	C.ember_request_adapter(i.h, opts, w)
	i.wait(w)
	if w.status != C.WGPURequestAdapterStatus_Success || w.object == nil {
		return nil, fmt.Errorf("wgpu: no adapter: %s", C.GoString(&w.message[0]))
	}
	return &Adapter{h: C.WGPUAdapter(w.object), inst: i}, nil
}

// Backends are the APIs wgpu-native looks for an adapter through on this
// platform, in its order.
func Backends() []string {
	switch runtime.GOOS {
	case goosDarwin, "ios":
		return []string{"Metal"}
	case "windows":
		return []string{"Direct3D 12", backendVulkan, backendOpenGL}
	}
	return []string{backendVulkan, backendOpenGL}
}

// AdapterInfo says what an adapter is.
type AdapterInfo struct {
	Vendor, Architecture, Device, Description, Backend string
}

//nolint:gochecknoglobals // a lookup table
var backends = map[C.WGPUBackendType]string{
	C.WGPUBackendType_Null: "null", C.WGPUBackendType_WebGPU: "WebGPU", C.WGPUBackendType_D3D11: "Direct3D 11",
	C.WGPUBackendType_D3D12: "Direct3D 12", C.WGPUBackendType_Metal: "Metal", C.WGPUBackendType_Vulkan: backendVulkan,
	C.WGPUBackendType_OpenGL: backendOpenGL, C.WGPUBackendType_OpenGLES: "OpenGL ES",
}

func (ad *Adapter) Info() AdapterInfo {
	var a arena
	defer a.free()
	info := alloc[C.WGPUAdapterInfo](&a)
	if C.e_wgpuAdapterGetInfo(ad.h, info) != C.WGPUStatus_Success {
		return AdapterInfo{}
	}
	defer C.e_wgpuAdapterInfoFreeMembers(*info)
	return AdapterInfo{
		Vendor:       goString(info.vendor),
		Architecture: goString(info.architecture),
		Device:       goString(info.device),
		Description:  goString(info.description),
		Backend:      backends[info.backendType],
	}
}

// Device makes the GPU's resources, and its queue runs their commands.
type Device struct {
	h    C.WGPUDevice
	q    C.WGPUQueue
	inst *Instance
	// timestamps is whether passes can be timed.
	timestamps bool
}

// RequestDevice opens the adapter's device and its queue.
func (ad *Adapter) RequestDevice() (*Device, error) {
	w := newWait()
	defer freeWait(w)
	timestamps := C.e_wgpuAdapterHasFeature(ad.h, C.WGPUFeatureName_TimestampQuery) != 0
	C.ember_request_device(ad.h, C.int(b2i(timestamps)), w)
	ad.inst.wait(w)
	if w.status != C.WGPURequestDeviceStatus_Success || w.object == nil {
		return nil, fmt.Errorf("wgpu: no device: %s", C.GoString(&w.message[0]))
	}
	d := C.WGPUDevice(w.object)
	return &Device{h: d, q: C.e_wgpuDeviceGetQueue(d), inst: ad.inst, timestamps: timestamps}, nil
}

// TakeError is the last error no error scope caught, nil for none, and
// clears it.
func TakeError() error {
	buf := make([]byte, 2048)
	if t := C.ember_take_error((*C.char)(unsafe.Pointer(&buf[0])), C.size_t(len(buf))); t != 0 {
		return errors.New("wgpu: " + C.GoString((*C.char)(unsafe.Pointer(&buf[0]))))
	}
	return nil
}

// scoped runs create inside a validation error scope, and gives the error
// it raised, nil for none.
func (d *Device) scoped(create func()) error {
	C.e_wgpuDevicePushErrorScope(d.h, C.WGPUErrorFilter_Validation)
	create()
	w := newWait()
	defer freeWait(w)
	C.ember_pop_error_scope(d.h, w)
	d.inst.wait(w)
	if w.status == C.WGPUPopErrorScopeStatus_Success && w._type != C.WGPUErrorType_NoError {
		return errors.New(C.GoString(&w.message[0]))
	}
	return nil
}

func b2i(b bool) int {
	if b {
		return 1
	}
	return 0
}
