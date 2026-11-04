package wgpu

// The header declares its flags as static const variables, which cgo can't
// take as constants with every C compiler (mingw links them as symbols the
// library never exports), so they are read through these functions.

/*
#include <stdint.h>
#include "procs.h"

static inline uint64_t ember_WGPUTextureUsage_CopySrc(void) { return (uint64_t)WGPUTextureUsage_CopySrc; }
static inline uint64_t ember_WGPUTextureUsage_CopyDst(void) { return (uint64_t)WGPUTextureUsage_CopyDst; }
static inline uint64_t ember_WGPUTextureUsage_TextureBinding(void) { return (uint64_t)WGPUTextureUsage_TextureBinding; }
static inline uint64_t ember_WGPUTextureUsage_RenderAttachment(void) {
	return (uint64_t)WGPUTextureUsage_RenderAttachment;
}
static inline uint64_t ember_WGPUBufferUsage_Vertex(void) { return (uint64_t)WGPUBufferUsage_Vertex; }
static inline uint64_t ember_WGPUBufferUsage_Index(void) { return (uint64_t)WGPUBufferUsage_Index; }
static inline uint64_t ember_WGPUBufferUsage_Uniform(void) { return (uint64_t)WGPUBufferUsage_Uniform; }
static inline uint64_t ember_WGPUBufferUsage_CopyDst(void) { return (uint64_t)WGPUBufferUsage_CopyDst; }
static inline uint64_t ember_WGPUBufferUsage_CopySrc(void) { return (uint64_t)WGPUBufferUsage_CopySrc; }
static inline uint64_t ember_WGPUBufferUsage_QueryResolve(void) { return (uint64_t)WGPUBufferUsage_QueryResolve; }
static inline uint64_t ember_WGPUBufferUsage_MapRead(void) { return (uint64_t)WGPUBufferUsage_MapRead; }
static inline uint64_t ember_WGPUBufferUsage_MapWrite(void) { return (uint64_t)WGPUBufferUsage_MapWrite; }
static inline uint64_t ember_WGPUShaderStage_Vertex(void) { return (uint64_t)WGPUShaderStage_Vertex; }
static inline uint64_t ember_WGPUShaderStage_Fragment(void) { return (uint64_t)WGPUShaderStage_Fragment; }
static inline uint64_t ember_WGPUColorWriteMask_All(void) { return (uint64_t)WGPUColorWriteMask_All; }
*/
import "C"

//nolint:gochecknoglobals // a value from C; cgo makes no Go constants of it
var colorWriteAll = C.ember_WGPUColorWriteMask_All()

// The usages and stages, from C.
//
//nolint:gochecknoglobals // values from C; cgo makes no Go constants of them
var (
	UsageCopySrc          = Usage(C.ember_WGPUTextureUsage_CopySrc())
	UsageCopyDst          = Usage(C.ember_WGPUTextureUsage_CopyDst())
	UsageTextureBinding   = Usage(C.ember_WGPUTextureUsage_TextureBinding())
	UsageRenderAttachment = Usage(C.ember_WGPUTextureUsage_RenderAttachment())

	BufferVertex       = Usage(C.ember_WGPUBufferUsage_Vertex())
	BufferIndex        = Usage(C.ember_WGPUBufferUsage_Index())
	BufferUniform      = Usage(C.ember_WGPUBufferUsage_Uniform())
	BufferCopyDst      = Usage(C.ember_WGPUBufferUsage_CopyDst())
	BufferCopySrc      = Usage(C.ember_WGPUBufferUsage_CopySrc())
	BufferMapRead      = Usage(C.ember_WGPUBufferUsage_MapRead())
	BufferQueryResolve = Usage(C.ember_WGPUBufferUsage_QueryResolve())
	BufferMapWrite     = Usage(C.ember_WGPUBufferUsage_MapWrite())
)

//nolint:gochecknoglobals // values from C; cgo makes no Go constants of them
var (
	StageVertex   = Stage(C.ember_WGPUShaderStage_Vertex())
	StageFragment = Stage(C.ember_WGPUShaderStage_Fragment())
)
