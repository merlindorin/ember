// Package quic carries net connections over QUIC: TLS 1.3 built in, faster
// handshakes than TCP+TLS, and connections that survive a client changing
// networks. Each game connection is one reliable bidirectional stream, so
// it satisfies the same contract as TCP (see package nettest). Datagrams
// are enabled, so snapshots' poses travel unreliably (see net.Poses).
//
//	tlsConf, pin, _ := quic.SelfSigned("localhost")
//	l, _ := quic.Listen(":7777", tlsConf)
//	server.AddPlugins(net.NewServer(net.WithListener(l)))
//
//	conn, _ := quic.Dial(ctx, "localhost:7777", quic.Pinned(pin))
package quic

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/hex"
	"errors"
	"fmt"
	"math/big"
	stdnet "net"
	"time"

	quicgo "github.com/quic-go/quic-go"
)

// ALPN is the application protocol negotiated when the TLS config names
// none.
const ALPN = "ember"

// preface opens every stream: QUIC only announces a stream to the peer
// once data flows on it.
const preface = byte(0xE1)

func config() *quicgo.Config {
	return &quicgo.Config{MaxIdleTimeout: 15 * time.Second, KeepAlivePeriod: 5 * time.Second, EnableDatagrams: true}
}

func withALPN(c *tls.Config) *tls.Config {
	c = c.Clone()
	if len(c.NextProtos) == 0 {
		c.NextProtos = []string{ALPN}
	}
	return c
}

// Listener accepts QUIC connections and implements net.Listener.
type Listener struct {
	ln     *quicgo.Listener
	conns  chan stdnet.Conn
	cancel context.CancelFunc
}

// Listen accepts QUIC connections on a UDP address.
func Listen(addr string, tlsConf *tls.Config) (*Listener, error) {
	ln, err := quicgo.ListenAddr(addr, withALPN(tlsConf), config())
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithCancel(context.Background())
	l := &Listener{ln: ln, conns: make(chan stdnet.Conn, 16), cancel: cancel}
	go l.accept(ctx)
	return l, nil
}

func (l *Listener) accept(ctx context.Context) {
	defer close(l.conns)
	for {
		c, err := l.ln.Accept(ctx)
		if err != nil {
			return
		}
		go l.handshake(ctx, c)
	}
}

func (l *Listener) handshake(ctx context.Context, c *quicgo.Conn) {
	conn, err := accept(ctx, c)
	if err != nil {
		_ = c.CloseWithError(1, err.Error())
		return
	}
	select {
	case l.conns <- conn:
	case <-ctx.Done():
		conn.Close()
	}
}

func accept(ctx context.Context, c *quicgo.Conn) (stdnet.Conn, error) {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	s, err := c.AcceptStream(ctx)
	if err != nil {
		return nil, err
	}
	var b [1]byte
	_ = s.SetReadDeadline(time.Now().Add(10 * time.Second))
	if _, err = s.Read(b[:]); err != nil {
		return nil, err
	}
	if b[0] != preface {
		return nil, errors.New("quic: bad stream preface")
	}
	_ = s.SetReadDeadline(time.Time{})
	return &conn{Stream: s, c: c}, nil
}

func (l *Listener) Conns() <-chan stdnet.Conn { return l.conns }
func (l *Listener) Addr() string              { return l.ln.Addr().String() }

func (l *Listener) Close() error {
	l.cancel()
	return l.ln.Close()
}

// Dial opens a QUIC connection and its game stream.
func Dial(ctx context.Context, addr string, tlsConf *tls.Config) (stdnet.Conn, error) {
	c, err := quicgo.DialAddr(ctx, addr, withALPN(tlsConf), config())
	if err != nil {
		return nil, err
	}
	s, err := c.OpenStreamSync(ctx)
	if err == nil {
		_, err = s.Write([]byte{preface})
	}
	if err != nil {
		_ = c.CloseWithError(1, err.Error())
		return nil, err
	}
	return &conn{Stream: s, c: c}, nil
}

// conn is one QUIC stream seen as a net.Conn. Closing it closes the whole
// QUIC connection, so both sides notice at once.
type conn struct {
	*quicgo.Stream
	c *quicgo.Conn
}

func (c *conn) LocalAddr() stdnet.Addr { return c.c.LocalAddr() }

// SupportsDatagrams reports whether both ends negotiated datagrams.
func (c *conn) SupportsDatagrams() bool {
	st := c.c.ConnectionState().SupportsDatagrams
	return st.Local && st.Remote
}

func (c *conn) SendDatagram(b []byte) error { return c.c.SendDatagram(b) }

func (c *conn) ReceiveDatagram(ctx context.Context) ([]byte, error) { return c.c.ReceiveDatagram(ctx) }
func (c *conn) RemoteAddr() stdnet.Addr                             { return c.c.RemoteAddr() }

func (c *conn) Close() error {
	c.CancelRead(0)
	_ = c.Stream.Close()
	return c.c.CloseWithError(0, "")
}

// SelfSigned makes a TLS config with a fresh self-signed certificate for
// hosts, for development and tests, and returns the SHA-256 fingerprint
// clients pin with Pinned.
func SelfSigned(hosts ...string) (*tls.Config, string, error) {
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, "", err
	}
	serial, err := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 62))
	if err != nil {
		return nil, "", err
	}
	tmpl := &x509.Certificate{
		SerialNumber: serial,
		Subject:      pkix.Name{CommonName: "ember dev"},
		NotBefore:    time.Now().Add(-time.Hour),
		NotAfter:     time.Now().Add(30 * 24 * time.Hour),
		KeyUsage:     x509.KeyUsageDigitalSignature,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
	}
	for _, h := range hosts {
		if ip := stdnet.ParseIP(h); ip != nil {
			tmpl.IPAddresses = append(tmpl.IPAddresses, ip)
		} else {
			tmpl.DNSNames = append(tmpl.DNSNames, h)
		}
	}
	der, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &key.PublicKey, key)
	if err != nil {
		return nil, "", err
	}
	sum := sha256.Sum256(der)
	conf := &tls.Config{Certificates: []tls.Certificate{{Certificate: [][]byte{der}, PrivateKey: key}}}
	return conf, hex.EncodeToString(sum[:]), nil
}

// Pinned trusts exactly the certificate whose SHA-256 fingerprint is pin,
// such as a SelfSigned development server's.
func Pinned(pin string) *tls.Config {
	return &tls.Config{
		//nolint:gosec // pinning replaces chain verification: VerifyPeerCertificate checks the fingerprint
		InsecureSkipVerify: true,
		//nolint:gosec // no ClientSessionCache, so no session is resumed past this check
		VerifyPeerCertificate: func(raw [][]byte, _ [][]*x509.Certificate) error {
			if len(raw) == 0 {
				return errors.New("quic: no server certificate")
			}
			sum := sha256.Sum256(raw[0])
			if got := hex.EncodeToString(sum[:]); got != pin {
				return fmt.Errorf("quic: server certificate %s does not match the pinned %s", got, pin)
			}
			return nil
		},
	}
}
