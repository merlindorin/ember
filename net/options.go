package net

import "time"

// ServerOption sets up a Server.
type ServerOption interface{ server(*Server) }

// ClientOption sets up a Client.
type ClientOption interface{ client(*Client) }

// Option sets up a Server or a Client alike.
type Option interface {
	ServerOption
	ClientOption
}

type serverOption func(*Server)

func (o serverOption) server(s *Server) { o(s) }

type clientOption func(*Client)

func (o clientOption) client(c *Client) { o(c) }

type option struct {
	s func(*Server)
	c func(*Client)
}

func (o option) server(s *Server) { o.s(s) }
func (o option) client(c *Client) { o.c(c) }

// WithCodec encodes the connection (default GobCodec).
func WithCodec(codec Codec) Option {
	return option{func(s *Server) { s.codec = codec }, func(c *Client) { c.codec = codec }}
}

// TickRate is the server's simulation rate: the snapshots it sends per
// second (default 20), and the step at which a client advances Predicted
// entities (default 30).
func TickRate(rate int) Option {
	if rate <= 0 {
		return option{func(*Server) {}, func(*Client) {}}
	}
	return option{func(s *Server) { s.interval = 1 / float32(rate) }, func(c *Client) { c.step = 1 / float32(rate) }}
}

// WithListener feeds the server the connections l accepts.
func WithListener(l Listener) ServerOption { return serverOption(func(s *Server) { s.listener = l }) }

// Interest limits replication to radius around each client's focus
// (default 60), on the ground plane, and despawns an entity only beyond
// radius+margin (default 8). Clients without a focus receive everything.
func Interest(radius, margin float32) ServerOption {
	return serverOption(func(s *Server) {
		if radius > 0 {
			s.radius = radius
		}
		if margin > 0 {
			s.margin = margin
		}
	})
}

// MaxIntents caps the intents a client may have queued between two
// fixed steps; a client exceeding it is disconnected (default 256).
func MaxIntents(n int) ServerOption {
	return serverOption(func(s *Server) {
		if n > 0 {
			s.maxIntents = n
		}
	})
}

// Region is the high part of every ID the server assigns.
func Region(region uint16) ServerOption {
	return serverOption(func(s *Server) { s.region = region })
}

// RequiredFocus replicates nothing to a client until it owns or focuses
// an entity, so connections that have not joined see no world.
func RequiredFocus() ServerOption { return serverOption(func(s *Server) { s.needFocus = true }) }

// SnapshotBudget caps the spawns and updates in one snapshot (default
// 256). The client's own entities go first, then the nearest; the rest wait
// for the next snapshot. Despawns are never held back.
func SnapshotBudget(n int) ServerOption {
	return serverOption(func(s *Server) {
		if n > 0 {
			s.budget = n
		}
	})
}

// Correction is the rate at which a misprediction is blended away
// (default 20 per second: 95% within 150 ms).
func Correction(rate float32) ClientOption {
	return clientOption(func(c *Client) {
		if rate > 0 {
			c.blend = rate
		}
	})
}

// SnapDistance is how far one predicted step must go to be a jump, such as
// a dash, shown at once instead of blended over the step (default 1, a
// metre in 3D; a 2D game passes it in its own units, above what a walk
// covers in one step).
func SnapDistance(d float32) ClientOption {
	return clientOption(func(c *Client) {
		if d > 0 {
			c.snap = d
		}
	})
}

// Hello opens the connection with msg (a join ticket, say). It is sent
// first, with sequence number 0, so it is never acknowledged nor replayed.
func Hello(msg any) ClientOption { return clientOption(func(c *Client) { c.hello = msg }) }

// WithEndpoint connects through the client end of a Lag pair instead of a
// connection.
func WithEndpoint(e Endpoint) ClientOption { return clientOption(func(c *Client) { c.peer = e.l }) }

// Clock times round trips and the prediction replay (default
// time.Now); tests drive it with the Lag pair's clock.
func Clock(now func() time.Time) ClientOption {
	return clientOption(func(c *Client) { c.now = now })
}
