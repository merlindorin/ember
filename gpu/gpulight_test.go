package gpu //nolint:testpackage // checks the unexported shader sources against gpuLight

import (
	"reflect"
	"strings"
	"testing"
)

func TestEveryShaderDeclaresGPULightAsGoLaysItOut(t *testing.T) {
	want := reflect.TypeFor[gpuLight]().NumField()
	for name, src := range map[string]string{"light picture": lightPictureWGSL, "silhouette mask": silhouetteMaskWGSL} {
		i := strings.Index(src, "struct GPULight {")
		if i < 0 {
			t.Fatalf("%s declares GPULight", name)
		}
		body := src[i : strings.Index(src[i:], "};")+i]
		if got := strings.Count(body, ": vec4f,"); got != want {
			t.Fatalf("%s's GPULight has %d vec4f fields, gpuLight %d", name, got, want)
		}
	}
}
