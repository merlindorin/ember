// Package errlog keeps the error lines the engine prints, for the crash
// log: a windowed Windows build has no console, so what only went to
// stderr would be lost. While To has a file, each line is written there
// as it comes, so it survives whatever ends the process.
package errlog

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

// keep is how many of the last lines Lines remembers.
const keep = 50

//nolint:gochecknoglobals // a process has one error log
var log struct {
	sync.Mutex
	lines []string
	path  string
	f     *os.File
}

// Print prints line to stderr and keeps it, and writes it to To's file.
func Print(line string) {
	fmt.Fprintln(os.Stderr, line)
	log.Lock()
	defer log.Unlock()
	log.lines = append(log.lines, line)
	if len(log.lines) > keep {
		log.lines = log.lines[len(log.lines)-keep:]
	}
	if log.path == "" {
		return
	}
	if log.f == nil {
		if err := os.MkdirAll(filepath.Dir(log.path), 0o755); err != nil {
			return
		}
		f, err := os.Create(log.path)
		if err != nil {
			return
		}
		log.f = f
	}
	fmt.Fprintln(log.f, line)
	_ = log.f.Sync() // best effort: the log never stops the game
}

// Lines are the last lines printed, oldest first.
func Lines() []string {
	log.Lock()
	defer log.Unlock()
	return append([]string(nil), log.lines...)
}

// To writes the lines printed from now on to path, made with the first
// one; a run that prints none leaves no file, and removes the last run's.
// stop closes it.
func To(path string) func() {
	log.Lock()
	defer log.Unlock()
	os.Remove(path)
	log.path, log.lines = path, nil
	return func() {
		log.Lock()
		defer log.Unlock()
		if log.f != nil {
			log.f.Close()
		}
		log.path, log.f = "", nil
	}
}
