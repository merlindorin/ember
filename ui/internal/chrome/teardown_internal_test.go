//go:build chrome

package chrome

import (
	"image"
	"image/png"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/merlindorin/ember/geom"
)

// fakeChrome writes a script standing for Chrome that does what action
// asks, then dies at teardown as Chrome's watchdog makes it on a loaded
// Mac, or fails outright when broken.
func fakeChrome(t *testing.T, broken bool) {
	t.Helper()
	dir := t.TempDir()
	shot := filepath.Join(dir, "shot.png")
	f, err := os.Create(shot)
	require.NoError(t, err)
	require.NoError(t, png.Encode(f, image.NewNRGBA(image.Rect(0, 0, 4, 4))))
	f.Close()
	body := `for a in "$@"; do
	case "$a" in
	--dump-dom) echo '<html><body>done</body></html>';;
	--screenshot=*) cp ` + shot + ` "${a#--screenshot=}";;
	esac
done
echo "Teardown watchdog expired; recording dump and terminating." >&2
exit 2
`
	if broken {
		body = "echo 'crashed before drawing' >&2\nexit 2\n"
	}
	script := filepath.Join(dir, "chrome")
	require.NoError(t, os.WriteFile(script, []byte("#!/bin/sh\n"+body), 0o755))
	t.Setenv("EMBER_CHROME", script)
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
