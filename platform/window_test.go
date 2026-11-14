package platform //nolint:testpackage // tests windows through an unexported fake backend

import (
	"fmt"
	"image"
	"testing"

	"github.com/lantervale/ember/world"

	"github.com/lantervale/ember/color"
	"github.com/lantervale/ember/geom"
)

func TestNewWindowDefaults(t *testing.T) {
	w := NewWindow()
	if w.Size().X != 1280 || w.Aspect() != 1280.0/720 || w.Background().A != 1 {
		t.Fatalf("defaults %v %v", w.Size(), w.Background())
	}
	if in := w.Input(); in == nil || w.Input() != in {
		t.Fatal("each window owns a stable input state")
	}
}

func TestCaptureIsTakenOnce(t *testing.T) {
	w := NewWindow()
	w.Capture("a.png")
	if w.TakeCapture() != "a.png" || w.TakeCapture() != "" {
		t.Fatal("capture requests are one-shot")
	}
}

func TestWindowsTakeOptions(t *testing.T) {
	w := NewWindow()
	if w.Title() != "ember" || w.Size() != geom.V2(1280, 720) || w.Background() != color.RGB(0.015, 0.015, 0.02) {
		t.Fatalf("defaults: %q %v %v", w.Title(), w.Size(), w.Background())
	}
	if !w.VSync() || w.Hidden() || w.Undecorated() || w.Position() != (geom.Vec2{}) {
		t.Fatal("vsync on, shown, decorated by default, placed by the system")
	}
	w = NewWindow(
		Title("Map"),
		Size(420, 300),
		Position(10, 20),
		Background(color.Red),
		VSync(false),
		Hidden(),
		Undecorated(),
	)
	if w.Title() != "Map" || w.Size() != geom.V2(420, 300) || w.Position() != geom.V2(10, 20) ||
		w.Background() != color.Red {
		t.Fatalf("options: %q %v %v %v", w.Title(), w.Size(), w.Position(), w.Background())
	}
	if w.VSync() || !w.Hidden() || !w.Undecorated() {
		t.Fatal("flags")
	}
	if empty := NewWindow(Size(0, 0)); empty.Size() != geom.V2(1280, 720) {
		t.Fatal("an empty size keeps the default")
	}
}

func TestTheBackgroundChangesAtRuntime(t *testing.T) {
	w := NewWindow(Background(color.Red))
	w.SetBackground(color.Blue)
	if w.Background() != color.Blue {
		t.Fatal("SetBackground")
	}
}

func TestWindowsTakeTheirResizeAndStateOptions(t *testing.T) {
	w := NewWindow()
	if !w.Resizable() || w.AlwaysOnTop() || w.Fullscreen() || w.Maximized() || w.Minimized() ||
		w.MinSize() != (geom.Vec2{}) || w.MaxSize() != (geom.Vec2{}) || w.Icon() != nil {
		t.Fatal("resizable, windowed, unbounded and without an icon by default")
	}
	if aw, ah := w.AspectRatio(); aw != 0 || ah != 0 {
		t.Fatal("a free aspect ratio by default")
	}
	icon := image.NewRGBA(image.Rect(0, 0, 16, 16))
	w = NewWindow(FixedSize(), MinSize(320, 200), MaxSize(1920, 1080), AspectRatio(16, 9),
		Maximized(), Fullscreen(), AlwaysOnTop(), Icon(icon))
	if w.Resizable() || !w.AlwaysOnTop() || !w.Fullscreen() || !w.Maximized() ||
		w.MinSize() != geom.V2(
			320,
			200,
		) || w.MaxSize() != geom.V2(1920, 1080) || len(w.Icon()) != 1 || w.Icon()[0] != icon {
		t.Fatal("options")
	}
	if aw, ah := w.AspectRatio(); aw != 16 || ah != 9 {
		t.Fatal("aspect ratio")
	}
}

type fake struct {
	Backend
	calls []string
}

