package nav_test

import (
	"os/exec"
	"strings"
	"testing"
)

func TestNavStaysPureGo(t *testing.T) {
	out, err := exec.Command("go", "list", "-deps", ".").CombinedOutput()
	if err != nil {
		t.Skipf("go list: %v\n%s", err, out)
	}
	for _, dep := range strings.Fields(string(out)) {
		for _, banned := range []string{"/gpu", "/gfx", "/platform", "/twod", "go-gl/"} {
			if strings.Contains(dep, banned) {
				t.Errorf("nav depends on %s", dep)
			}
		}
		if dep == "C" {
			t.Error("nav uses cgo")
		}
	}
}
