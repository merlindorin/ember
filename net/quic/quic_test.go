package quic_test

import (
	"context"
	stdnet "net"
	"testing"
	"time"

	"github.com/merlindorin/ember/net"
	"github.com/merlindorin/ember/net/nettest"
	"github.com/merlindorin/ember/net/quic"
)

func listen(t *testing.T) (*quic.Listener, string) {
	t.Helper()
	tlsConf, pin, err := quic.SelfSigned("127.0.0.1")
	if err != nil {
		t.Fatal(err)
	}
	l, err := quic.Listen("127.0.0.1:0", tlsConf)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = l.Close() })
	return l, pin
}

// handshake bounds a dial: generous, because a handshake that works
// returns at once and a loaded machine can be slow to schedule one.
const handshake = 30 * time.Second

func TestQUICTransport(t *testing.T) {
	nettest.Run(t, func(t *testing.T) (stdnet.Conn, stdnet.Conn) {
		l, pin := listen(t)
		ctx, cancel := context.WithTimeout(context.Background(), handshake)
		defer cancel()
		c, err := quic.Dial(ctx, l.Addr(), quic.Pinned(pin))
		if err != nil {
			t.Fatal(err)
		}
		select {
		case s := <-l.Conns():
			return s, c
		case <-ctx.Done():
			t.Fatal("no connection accepted")
		}
		return nil, nil
	})
}

func TestDialRefusesAnUnpinnedCertificate(t *testing.T) {
	l, _ := listen(t)
	_, other, err := quic.SelfSigned("127.0.0.1")
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), handshake)
	defer cancel()
	if _, err = quic.Dial(ctx, l.Addr(), quic.Pinned(other)); err == nil {
		t.Fatal("dialed a server whose certificate is not the pinned one")
	}
}

func TestQUICCarriesDatagrams(t *testing.T) {
	l, pin := listen(t)
	ctx, cancel := context.WithTimeout(context.Background(), handshake)
	defer cancel()
	c, err := quic.Dial(ctx, l.Addr(), quic.Pinned(pin))
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = c.Close() }()
	s := <-l.Conns()
	defer func() { _ = s.Close() }()
	for _, end := range []stdnet.Conn{c, s} {
		if d, ok := end.(net.DatagramConn); !ok || !d.SupportsDatagrams() {
			t.Fatal("a QUIC connection offers no datagrams")
		}
	}
	cd, _ := c.(net.DatagramConn)
	sd, _ := s.(net.DatagramConn)
	if err = cd.SendDatagram([]byte("pose")); err != nil {
		t.Fatal(err)
	}
	got, err := sd.ReceiveDatagram(ctx)
	if err != nil || string(got) != "pose" {
		t.Fatalf("received %q, %v", got, err)
	}
}