func (f *fake) recordf(format string, args ...any) {
	f.calls = append(f.calls, fmt.Sprintf(format, args...))
}
func (f *fake) SetResizable(on bool)           { f.recordf("resizable %v", on) }
func (f *fake) SetAlwaysOnTop(on bool)         { f.recordf("on top %v", on) }
func (f *fake) SetSizeLimits(lo, hi geom.Vec2) { f.recordf("limits %v %v", lo, hi) }
func (f *fake) SetAspectRatio(w, h int)        { f.recordf("aspect %d:%d", w, h) }
func (f *fake) Maximize()                      { f.recordf("maximize") }
func (f *fake) Minimize()                      { f.recordf("minimize") }
func (f *fake) Restore()                       { f.recordf("restore") }
func (f *fake) SetFullscreen(on bool)          { f.recordf("fullscreen %v", on) }
func (f *fake) SetIcon(images []image.Image)   { f.recordf("icon %d", len(images)) }
func (f *fake) SetCursor(c Cursor)             { f.recordf("cursor %d", c) }
func (f *fake) Clipboard() string              { return "copied" }
func (f *fake) SetClipboard(text string)       { f.recordf("clipboard %s", text) }

func TestWindowSettersReachTheOpenWindow(t *testing.T) {
	w := NewWindow()
	w.SetResizable(false)
	if w.Resizable() {
		t.Fatal("a closed window keeps the setting for when it opens")
	}
	f := &fake{}
	w.Attach(f)
	w.SetResizable(true)
	w.SetAlwaysOnTop(true)
	w.SetSizeLimits(geom.V2(100, 100), geom.Vec2{})
	w.SetAspectRatio(4, 3)
	w.Maximize()
	if !w.Maximized() || w.Minimized() {
		t.Fatal("maximized")
	}
	w.Minimize()
	if !w.Minimized() {
		t.Fatal("minimized")
	}
	w.Restore()
	if w.Maximized() || w.Minimized() {
		t.Fatal("restored")
	}
	w.SetFullscreen(true)
	w.SetIcon(image.NewRGBA(image.Rect(0, 0, 1, 1)))
	want := []string{
		"resizable true",
		"on top true",
		"limits {100 100} {0 0}",
		"aspect 4:3",
		"maximize",
		"minimize",
		"restore",
		"fullscreen true",
		"icon 1",
	}
	if fmt.Sprint(f.calls) != fmt.Sprint(want) {
		t.Fatalf("calls %q", f.calls)
	}
	if !w.Resizable() || !w.AlwaysOnTop() || !w.Fullscreen() || w.MinSize() != geom.V2(100, 100) {
		t.Fatal("the window keeps what it was set to")
	}
	w.SyncState(true, false)
	if !w.Maximized() || w.Minimized() {
		t.Fatal("the user maximizing the window shows")
	}
}

func TestAWindowAddsChildrenAndReturnsItself(t *testing.T) {
	win := NewWindow()
	a, b := world.NewBasic(), world.NewBasic()
	if win.Add(a, b) != win || a.Parent() != win.BasicEntity || b.Parent() != win.BasicEntity {
		t.Fatal("both children, and the window back for chaining")
	}
}

func TestTheCursorAndClipboardReachTheWindow(t *testing.T) {
	w := NewWindow()
	w.SetCursor(CursorHand)
	if w.Cursor() != CursorHand || w.Clipboard() != "" {
		t.Fatal("a closed window keeps its cursor and has no clipboard")
	}
	f := &fake{}
	w.Attach(f)
	w.SetCursor(CursorText)
	w.SetClipboard("hello")
	if w.Clipboard() != "copied" {
		t.Fatal("the system's clipboard")
	}
	want := []string{"cursor 3", "cursor 2", "clipboard hello"}
	if fmt.Sprint(f.calls) != fmt.Sprint(want) {
		t.Fatalf("calls %v, want %v", f.calls, want)
	}
}
