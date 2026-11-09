package net

import (
	"bufio"
	"context"
	"crypto/tls"
	"encoding/binary"
	"encoding/gob"
	"errors"
	"fmt"
	"io"
	stdnet "net"
	"sync"
	"time"
)

// Listener feeds a dedicated server its client connections, whatever the
// transport: TCP (Listen) or QUIC (package quic).
type Listener interface {
	Conns() <-chan stdnet.Conn
	Addr() string
	Close() error
}

// TCPListener accepts TCP connections.
type TCPListener struct {
	ln    stdnet.Listener
	conns chan stdnet.Conn
	done  chan struct{}
	once  sync.Once
}

// Listen starts accepting TCP connections on addr ("host:port", ":0" for
// any free port).
func Listen(addr string) (*TCPListener, error) {
	ln, err := (&stdnet.ListenConfig{}).Listen(context.Background(), "tcp", addr)
	if err != nil {
		return nil, err
	}
	return accepting(ln), nil
}

// ListenTLS accepts TLS over TCP on addr. Each handshake runs on its own,
// within handshakeTimeout, so a slow or plaintext client never holds the
// accept loop and only completed connections are yielded.
func ListenTLS(addr string, conf *tls.Config) (*TCPListener, error) {
	ln, err := (&stdnet.ListenConfig{}).Listen(context.Background(), "tcp", addr)
	if err != nil {
		return nil, err
	}
	conf = conf.Clone()
	conf.MinVersion = max(conf.MinVersion, tls.VersionTLS13)
	return accepting(tls.NewListener(ln, conf)), nil
}

// DialTLS connects with TLS over TCP, completing the handshake within ctx.
func DialTLS(ctx context.Context, addr string, conf *tls.Config) (stdnet.Conn, error) {
	conf = conf.Clone()
	conf.MinVersion = max(conf.MinVersion, tls.VersionTLS13)
	return (&tls.Dialer{Config: conf}).DialContext(ctx, "tcp", addr)
}

func accepting(ln stdnet.Listener) *TCPListener {
	l := &TCPListener{ln: ln, conns: make(chan stdnet.Conn, 16), done: make(chan struct{})}
	var handshakes sync.WaitGroup
	yield := func(c stdnet.Conn) {
		select {
		case l.conns <- c:
		case <-l.done:
			_ = c.Close()
		}
	}
	go func() {
		defer func() {
			handshakes.Wait()
			close(l.conns)
		}()
		for {
			c, err := ln.Accept()
			if err != nil {
				return
			}
			tc, ok := c.(*tls.Conn)
			if !ok {
				yield(c)
				continue
			}
			handshakes.Add(1)
			go func() {
				defer handshakes.Done()
				ctx, cancel := context.WithTimeout(context.Background(), handshakeTimeout)
				defer cancel()
				if tc.HandshakeContext(ctx) != nil {
					_ = tc.Close()
					return
				}
				yield(tc)
			}()
		}
	}()
	return l
}

const handshakeTimeout = 10 * time.Second

func (l *TCPListener) Addr() string { return l.ln.Addr().String() }

func (l *TCPListener) Close() error {
	l.once.Do(func() { close(l.done) })
	return l.ln.Close()
}

// Conns yields accepted connections.
func (l *TCPListener) Conns() <-chan stdnet.Conn { return l.conns }

// Dial connects to a dedicated server.
func Dial(addr string) (stdnet.Conn, error) {
	return (&stdnet.Dialer{}).DialContext(context.Background(), "tcp", addr)
}

// Pipe returns two connected in-memory ends, for an integrated server.
//
//nolint:nonamedreturns // the names tell the two ends apart
func Pipe() (server, client stdnet.Conn) { return stdnet.Pipe() }

// Codec turns messages into a byte stream and back. GobCodec, the default,
// carries any registered Go type; a game wanting a stable or cross-language
// wire format maps the wire messages and its own onto, for example,
// protobuf with Frames.
type Codec interface {
	NewEncoder(w io.Writer) Encoder
	NewDecoder(r io.Reader) Decoder
}

type Encoder interface{ Encode(msg any) error }

type Decoder interface{ Decode() (any, error) }

// GobCodec encodes registered Go values with encoding/gob.
type GobCodec struct{}

