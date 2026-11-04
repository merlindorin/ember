package twod

import "github.com/merlindorin/ember/internal/draw"

// Mark is a point in what a Canvas draws, to record from.
type Mark struct{ m draw.Mark }

// Recording is what a Canvas drew from a Mark, kept to draw again: a UI
// host's last frame, replayed when nothing in it changed. Games draw their
// entities each frame and never need one.
type Recording struct{ r draw.Recording }

// Mark is where u's drawing is now.
func (u *Canvas) Mark() Mark { return Mark{u.list.Mark()} }

// Keep keeps what u drew since m in r, reusing its room, and reports
// whether Replay can draw it again: not when it was lit, masked or a
// pass's picture, or when u's transform, modulate, blend or clip moved
// since m.
func (u *Canvas) Keep(m Mark, r *Recording) bool { return u.list.Record(m.m, &r.r) }

// Replay draws r again, at the same place, and reports whether it did:
// not when r holds nothing replayable or u's transform, modulate, blend or
// clip are not what they were when r was recorded.
func (u *Canvas) Replay(r *Recording) bool { return u.list.Replay(&r.r) }
