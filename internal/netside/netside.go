// Package netside is how engine packages find the net.Server or net.Client
// a World was given. Games hold theirs instead.
package netside

import "github.com/merlindorin/ember/world"

type sides struct{ server, client any }

func of(w *world.World) *sides { return world.ResourceOrInit[sides](w) }

// SetServer records the World's server.
func SetServer(w *world.World, s any) { of(w).server = s }

// SetClient records the World's client.
func SetClient(w *world.World, c any) { of(w).client = c }

// Server is the World's server, nil without one.
func Server(w *world.World) any { return of(w).server }

// Client is the World's client, nil without one.
func Client(w *world.World) any { return of(w).client }
