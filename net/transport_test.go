package net_test

import (
	"context"
	"io"
	stdnet "net"
	"testing"
	"time"

	"github.com/merlindorin/ember/net"
	"github.com/merlindorin/ember/net/nettest"
	"github.com/merlindorin/ember/net/quic"
)

func TestPipeTransport(t *testing.T) {
	nettest.Run(t, func(*testing.T) (stdnet.Conn, stdnet.Conn) { return net.Pipe() })
}

func TestTCPTransport(t *testing.T) {
	nettest.Run(t, func(t *testing.T) (stdnet.Conn, stdnet.Conn) {
		ln, err := net.Listen("127.0.0.1:0")
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { ln.Close() })
		c, err := net.Dial(ln.Addr())
		if err != nil {
			t.Fatal(err)
		}
		return <-ln.Conns(), c
	})
}

func TestMergedListenersAcceptOnEveryOne(t *testing.T) {
	a, err := net.Listen("127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	b, err := net.Listen("127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	l := net.Merge(a, b)
	defer l.Close()
	if l.Addr() != a.Addr() {
		t.Fatalf("merged address %s, want the first one's %s", l.Addr(), a.Addr())
	}
	for _, addr := range []string{a.Addr(), b.Addr()} {
		c, dialErr := net.Dial(addr)
		if dialErr != nil {
			t.Fatal(dialErr)
		}
		defer c.Close()
		select {
		case s := <-l.Conns():
			s.Close()
		case <-time.After(2 * time.Second):
			t.Fatalf("no connection from %s", addr)
		}
	}
	l.Close()
	select {
	case _, open := <-l.Conns():
		if open {
			t.Fatal("a connection arrived after Close")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("the merged channel stays open after Close")
	}
}

func tlsPair(t *testing.T) (*net.TCPListener, string) {
	t.Helper()
	conf, pin, err := quic.SelfSigned("127.0.0.1")
	if err != nil {
		t.Fatal(err)
	}
	ln, err := net.ListenTLS("127.0.0.1:0", conf)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { ln.Close() })
	return ln, pin
}

func TestTLSTransport(t *testing.T) {
	nettest.Run(t, func(t *testing.T) (stdnet.Conn, stdnet.Conn) {
		ln, pin := tlsPair(t)
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		c, err := net.DialTLS(ctx, ln.Addr(), quic.Pinned(pin))
		if err != nil {
			t.Fatal(err)
		}
		return <-ln.Conns(), c
	})
}

func TestTLSRefusesAWrongPinAndPlaintext(t *testing.T) {
	ln, _ := tlsPair(t)
	_, other, _ := quic.SelfSigned("127.0.0.1")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if c, err := net.DialTLS(ctx, ln.Addr(), quic.Pinned(other)); err == nil {
		c.Close()
		t.Fatal("a wrong pin was accepted")
	}
	plain, err := net.Dial(ln.Addr())
	if err != nil {
		t.Fatal(err)
	}
	defer plain.Close()
	_, _ = plain.Write([]byte("hello in clear, which is no TLS record\n"))
	_ = plain.SetReadDeadline(time.Now().Add(3 * time.Second))
	if _, err = io.ReadAll(plain); err != nil {
		t.Fatalf("the shard kept a plaintext client: %v", err)
	}
	select {
	case c := <-ln.Conns():
		c.Close()
		t.Fatal("a plaintext connection reached the game")
	case <-time.After(200 * time.Millisecond):
	}
}
