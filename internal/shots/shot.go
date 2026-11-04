// Package shots makes the guide's pictures: each Shot runs a package
// hidden and captures it (EMBER_CAPTURE and its scripts), then crops,
// resizes or lines up the capture and writes it under docs/. The list is
// Shots, in shots_list.go; task docs:shots runs it through ./run.
package shots

// A Shot is one picture of the guide.
type Shot struct {
	// Out is the picture's path under docs/: a .png or a .gif.
	Out string `json:"out"`
	// Run is the package captured, from ember's root, such as
	// "./examples/20-sprite-sheet", with Args on its command line.
	Run  string   `json:"run"`
	Args []string `json:"args"`
	// Env is added to the run's environment: the EMBER_CAPTURE_* scripts
	// (KEYS, MOUSE, TEXT, PAD, TOUCH, SIZE, FRAMES, EVERY) and anything
	// else. EMBER_HIDDEN, EMBER_SCALE and EMBER_CAPTURE are the runner's.
	Env map[string]string `json:"env"`
	// Frame is EMBER_CAPTURE_FRAME: 30 for a picture unless set, the
	// first recorded frame for a GIF.
	Frame int `json:"frame"`
	// Window is which window's capture is kept, the primary's by default.
	Window int `json:"window"`
	// Crop is x, y, w, h in the capture's pixels, taken before resizing;
	// the zero Crop keeps it whole.
	Crop [4]int `json:"crop"`
	// Size is the picture's size in points; the zero Size halves a .png
	// from the capture's pixels and keeps a .gif as recorded.
	Size [2]int `json:"size"`
	// Panels makes a light strip: each lightpanels -panel captured at
	// Frame (10 unless set), sized to 320×240 points, side by side.
	Panels []string `json:"panels"`
	// Copy is a file copied as the picture, from ember's root.
	Copy string `json:"copy"`
	// Files are copied, from ember's root, into a folder of the shot's
	// own, and Write's files written there; {tmp} in Args and Env names it.
	Files []string          `json:"files"`
	Write map[string]string `json:"write"`
}

// The settings a Shot's Env sets most, as the desktop reads them
// (EMBER_CAPTURE_* in platform/desktop), and the tileset tool's folder.
const (
	CaptureFrames = "EMBER_CAPTURE_FRAMES"
	CaptureEvery  = "EMBER_CAPTURE_EVERY"
	CaptureKeys   = "EMBER_CAPTURE_KEYS"
	CaptureMouse  = "EMBER_CAPTURE_MOUSE"
	CaptureText   = "EMBER_CAPTURE_TEXT"
	TilesetDir    = "EMBER_TILESET_DIR"
)
