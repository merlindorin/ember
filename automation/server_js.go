//go:build js

package automation

import "errors"

// serve can't listen in a page: a browser build records its widgets and
// actions for the game, and serves nothing.
func serve(_, _ string, _ *controller) (func(), string, error) {
	return nil, "", errors.New("a page can't serve automation")
}
