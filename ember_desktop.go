//go:build !js

package ember

import (
	"github.com/merlindorin/ember/platform/desktop"
	"github.com/merlindorin/ember/world"
)

// runnerPlugin runs the World in native windows.
func runnerPlugin(crashLog string) world.Plugin { return desktop.Plugin{CrashLog: crashLog} }