type envelope struct{ Msg any }

type gobEncoder struct{ *gob.Encoder }

func (e gobEncoder) Encode(m any) error { return e.Encoder.Encode(&envelope{m}) }

type gobDecoder struct{ *gob.Decoder }

func (d gobDecoder) Decode() (any, error) {
	var e envelope
	err := d.Decoder.Decode(&e)
	return e.Msg, err
}

func (GobCodec) NewEncoder(w io.Writer) Encoder { return gobEncoder{gob.NewEncoder(w)} }
func (GobCodec) NewDecoder(r io.Reader) Decoder { return gobDecoder{gob.NewDecoder(r)} }

// Frames is a Codec writing each message as one length-prefixed frame
// (uvarint size, then marshal's bytes). Frames larger than MaxSize (default
// 1 MiB) close the connection.
type Frames struct {
	Marshal   func(msg any) ([]byte, error)
	Unmarshal func(b []byte) (any, error)
	MaxSize   int
}

func (f Frames) MarshalMessage(m any) ([]byte, error)   { return f.Marshal(m) }
func (f Frames) UnmarshalMessage(b []byte) (any, error) { return f.Unmarshal(b) }

// DatagramConn is a connection that also carries unreliable datagrams, such
// as a QUIC connection.
type DatagramConn interface {
	SupportsDatagrams() bool
	SendDatagram(b []byte) error
	ReceiveDatagram(ctx context.Context) ([]byte, error)
}

// MessageCodec encodes one message per buffer, as datagrams need; Frames
// does.
type MessageCodec interface {
	MarshalMessage(m any) ([]byte, error)
	UnmarshalMessage(b []byte) (any, error)
}

func (f Frames) limit() int {
	if f.MaxSize > 0 {
		return f.MaxSize
	}
	return 1 << 20
}

type frameEncoder struct {
	f Frames
	w *bufio.Writer
}

func (e frameEncoder) Encode(m any) error {
	b, err := e.f.Marshal(m)
	if err != nil {
		return err
	}
	if len(b) > e.f.limit() {
		return fmt.Errorf("net: %T frame of %d bytes exceeds %d", m, len(b), e.f.limit())
	}
	var size [binary.MaxVarintLen64]byte
	if _, err = e.w.Write(size[:binary.PutUvarint(size[:], uint64(len(b)))]); err != nil {
		return err
	}
	if _, err = e.w.Write(b); err != nil {
		return err
	}
	return e.w.Flush()
}

type frameDecoder struct {
	f Frames
	r *bufio.Reader
}

func (d frameDecoder) Decode() (any, error) {
	n, err := binary.ReadUvarint(d.r)
	if err != nil {
		return nil, err
	}
	if n > uint64(d.f.limit()) { //nolint:gosec // limit is always positive
		return nil, fmt.Errorf("net: frame of %d bytes exceeds %d", n, d.f.limit())
	}
	b := make([]byte, n)
	if _, err = io.ReadFull(d.r, b); err != nil {
		return nil, err
	}
	return d.f.Unmarshal(b)
}

func (f Frames) NewEncoder(w io.Writer) Encoder { return frameEncoder{f, bufio.NewWriter(w)} }
func (f Frames) NewDecoder(r io.Reader) Decoder { return frameDecoder{f, bufio.NewReader(r)} }

// maxOutbox is how many messages may wait for a slow connection before it is
// dropped: about half a minute of snapshots.
const maxOutbox = 1024

// peer moves messages between a connection and the game loop without ever
// blocking the loop: a reader goroutine decodes into in, a writer goroutine
// drains an outbox capped at limit messages.
type peer struct {
	conn  stdnet.Conn
	codec Codec
	in    chan any
	limit int

	mu      sync.Mutex
	outbox  []any
	closing bool
	wake    chan struct{}
	done    chan struct{}
	once    sync.Once

	dg   DatagramConn
	mc   MessageCodec
	dgIn chan any
}

