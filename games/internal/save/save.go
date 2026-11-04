// Package save keeps a game's leaderboard and sound settings in a JSON file,
// <dir>/<game>.json, dir being $EMBER_GAMES_DIR or the user's config folder.
package save

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"time"
)

// Places is how many scores the leaderboard keeps.
const Places = 10

// An Entry is a score on the leaderboard.
type Entry struct {
	Score int       `json:"score"`
	When  time.Time `json:"when"`
}

// Data is what a game keeps between runs.
type Data struct {
	Scores     []Entry `json:"scores"`
	Music      float32 `json:"music"`
	Effects    float32 `json:"effects"`
	MusicOff   bool    `json:"musicOff"`
	Fullscreen bool    `json:"fullscreen"`
}

// Defaults are a first run's settings.
func Defaults() Data { return Data{Music: 0.6, Effects: 0.8} }

// Add puts a score on the leaderboard and returns its place, from 1, or 0
// when it is below the last kept.
func (d *Data) Add(score int, when time.Time) int {
	i, _ := slices.BinarySearchFunc(d.Scores, score, func(e Entry, s int) int {
		switch {
		case e.Score > s:
			return -1
		case e.Score < s:
			return 1
		default:
			return -1 // an equal score already there stays above
		}
	})
	if i >= Places {
		return 0
	}
	d.Scores = slices.Insert(d.Scores, i, Entry{Score: score, When: when})
	if len(d.Scores) > Places {
		d.Scores = d.Scores[:Places]
	}
	return i + 1
}

// Path is where a game's data lives.
func Path(game string) (string, error) {
	dir := os.Getenv("EMBER_GAMES_DIR")
	if dir == "" {
		config, err := os.UserConfigDir()
		if err != nil {
			return "", fmt.Errorf("save: %w", err)
		}
		dir = filepath.Join(config, "ember-games")
	}
	return filepath.Join(dir, game+".json"), nil
}

// Load reads a game's data, or the defaults when it has none yet.
func Load(game string) (Data, error) {
	path, err := Path(game)
	if err != nil {
		return Defaults(), err
	}
	body, err := os.ReadFile(path)
	if errors.Is(err, fs.ErrNotExist) {
		return Defaults(), nil
	}
	if err != nil {
		return Defaults(), fmt.Errorf("save: %w", err)
	}
	d := Defaults()
	if err = json.Unmarshal(body, &d); err != nil {
		return Defaults(), fmt.Errorf("save: %s: %w", path, err)
	}
	return d, nil
}

// Store writes a game's data.
func Store(game string, d Data) error {
	path, err := Path(game)
	if err != nil {
		return err
	}
	if err = os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
		return fmt.Errorf("save: %w", err)
	}
	body, err := json.MarshalIndent(d, "", "  ")
	if err != nil {
		return fmt.Errorf("save: %w", err)
	}
	if err = os.WriteFile(path, body, 0o600); err != nil {
		return fmt.Errorf("save: %w", err)
	}
	return nil
}
