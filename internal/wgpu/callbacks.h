#ifndef EMBER_WGPU_CALLBACKS_H
#define EMBER_WGPU_CALLBACKS_H

#include "procs.h"

// ember_wait is an asynchronous call's result: done once its callback ran,
// with its status, its object and its message.
typedef struct {
	int done;
	int status;
	int type;
	void *object;
	char message[1024];
} ember_wait;

WGPUFuture ember_request_adapter(WGPUInstance instance, WGPURequestAdapterOptions const *options, ember_wait *w);
WGPUFuture ember_request_device(WGPUAdapter adapter, int timestamps, ember_wait *w);
WGPUFuture ember_pop_error_scope(WGPUDevice device, ember_wait *w);
WGPUFuture ember_map_read(WGPUBuffer buffer, size_t size, ember_wait *w);

// ember_last_error is the last error nothing caught, with its type, and is
// cleared by ember_take_error.
int ember_take_error(char *out, size_t cap);

#endif
