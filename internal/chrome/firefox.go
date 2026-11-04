//go:build chrome

package chrome

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// FirefoxPath is the Firefox LaunchFirefox runs: EMBER_FIREFOX, else
// macOS's Firefox.
func FirefoxPath() string {
	if p := os.Getenv("EMBER_FIREFOX"); p != "" {
		return p
	}
	return "/Applications/Firefox.app/Contents/MacOS/firefox"
}

// LaunchFirefox starts a headless Firefox on a fresh profile, driven over
// WebDriver BiDi, with a viewport of width by height points at scale pixels
// a point. The same Navigate, Eval, Wait and Screenshot work on it.
func LaunchFirefox(width, height int, scale float64) (*Browser, error) {
	dir, err := os.MkdirTemp("", "ember-firefox-")
	if err != nil {
		return nil, err
	}
	b := &Browser{dir: dir, bidi: true}
	prefs := "user_pref(\"browser.shell.checkDefaultBrowser\", false);\n" +
		"user_pref(\"datareporting.policy.dataSubmissionEnabled\", false);\n" +
		os.Getenv("EMBER_FIREFOX_PREFS")
	//nolint:gosec // dir is the fresh temporary profile made above, user.js a fixed name
	if err = os.WriteFile(filepath.Join(dir, "user.js"), []byte(prefs), 0o600); err != nil {
		os.RemoveAll(dir)
		return nil, err
	}
	args := []string{"--headless", "--no-remote", "--profile", dir, "--remote-debugging-port=0", "about:blank"}
	if os.Getenv("EMBER_FIREFOX_WINDOW") != "" {
		args = args[1:]
	}
	//nolint:gosec // the Firefox the developer names in EMBER_FIREFOX, with the harness's own flags
	b.cmd = exec.CommandContext(context.Background(), FirefoxPath(), args...)
	stderr, err := b.cmd.StderrPipe()
	if err != nil {
		os.RemoveAll(dir)
		return nil, err
	}
	if err = b.cmd.Start(); err != nil {
		os.RemoveAll(dir)
		return nil, err
	}
	ws := make(chan string, 1)
	go func() {
		lines := bufio.NewScanner(stderr)
		for lines.Scan() {
			if _, at, ok := strings.Cut(lines.Text(), "WebDriver BiDi listening on "); ok {
				ws <- at
			}
		}
	}()
	var at string
	select {
	case at = <-ws:
	case <-time.After(30 * time.Second):
		b.Close()
		return nil, errors.New("firefox: no WebDriver BiDi port")
	}
	if err = b.dial(at + "/session"); err != nil {
		b.Close()
		return nil, err
	}
	if _, err = b.Call("session.new", map[string]any{"capabilities": map[string]any{}}); err != nil {
		b.Close()
		return nil, err
	}
	res, err := b.Call("browsingContext.getTree", map[string]any{})
	if err != nil {
		b.Close()
		return nil, err
	}
	var tree struct {
		Contexts []struct {
			Context string `json:"context"`
		} `json:"contexts"`
	}
	if err = json.Unmarshal(res, &tree); err != nil || len(tree.Contexts) == 0 {
		b.Close()
		return nil, errors.New("firefox: no browsing context")
	}
	b.ctx = tree.Contexts[0].Context
	if _, err = b.Call("browsingContext.setViewport", map[string]any{bidiContext: b.ctx,
		"viewport": map[string]any{"width": width, "height": height}, "devicePixelRatio": scale}); err != nil {
		b.Close()
		return nil, err
	}
	return b, nil
}

func (b *Browser) evalBiDi(expr string, v any) error {
	res, err := b.Call("script.evaluate", map[string]any{"expression": expr, "target": map[string]any{bidiContext: b.ctx},
		"awaitPromise": true, "resultOwnership": "none"})
	if err != nil {
		return err
	}
	var r struct {
		Type             string      `json:"type"`
		Result           remoteValue `json:"result"`
		ExceptionDetails *struct {
			Text string `json:"text"`
		} `json:"exceptionDetails"`
	}
	if err = json.Unmarshal(res, &r); err != nil {
		return err
	}
	if r.ExceptionDetails != nil {
		return fmt.Errorf("firefox: %s: %s", expr, r.ExceptionDetails.Text)
	}
	data, err := json.Marshal(r.Result.plain())
	if err != nil {
		return err
	}
	return json.Unmarshal(data, v)
}

// remoteValue is a BiDi script value, made plain by plain.
type remoteValue struct {
	Type  string          `json:"type"`
	Value json.RawMessage `json:"value"`
}

func (r remoteValue) plain() any {
	switch r.Type {
	case "string", "number", "boolean":
		var v any
		_ = json.Unmarshal(r.Value, &v)
		return v
	case "array":
		var items []remoteValue
		_ = json.Unmarshal(r.Value, &items)
		out := make([]any, len(items))
		for i, it := range items {
			out[i] = it.plain()
		}
		return out
	case "object":
		var pairs [][2]json.RawMessage
		_ = json.Unmarshal(r.Value, &pairs)
		out := map[string]any{}
		for _, p := range pairs {
			var k string
			_ = json.Unmarshal(p[0], &k)
			var v remoteValue
			_ = json.Unmarshal(p[1], &v)
			out[k] = v.plain()
		}
		return out
	}
	return nil
}
