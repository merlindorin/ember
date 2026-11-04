//go:build js

package ember

import (
	"github.com/merlindorin/ember/platform/web"
	"github.com/merlindorin/ember/world"
)

// runnerPlugin runs the World in the page's canvas; a browser has no
// crash log file, so the error shows on the page instead.
func runnerPlugin(string) world.Plugin { return web.Plugin{} }