func newPeer(c stdnet.Conn, codec Codec) *peer {
	if codec == nil {
		codec = GobCodec{}
	}
	p := &peer{
		conn:  c,
		codec: codec,
		in:    make(chan any, 4096),
		limit: maxOutbox,
		wake:  make(chan struct{}, 1),
		done:  make(chan struct{}),
	}
	go p.read()
	go p.write()
	dg, isDatagram := c.(DatagramConn)
	mc, isMessage := codec.(MessageCodec)
	if isDatagram && dg.SupportsDatagrams() && isMessage {
		p.dg, p.mc, p.dgIn = dg, mc, make(chan any, 1024)
		go p.readDatagrams()
	}
	return p
}

func (p *peer) datagrams() bool { return p.dg != nil }

// sendDatagram drops m when it cannot be sent at once: it is sent again.
func (p *peer) sendDatagram(m any) {
	if p.dg == nil {
		p.send(m)
		return
	}
	if b, err := p.mc.MarshalMessage(m); err == nil {
		_ = p.dg.SendDatagram(b)
	}
}

func (p *peer) readDatagrams() {
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		<-p.done
		cancel()
	}()
	for {
		b, err := p.dg.ReceiveDatagram(ctx)
		if err != nil {
			return
		}
		m, err := p.mc.UnmarshalMessage(b)
		if err != nil {
			continue
		}
		select {
		case p.dgIn <- m:
		default:
		}
	}
}

func (p *peer) read() {
	defer close(p.in)
	dec := p.codec.NewDecoder(p.conn)
	for {
		m, err := dec.Decode()
		if err != nil {
			p.close()
			return
		}
		select {
		case p.in <- m:
		case <-p.done:
			return
		}
	}
}

func (p *peer) write() {
	enc := p.codec.NewEncoder(p.conn)
	for {
		select {
		case <-p.done:
			return
		case <-p.wake:
		}
		p.mu.Lock()
		batch := p.outbox
		p.outbox = nil
		closing := p.closing
		p.mu.Unlock()
		for _, m := range batch {
			if err := enc.Encode(m); err != nil {
				p.close()
				return
			}
		}
		if closing {
			p.close()
			return
		}
	}
}

// closeAfterFlush closes the connection once what is queued has been
// written, or after a second.
func (p *peer) closeAfterFlush() {
	p.mu.Lock()
	p.closing = true
	p.mu.Unlock()
	select {
	case p.wake <- struct{}{}:
	default:
	}
	time.AfterFunc(time.Second, p.close)
}

// send queues m, closing the connection instead once the outbox is full.
func (p *peer) send(m any) {
	Register(m)
	select {
	case <-p.done:
		return
	default:
	}
	p.mu.Lock()
	full := len(p.outbox) >= p.limit
	if !full {
		p.outbox = append(p.outbox, m)
	}
	p.mu.Unlock()
	if full {
		p.close()
		return
	}
	select {
	case p.wake <- struct{}{}:
	default:
	}
}

// poll returns pending messages; alive is false once the connection is gone
// and every message has been drained.
func (p *peer) poll() ([]any, bool) {
	var msgs []any
	for drained := p.dgIn == nil; !drained; {
		select {
		case m := <-p.dgIn:
			msgs = append(msgs, m)
		default:
			drained = true
		}
	}
	for {
		select {
		case m, ok := <-p.in:
			if !ok {
				return msgs, false
			}
			msgs = append(msgs, m)
		default:
			return msgs, true
		}
	}
}

func (p *peer) closed() <-chan struct{} { return p.done }

func (p *peer) close() {
	p.once.Do(func() {
		close(p.done)
		_ = p.conn.Close()
	})
}

// Merge accepts on several listeners as one, such as QUIC and TCP on the
// same port; Addr is the first one's.
func Merge(ls ...Listener) Listener {
	m := &merged{ls: ls, conns: make(chan stdnet.Conn, 16)}
	var wg sync.WaitGroup
	for _, l := range ls {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for c := range l.Conns() {
				m.conns <- c
			}
		}()
	}
	go func() {
		wg.Wait()
		close(m.conns)
	}()
	return m
}

type merged struct {
	ls    []Listener
	conns chan stdnet.Conn
}

func (m *merged) Conns() <-chan stdnet.Conn { return m.conns }
func (m *merged) Addr() string              { return m.ls[0].Addr() }

func (m *merged) Close() error {
	var errs []error
	for _, l := range m.ls {
		errs = append(errs, l.Close())
	}
	return errors.Join(errs...)
}
