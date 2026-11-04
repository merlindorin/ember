package net

import (
	"math/rand/v2"
	"slices"
	"sync"
	"time"
)

// link carries messages to and from one peer: a connection (peer) or an
// in-memory Lag endpoint.
type link interface {
	send(m any)
	poll() (msgs []any, alive bool)
	close()
	closeAfterFlush()
	// closed is done once the link is closed.
	closed() <-chan struct{}
	// datagrams reports whether sendDatagram may lose or reorder messages
	// instead of queueing them on the reliable stream.
	datagrams() bool
	// sendDatagram sends m unreliably, without waiting for what is queued.
	sendDatagram(m any)
}

// Endpoint is one end of a Lag pair, for Server.ConnectEndpoint and
// WithEndpoint.
type Endpoint struct{ l *lagLink }

// Close drops the connection, as a network failure would: what is still
// in flight is lost.
func (e Endpoint) Close() { e.l.close() }

// Lag returns two connected endpoints that deliver every message after
// delay, as read on now. They encode nothing and start no goroutine, so a
// test driving both apps and the clock in lockstep is deterministic,
// latency included.
//
//nolint:nonamedreturns // the names tell the two ends apart
func Lag(now func() time.Time, delay time.Duration) (server, client Endpoint) {
	return LagWith(now, LagOptions{Delay: delay})
}

// LagJitter is Lag with each message delayed by an extra random share of
// jitter, as a real network does, delivered in order.
//
//nolint:nonamedreturns // the names tell the two ends apart
func LagJitter(now func() time.Time, delay, jitter time.Duration, seed uint64) (server, client Endpoint) {
	return LagWith(now, LagOptions{Delay: delay, Jitter: jitter, Seed: seed})
}

// LagOptions shape a Lag pair's network.
type LagOptions struct {
	Delay time.Duration
	// Jitter adds a random share of itself to each message's delay.
	Jitter time.Duration
	// Loss is the chance a packet is lost. On the reliable stream a loss
	// holds that message and every later one back by a retransmission, as
	// TCP does; a lost datagram is gone.
	Loss float64
	Seed uint64
	// Datagrams offers an unreliable channel next to the stream, as QUIC
	// does: datagrams may be lost and arrive out of order.
	Datagrams bool
}

// retransmit is how much later a lost reliable packet arrives.
const retransmit = 200 * time.Millisecond

//nolint:nonamedreturns // the names tell the two ends apart
func LagWith(now func() time.Time, o LagOptions) (server, client Endpoint) {
	//nolint:gosec // a seeded simulated network, reproducible in tests
	w := &lagWire{now: now, o: o, rng: rand.New(rand.NewPCG(o.Seed, 1))}
	a, b := &lagLink{wire: w}, &lagLink{wire: w}
	a.peer, b.peer = b, a
	return Endpoint{a}, Endpoint{b}
}

type lagWire struct {
	mu     sync.Mutex
	now    func() time.Time
	o      LagOptions
	rng    *rand.Rand
	closed bool
}

type delivery struct {
	at  time.Time
	msg any
}

type lagLink struct {
	wire *lagWire
	peer *lagLink
	// inbox is sorted by arrival time.
	inbox   []delivery
	stream  time.Time
	closing bool
}

func (w *lagWire) arrival() time.Time {
	at := w.now().Add(w.o.Delay)
	if w.o.Jitter > 0 {
		at = at.Add(time.Duration(w.rng.Int64N(int64(w.o.Jitter))))
	}
	return at
}

func (l *lagLink) deliver(d delivery) {
	in := l.peer.inbox
	i := len(in)
	for i > 0 && in[i-1].at.After(d.at) {
		i--
	}
	l.peer.inbox = slices.Insert(in, i, d)
}

func (l *lagLink) send(m any) {
	w := l.wire
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.closed || l.closing {
		return
	}
	at := w.arrival()
	if w.o.Loss > 0 && w.rng.Float64() < w.o.Loss {
		at = at.Add(retransmit)
	}
	if at.Before(l.stream) {
		at = l.stream
	}
	l.stream = at
	l.deliver(delivery{at, m})
}

func (l *lagLink) datagrams() bool { return l.wire.o.Datagrams }

func (l *lagLink) sendDatagram(m any) {
	w := l.wire
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.closed || l.closing || w.o.Loss > 0 && w.rng.Float64() < w.o.Loss {
		return
	}
	l.deliver(delivery{w.arrival(), m})
}

func (l *lagLink) poll() ([]any, bool) {
	w := l.wire
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.closed {
		return nil, false
	}
	now := w.now()
	var msgs []any
	i := 0
	for ; i < len(l.inbox) && !l.inbox[i].at.After(now); i++ {
		msgs = append(msgs, l.inbox[i].msg)
	}
	l.inbox = l.inbox[i:]
	if l.peer.closing && len(l.inbox) == 0 {
		w.closed = true
		return msgs, false
	}
	return msgs, true
}

func (l *lagLink) close() {
	l.wire.mu.Lock()
	defer l.wire.mu.Unlock()
	l.wire.closed = true
}

// closed is done at once: a Lag link delivers what it holds as the clock
// advances, with nothing left to write.
func (l *lagLink) closed() <-chan struct{} {
	done := make(chan struct{})
	close(done)
	return done
}

// closeAfterFlush lets what was already sent arrive, then closes.
func (l *lagLink) closeAfterFlush() {
	l.wire.mu.Lock()
	defer l.wire.mu.Unlock()
	l.closing = true
}
