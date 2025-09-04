package mahakam

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"net/http"

	"github.com/cloudwego/netpoll"
)

type netpoolFramework struct {
	Address         string
	mux             *http.ServeMux
	middleware      []Middleware
	ErrorHandler    func(http.ResponseWriter, *http.Request, error)
	certificatePath string
	keyPath         string
}

func (s *netpoolFramework) listenAndServe() error {
	listener, err := netpoll.CreateListener("tcp", s.Address)
	if err != nil {
		return err
	}

	defer listener.Close()

	eventLoop, err := netpoll.NewEventLoop(s.onRequest)

	if err != nil {
		return err
	}

	if err := eventLoop.Serve(listener); err != nil {
		return err
	}

	return nil
}

func (s *netpoolFramework) listenAndServeTLS() error {
	// Load the certificate and key
	cert, err := tls.LoadX509KeyPair(s.certificatePath, s.keyPath)
	if err != nil {
		return fmt.Errorf("failed to load certificate: %w", err)
	}

	// Create TLS config
	tlsConfig := &tls.Config{
		Certificates: []tls.Certificate{cert},
	}

	// Create a standard TCP listener first
	tcpListener, err := net.Listen("tcp", s.Address)
	if err != nil {
		return err
	}

	// Wrap it with TLS
	tlsListener := tls.NewListener(tcpListener, tlsConfig)
	defer tlsListener.Close()

	// Create netpoll listener from the TLS listener
	// Note: We need to handle TLS connections manually since netpoll doesn't directly support TLS wrapping
	for {
		conn, err := tlsListener.Accept()
		if err != nil {
			return err
		}

		go func(conn net.Conn) {
			if err := s.handleConnection(conn); err != nil {
				if s.ErrorHandler != nil {
					s.ErrorHandler(nil, nil, err)
				}
			}
		}(conn)
	}
}

func (s *netpoolFramework) handleConnection(conn net.Conn) error {
	defer conn.Close()

	w := NewRW(conn)

	r, err := http.ReadRequest(w.buf.Reader)
	if err != nil {
		return err
	}

	defer func() {
		if recovered := recover(); recovered != nil {
			if s.ErrorHandler != nil {
				var err error
				switch v := recovered.(type) {
				case error:
					err = v
				case string:
					err = fmt.Errorf("%s", v)
				default:
					err = fmt.Errorf("%v", v)
				}
				s.ErrorHandler(w, r, err)
			} else {
				panic(recovered)
			}
		}
	}()

	handler := s.mux.ServeHTTP
	for i := len(s.middleware) - 1; i >= 0; i-- {
		handler = s.middleware[i](handler)
	}

	handler(w, r)

	if err := w.buf.Flush(); err != nil {
		return err
	}

	return nil
}

func (s *netpoolFramework) onRequest(ctx context.Context, conn netpoll.Connection) error {
	if err := s.handleConnection(conn); err != nil {
		if s.ErrorHandler != nil {
			s.ErrorHandler(nil, nil, err)
		}
		return err
	}
	return nil
}
