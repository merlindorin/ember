package audio_test

import (
	"os/exec"
	"strings"
	"testing"
)

func TestAudioStaysPureGo(t *testing.T) {
	out, err := exec.Command("go", "list", "-deps", ".", "./wav", "./vorbis").CombinedOutput()
	if err != nil {
		t.Skipf("go list: %v\n%s", err, out)
	}
	for _, dep := range strings.Fields(string(out)) {
		for _, banned := range []string{
			"/gpu", "/gfx", "/platform/desktop", "/internal/audiodevice", "go-gl/", "ebitengine/oto", "C",
		} {
			if dep == banned || strings.Contains(dep, banned) && banned != "C" {
				t.Errorf("audio depends on %s", dep)
			}
		}
	}
}
