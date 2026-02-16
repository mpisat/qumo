package relay

// Fixed WebTransport server wrapper that properly initializes the H3 field.
//
// gomoqt's webtransportgo.NewServer creates a webtransport-go Server with
// H3 = nil. In webtransport-go v0.10.0, H3 changed from value type to pointer
// type, so nil H3 causes a panic in ServeQUICConn. This file provides a
// replacement that sets H3 to a properly configured *http3.Server.

import (
	"context"
	"errors"
	"net"
	"net/http"
	"time"

	"github.com/okdaichi/gomoqt/quic"
	gomoqt_wt "github.com/okdaichi/gomoqt/webtransport"
	quicgo "github.com/quic-go/quic-go"
	"github.com/quic-go/quic-go/http3"
	webtransport "github.com/quic-go/webtransport-go"
)

// newFixedWebTransportServer creates a webtransport.Server with H3 properly
// initialized. This works around the nil H3 bug in gomoqt's NewServer.
func newFixedWebTransportServer(checkOrigin func(*http.Request) bool) gomoqt_wt.Server {
	h3Server := &http3.Server{
		Handler: http.DefaultServeMux,
	}
	webtransport.ConfigureHTTP3Server(h3Server)

	wtserver := &webtransport.Server{
		H3:          h3Server,
		CheckOrigin: checkOrigin,
	}

	return &fixedWTServer{server: wtserver}
}

// fixedWTServer implements gomoqt's webtransport.Server interface by wrapping
// the quic-go/webtransport-go Server with proper H3 configuration.
type fixedWTServer struct {
	server *webtransport.Server
}

func (w *fixedWTServer) Upgrade(rw http.ResponseWriter, r *http.Request) (quic.Connection, error) {
	sess, err := w.server.Upgrade(rw, r)
	if err != nil {
		return nil, err
	}
	return &wtSessionConn{sess: sess}, nil
}

type quicgoUnwrapper interface {
	Unwrap() *quicgo.Conn
}

func (w *fixedWTServer) ServeQUICConn(conn quic.Connection) error {
	if conn == nil {
		return nil
	}
	if u, ok := conn.(quicgoUnwrapper); ok {
		return w.server.ServeQUICConn(u.Unwrap())
	}
	return errors.New("invalid connection type: expected a wrapped quic-go connection with Unwrap() method")
}

func (w *fixedWTServer) Close() error {
	return w.server.Close()
}

func (w *fixedWTServer) Shutdown(ctx context.Context) error {
	closeCh := make(chan struct{})
	go func() {
		_ = w.server.Close()
		close(closeCh)
	}()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-closeCh:
		return nil
	}
}

// wtSessionConn wraps *webtransport.Session as quic.Connection for gomoqt.
type wtSessionConn struct {
	sess *webtransport.Session
}

func (c *wtSessionConn) AcceptStream(ctx context.Context) (quic.Stream, error) {
	s, err := c.sess.AcceptStream(ctx)
	if err != nil {
		return nil, err
	}
	return &wtStream{stream: s}, nil
}

func (c *wtSessionConn) AcceptUniStream(ctx context.Context) (quic.ReceiveStream, error) {
	s, err := c.sess.AcceptUniStream(ctx)
	if err != nil {
		return nil, err
	}
	return &wtRecvStream{stream: s}, nil
}

func (c *wtSessionConn) CloseWithError(code quic.ApplicationErrorCode, msg string) error {
	return c.sess.CloseWithError(webtransport.SessionErrorCode(code), msg)
}

func (c *wtSessionConn) ConnectionState() quic.ConnectionState {
	return c.sess.SessionState().ConnectionState
}

func (c *wtSessionConn) Context() context.Context    { return c.sess.Context() }
func (c *wtSessionConn) LocalAddr() net.Addr          { return c.sess.LocalAddr() }
func (c *wtSessionConn) RemoteAddr() net.Addr          { return c.sess.RemoteAddr() }
func (c *wtSessionConn) ReceiveDatagram(ctx context.Context) ([]byte, error) {
	return c.sess.ReceiveDatagram(ctx)
}
func (c *wtSessionConn) SendDatagram(b []byte) error { return c.sess.SendDatagram(b) }

func (c *wtSessionConn) OpenStream() (quic.Stream, error) {
	s, err := c.sess.OpenStream()
	if err != nil {
		return nil, err
	}
	return &wtStream{stream: s}, nil
}

func (c *wtSessionConn) OpenStreamSync(ctx context.Context) (quic.Stream, error) {
	s, err := c.sess.OpenStreamSync(ctx)
	if err != nil {
		return nil, err
	}
	return &wtStream{stream: s}, nil
}

func (c *wtSessionConn) OpenUniStream() (quic.SendStream, error) {
	s, err := c.sess.OpenUniStream()
	if err != nil {
		return nil, err
	}
	return &wtSendStream{stream: s}, nil
}

func (c *wtSessionConn) OpenUniStreamSync(ctx context.Context) (quic.SendStream, error) {
	s, err := c.sess.OpenUniStreamSync(ctx)
	if err != nil {
		return nil, err
	}
	return &wtSendStream{stream: s}, nil
}

// Stream wrappers bridge webtransport-go stream types to gomoqt quic types.

type wtStream struct {
	stream *webtransport.Stream
}

func (s *wtStream) Read(b []byte) (int, error)            { return s.stream.Read(b) }
func (s *wtStream) Write(b []byte) (int, error)           { return s.stream.Write(b) }
func (s *wtStream) Close() error                          { return s.stream.Close() }
func (s *wtStream) Context() context.Context              { return s.stream.Context() }
func (s *wtStream) CancelRead(c quic.StreamErrorCode)     { s.stream.CancelRead(webtransport.StreamErrorCode(c)) }
func (s *wtStream) CancelWrite(c quic.StreamErrorCode)    { s.stream.CancelWrite(webtransport.StreamErrorCode(c)) }
func (s *wtStream) SetDeadline(t time.Time) error         { return s.stream.SetDeadline(t) }
func (s *wtStream) SetReadDeadline(t time.Time) error     { return s.stream.SetReadDeadline(t) }
func (s *wtStream) SetWriteDeadline(t time.Time) error    { return s.stream.SetWriteDeadline(t) }

type wtRecvStream struct {
	stream *webtransport.ReceiveStream
}

func (s *wtRecvStream) Read(b []byte) (int, error)            { return s.stream.Read(b) }
func (s *wtRecvStream) CancelRead(c quic.StreamErrorCode)     { s.stream.CancelRead(webtransport.StreamErrorCode(c)) }
func (s *wtRecvStream) SetReadDeadline(t time.Time) error     { return s.stream.SetReadDeadline(t) }

type wtSendStream struct {
	stream *webtransport.SendStream
}

func (s *wtSendStream) Write(b []byte) (int, error)           { return s.stream.Write(b) }
func (s *wtSendStream) Close() error                          { return s.stream.Close() }
func (s *wtSendStream) Context() context.Context              { return s.stream.Context() }
func (s *wtSendStream) CancelWrite(c quic.StreamErrorCode)    { s.stream.CancelWrite(webtransport.StreamErrorCode(c)) }
func (s *wtSendStream) SetWriteDeadline(t time.Time) error    { return s.stream.SetWriteDeadline(t) }
