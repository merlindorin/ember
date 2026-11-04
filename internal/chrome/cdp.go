//go:build chrome

// Package chrome drives a headless Google Chrome through the DevTools
// protocol, in real time: for pages that draw over several frames, such
// as WebGPU ones, which Chrome's one-shot --screenshot catches too early
// (its virtual time doesn't wait for the GPU). The browser is EMBER_CHROME,
// else macOS's Google Chrome; EMBER_CHROME_FLAGS adds flags, such as a
// software WebGPU adapter on a CI runner without a GPU. LaunchFirefox drives
// Firefox the same way over WebDriver BiDi.
package chrome

import (
	"bufio"
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// bidiContext is the key BiDi commands name a browsing context by.
const bidiContext = "context"

// Path is the Chrome the package runs.
func Path() string {
	if p := os.Getenv("EMBER_CHROME"); p != "" {
		return p
	}
	return "/Applications/Google Chrome.app/Contents/MacOS/Google Chrome"
}

// Browser is a headless Chrome with one page, driven over a websocket.
type Browser struct {
	cmd  *exec.Cmd
	dir  string
	conn net.Conn
	r    *bufio.Reader
	id   int
	// bidi is a Firefox driven over WebDriver BiDi, its page the browsing
	// context ctx.
	bidi bool
	ctx  string
}

// Launch starts a headless Chrome with WebGPU on, a viewport of width by
// height points at scale pixels a point.
func Launch(width, height int, scale float64) (*Browser, error) {
	dir, err := os.MkdirTemp("", "ember-chrome-")
	if err != nil {
		return nil, err
	}
	b := &Browser{dir: dir}
	args := []string{"--headless=new", "--user-data-dir=" + dir, "--remote-debugging-port=0",
		"--enable-unsafe-webgpu", "--no-first-run", "--no-default-browser-check"}
	args = append(append(args, strings.Fields(os.Getenv("EMBER_CHROME_FLAGS"))...), "about:blank")
	//nolint:gosec // the browser the developer names in EMBER_CHROME, with the harness's own flags
	b.cmd = exec.CommandContext(context.Background(), Path(), args...)
	if err = b.cmd.Start(); err != nil {
		os.RemoveAll(dir)
		return nil, err
	}
	port, err := b.port()
	if err != nil {
		b.Close()
		return nil, err
	}
	ws, err := pageSocket(port)
	if err != nil {
		b.Close()
		return nil, err
	}
	if err = b.dial(ws); err != nil {
		b.Close()
		return nil, err
	}
	metrics := map[string]any{"width": width, "height": height, "deviceScaleFactor": scale, "mobile": false}
	if _, err = b.Call("Emulation.setDeviceMetricsOverride", metrics); err != nil {
		b.Close()
		return nil, err
	}
	return b, nil
}

// port waits for the port Chrome chose, written to DevToolsActivePort.
func (b *Browser) port() (string, error) {
	for range 100 {
		if data, err := os.ReadFile(filepath.Join(b.dir, "DevToolsActivePort")); err == nil {
			if line, _, _ := strings.Cut(string(data), "\n"); line != "" {
				return line, nil
			}
		}
		time.Sleep(100 * time.Millisecond)
	}
	return "", errors.New("chrome: no DevTools port")
}

func pageSocket(port string) (string, error) {
	list := "http://127.0.0.1:" + port + "/json/list"
	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, list, nil)
	if err != nil {
		return "", err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	var targets []struct {
		Type                 string `json:"type"`
		WebSocketDebuggerURL string `json:"webSocketDebuggerUrl"`
	}
	if err = json.NewDecoder(resp.Body).Decode(&targets); err != nil {
		return "", err
	}
	for _, t := range targets {
		if t.Type == "page" {
			return t.WebSocketDebuggerURL, nil
		}
	}
	return "", errors.New("chrome: no page target")
}

func (b *Browser) dial(ws string) error {
	u, err := url.Parse(ws)
	if err != nil {
		return err
	}
	var d net.Dialer
	b.conn, err = d.DialContext(context.Background(), "tcp", u.Host)
	if err != nil {
		return err
	}
	key := make([]byte, 16)
	rand.Read(key)
	fmt.Fprintf(b.conn, "GET %s HTTP/1.1\r\nHost: %s\r\nUpgrade: websocket\r\nConnection: Upgrade\r\n"+
		"Sec-WebSocket-Key: %s\r\nSec-WebSocket-Version: 13\r\n\r\n",
		u.Path, u.Host, base64.StdEncoding.EncodeToString(key))
	b.r = bufio.NewReader(b.conn)
	//nolint:bodyclose // a 101 response has no body: the connection goes on as the websocket, read by message
	resp, err := http.ReadResponse(b.r, nil)
	if err != nil {
		return err
	}
	if resp.StatusCode != http.StatusSwitchingProtocols {
		return fmt.Errorf("chrome: websocket handshake: %s", resp.Status)
	}
	return nil
}

// Call sends a DevTools command and waits for its result, skipping the
// events in between.
func (b *Browser) Call(method string, params any) (json.RawMessage, error) {
	b.id++
	msg, err := json.Marshal(map[string]any{"id": b.id, "method": method, "params": params})
	if err != nil {
		return nil, err
	}
	frame := []byte{0x81}
	switch n := len(msg); {
	case n < 126:
		frame = append(frame, byte(n)|0x80)
	case n < 1<<16:
		//nolint:gosec // the 16-bit length, big-endian: n is under 1<<16 here
		frame = append(frame, 126|0x80, byte(n>>8), byte(n))
	default:
		frame = binary.BigEndian.AppendUint64(append(frame, 127|0x80), uint64(n))
	}
	mask := make([]byte, 4)
	rand.Read(mask)
	frame = append(frame, mask...)
	for i, c := range msg {
		frame = append(frame, c^mask[i%4])
	}
	if _, err = b.conn.Write(frame); err != nil {
		return nil, err
	}
	for {
		data, readErr := b.message()
		if readErr != nil {
			return nil, readErr
		}
		var m struct {
			ID      int             `json:"id"`
			Result  json.RawMessage `json:"result"`
			Error   json.RawMessage `json:"error"`
			Message string          `json:"message"`
		}
		if json.Unmarshal(data, &m) != nil || m.ID != b.id {
			continue
		}
		if len(m.Error) != 0 {
			var cdp struct {
				Message string `json:"message"`
			}
			if json.Unmarshal(m.Error, &cdp) == nil {
				return nil, fmt.Errorf("chrome: %s: %s", method, cdp.Message)
			}
			return nil, fmt.Errorf("firefox: %s: %s: %s", method, m.Error, m.Message)
		}
		return m.Result, nil
	}
}

// message reads one websocket message, joining its frames.
func (b *Browser) message() ([]byte, error) {
	var out []byte
	for {
		h := make([]byte, 2)
		if _, err := io.ReadFull(b.r, h); err != nil {
			return nil, err
		}
		n := uint64(h[1] & 0x7f)
		switch n {
		case 126:
			var l [2]byte
			if _, err := io.ReadFull(b.r, l[:]); err != nil {
				return nil, err
			}
			n = uint64(binary.BigEndian.Uint16(l[:]))
		case 127:
			var l [8]byte
			if _, err := io.ReadFull(b.r, l[:]); err != nil {
				return nil, err
			}
			n = binary.BigEndian.Uint64(l[:])
		}
		p := make([]byte, n)
		if _, err := io.ReadFull(b.r, p); err != nil {
			return nil, err
		}
		out = append(out, p...)
		if h[0]&0x80 != 0 {
			return out, nil
		}
	}
}

// Navigate loads url.
func (b *Browser) Navigate(url string) error {
	if b.bidi {
		_, err := b.Call("browsingContext.navigate", map[string]any{bidiContext: b.ctx, "url": url, "wait": "complete"})
		return err
	}
	_, err := b.Call("Page.navigate", map[string]any{"url": url})
	return err
}

// Eval is expr's value in the page, decoded into v.
func (b *Browser) Eval(expr string, v any) error {
	if b.bidi {
		return b.evalBiDi(expr, v)
	}
	res, err := b.Call("Runtime.evaluate", map[string]any{"expression": expr, "returnByValue": true})
	if err != nil {
		return err
	}
	var r struct {
		Result struct {
			Value json.RawMessage `json:"value"`
		} `json:"result"`
	}
	if err = json.Unmarshal(res, &r); err != nil {
		return err
	}
	if len(r.Result.Value) == 0 {
		return nil
	}
	return json.Unmarshal(r.Result.Value, v)
}

// Wait polls expr every 100 ms until it is a non-empty string, which it
// returns, or timeout passes.
func (b *Browser) Wait(expr string, timeout time.Duration) (string, error) {
	for end := time.Now().Add(timeout); time.Now().Before(end); time.Sleep(100 * time.Millisecond) {
		var s string
		if err := b.Eval(expr, &s); err != nil {
			return "", err
		}
		if s != "" {
			return s, nil
		}
	}
	return "", fmt.Errorf("chrome: %s still empty after %v", expr, timeout)
}

// Screenshot is the page's picture as PNG bytes.
func (b *Browser) Screenshot() ([]byte, error) {
	method, params := "Page.captureScreenshot", map[string]any{"format": "png"}
	if b.bidi {
		method, params = "browsingContext.captureScreenshot", map[string]any{bidiContext: b.ctx}
	}
	res, err := b.Call(method, params)
	if err != nil {
		return nil, err
	}
	var shot struct {
		Data string `json:"data"`
	}
	if err = json.Unmarshal(res, &shot); err != nil {
		return nil, err
	}
	return base64.StdEncoding.DecodeString(shot.Data)
}

// Close stops the browser and removes its profile.
func (b *Browser) Close() {
	if b.conn != nil {
		b.conn.Close()
	}
	if b.cmd != nil && b.cmd.Process != nil {
		_ = b.cmd.Process.Kill()
		_ = b.cmd.Wait()
	}
	os.RemoveAll(b.dir)
}
