#include <string.h>
#include "callbacks.h"

static void keep(ember_wait *w, WGPUStringView m) {
	size_t n = 0;
	if (m.data) {
		n = m.length == WGPU_STRLEN ? strlen(m.data) : m.length;
		if (n >= sizeof(w->message)) n = sizeof(w->message) - 1;
		memcpy(w->message, m.data, n);
	}
	w->message[n] = 0;
}

static void on_adapter(WGPURequestAdapterStatus s, WGPUAdapter a, WGPUStringView m, void *u1, void *u2) {
	ember_wait *w = u1;
	w->status = s;
	w->object = a;
	keep(w, m);
	w->done = 1;
}

WGPUFuture ember_request_adapter(WGPUInstance instance, WGPURequestAdapterOptions const *options, ember_wait *w) {
	WGPURequestAdapterCallbackInfo info = {0};
	info.mode = WGPUCallbackMode_AllowProcessEvents;
	info.callback = on_adapter;
	info.userdata1 = w;
	return e_wgpuInstanceRequestAdapter(instance, options, info);
}

static void on_device(WGPURequestDeviceStatus s, WGPUDevice d, WGPUStringView m, void *u1, void *u2) {
	ember_wait *w = u1;
	w->status = s;
	w->object = d;
	keep(w, m);
	w->done = 1;
}

static char last_error[2048];
static int last_type;

static void on_uncaptured(WGPUDevice const *device, WGPUErrorType type, WGPUStringView m, void *u1, void *u2) {
	size_t n = 0;
	if (m.data) {
		n = m.length == WGPU_STRLEN ? strlen(m.data) : m.length;
		if (n >= sizeof(last_error)) n = sizeof(last_error) - 1;
		memcpy(last_error, m.data, n);
	}
	last_error[n] = 0;
	last_type = type;
}

int ember_take_error(char *out, size_t cap) {
	int t = last_type;
	if (!t) return 0;
	strncpy(out, last_error, cap - 1);
	out[cap - 1] = 0;
	last_type = 0;
	return t;
}

WGPUFuture ember_request_device(WGPUAdapter adapter, int timestamps, ember_wait *w) {
	WGPUDeviceDescriptor desc = {0};
	static const WGPUFeatureName features[] = {WGPUFeatureName_TimestampQuery};
	if (timestamps) {
		desc.requiredFeatureCount = 1;
		desc.requiredFeatures = features;
	}
	desc.uncapturedErrorCallbackInfo.callback = on_uncaptured;
	WGPURequestDeviceCallbackInfo info = {0};
	info.mode = WGPUCallbackMode_AllowProcessEvents;
	info.callback = on_device;
	info.userdata1 = w;
	return e_wgpuAdapterRequestDevice(adapter, &desc, info);
}

static void on_pop(WGPUPopErrorScopeStatus s, WGPUErrorType type, WGPUStringView m, void *u1, void *u2) {
	ember_wait *w = u1;
	w->status = s;
	w->type = type;
	keep(w, m);
	w->done = 1;
}

WGPUFuture ember_pop_error_scope(WGPUDevice device, ember_wait *w) {
	WGPUPopErrorScopeCallbackInfo info = {0};
	info.mode = WGPUCallbackMode_AllowProcessEvents;
	info.callback = on_pop;
	info.userdata1 = w;
	return e_wgpuDevicePopErrorScope(device, info);
}

static void on_map(WGPUMapAsyncStatus s, WGPUStringView m, void *u1, void *u2) {
	ember_wait *w = u1;
	w->status = s;
	keep(w, m);
	w->done = 1;
}

WGPUFuture ember_map_read(WGPUBuffer buffer, size_t size, ember_wait *w) {
	WGPUBufferMapCallbackInfo info = {0};
	info.mode = WGPUCallbackMode_AllowProcessEvents;
	info.callback = on_map;
	info.userdata1 = w;
	return e_wgpuBufferMapAsync(buffer, WGPUMapMode_Read, 0, size, info);
}
