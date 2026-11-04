//go:build !windows

package desktop

// showError does nothing outside Windows, where a crash is printed to the
// terminal that started the game.
func showError(_, _ string) {}
