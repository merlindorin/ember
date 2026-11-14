//go:build darwin

package desktop //nolint:testpackage // tests the unexported AppKit text bridge

import (
	"os"
	"runtime"
	"testing"

	"github.com/go-gl/glfw/v3.3/glfw"
	"github.com/lantervale/ember/input"
	"github.com/lantervale/ember/internal/inputfeed"
)

// onMain runs functions on the main thread, where AppKit and GLFW must be
// called.
var onMain = make(chan func()) //nolint:gochecknoglobals // TestMain lends the main thread to the tests

func TestMain(m *testing.M) {
	runtime.LockOSThread()
	done := make(chan int)
	go func() { done <- m.Run() }()
	for {
		select {
		case f := <-onMain:
			f()
		case code := <-done:
			os.Exit(code)
		}
	}
}

func main(f func()) {
	ok := make(chan struct{})
	onMain <- func() { f(); close(ok) }
	<-ok
}

func TestTheEmojiPickersTextReachesTheInputWhole(t *testing.T) {
	var in input.State
	var failed error
	main(func() {
		if err := glfw.Init(); err != nil {
			failed = err
			return
		}
		glfw.WindowHint(glfw.Visible, glfw.False)
		glfw.WindowHint(glfw.ClientAPI, glfw.NoAPI)
		h, err := glfw.CreateWindow(64, 64, "text", nil, nil)
		if err != nil {
			failed = err
			return
		}
		defer h.Destroy()
		listenText(h, &in)
		defer forgetText(h)
		insert(h, "hi 👨‍👩‍👧🇫🇷1️⃣")
	})
	if failed != nil {
		t.Skipf("no window here: %v", failed)
	}
	inputfeed.Of(&in).BeginFrame()
	if got := string(in.Text()); got != "hi 👨‍👩‍👧🇫🇷1️⃣" {
		t.Fatalf("the family, the flag and the keycap arrive whole: %q", got)
	}
}

func TestTypedTextKeepsEmojiAndDropsControls(t *testing.T) {
	var in input.State
	addText(&in, "a\tb\x7f🧙‍♂️")
	inputfeed.Of(&in).BeginFrame()
	if got := string(in.Text()); got != "ab🧙‍♂️" {
		t.Fatalf("%q", got)
	}
}
