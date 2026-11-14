//go:build chrome

package chrome

import (
	"fmt"
	"image"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/lantervale/ember/geom"
)

// TestMain lets the test binary stand for Chrome when fakeChrome asks it
// to, a fake that runs on every system.
func TestMain(m *testing.M) {
	if mode, ok := os.LookupEnv("EMBER_FAKE_CHROME"); ok {
		os.Exit(fakeChromeMain(mode, os.Args[1:]))
	}
	os.Exit(m.Run())
}

// fakeChromeMain does what Chrome's arguments ask, the picture copied from
// the file mode names, then dies at teardown as Chrome's watchdog makes it
// on a loaded Mac; mode "broken" fails outright.
func fakeChromeMain(mode string, args []string) int {
	if mode == "broken" {
		fmt.Fprintln(os.Stderr, "crashed before drawing")
		return 2
	}
	for _, a := range args {
		switch {
		case a == dumpDOM:
			fmt.Fprintln(os.Stdout, "<html><body>done</body></html>")
		case strings.HasPrefix(a, "--screenshot="):
			shot, err := os.ReadFile(mode)
			if err != nil {
				fmt.Fprintln(os.Stderr, err)
				return 1
			}
			if err = os.WriteFile(strings.TrimPrefix(a, "--screenshot="), shot, 0o600); err != nil {
				fmt.Fprintln(os.Stderr, err)
				return 1
			}
		}
	}
	fmt.Fprintln(os.Stderr, "Teardown watchdog expired; recording dump and terminating.")
	return 2
}

// fakeChrome has the test binary stand for Chrome: doing what action asks,
// then dying at teardown, or failing outright when broken.
func fakeChrome(t *testing.T, broken bool) {
	t.Helper()
	mode := "broken"
	if !broken {
		mode = filepath.Join(t.TempDir(), "shot.png")
		f, err := os.Create(mode)
		require.NoError(t, err)
		require.NoError(t, png.Encode(f, image.NewNRGBA(image.Rect(0, 0, 4, 4))))
		f.Close()
	}
	self, err := os.Executable()
	require.NoError(t, err)
	t.Setenv("EMBER_FAKE_CHROME", mode)
	t.Setenv("EMBER_CHROME", self)
}

func TestAChromeKilledAtTeardownAfterItsWorkStillCounts(t *testing.T) {
	fakeChrome(t, false)
	out, err := runRaw([]byte("<html></html>"), geom.V2(100, 100), "--dump-dom")
	require.NoError(t, err)
	assert.Contains(t, string(out), "done")

	shot := filepath.Join(t.TempDir(), "page.png")
	_, err = runRaw([]byte("<html></html>"), geom.V2(4, 4), "--screenshot="+shot)
	require.NoError(t, err, "the picture was written before teardown")
}

func TestAChromeThatFailsBeforeItsWorkIsAnError(t *testing.T) {
	fakeChrome(t, true)
	_, err := runRaw([]byte("<html></html>"), geom.V2(100, 100), "--dump-dom")
	assert.Error(t, err)
	_, err = runRaw([]byte("<html></html>"), geom.V2(4, 4), "--screenshot="+filepath.Join(t.TempDir(), "page.png"))
	assert.Error(t, err)
}
