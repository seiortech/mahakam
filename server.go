package mahakam

import (
	"context"
	"fmt"
	"net"
	"net/http"

	"github.com/cloudwego/netpoll"
	"github.com/radenrishwan/mahakam/extensions"
)

type Middleware = func(http.HandlerFunc) http.HandlerFunc

// Server is a custom HTTP server that uses netpoll for handling connections.
type Server struct {
	Address      string
	mux          *http.ServeMux
	middleware   []Middleware
	ErrorHandler func(http.ResponseWriter, *http.Request, error)
}

// NewServer creates a new Server instance with the specified address and HTTP ServeMux.
func NewServer(address string, mux *http.ServeMux) *Server {
	return &Server{
		Address: address,
		mux:     mux,
		ErrorHandler: func(w http.ResponseWriter, r *http.Request, err error) {
			validationErr, ok := err.(extensions.ValidationError)
			if ok {
				resp, err := validationErr.JSON()
				if err != nil {
					// TODO: edit the message error later to avoid exposing internal errors
					http.Error(w, fmt.Sprintf("Failed to marshal validation error: %v", err), http.StatusInternalServerError)
					return
				}

				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(validationErr.Code)
				w.Write(resp)
				return
			}

			w.WriteHeader(http.StatusInternalServerError)
			w.Write(fmt.Appendf(nil, "%v", err))
		},
	}
}

// ListenAndServe starts the server and listens for incoming connections.
func (s *Server) ListenAndServe() error {
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

func (s *Server) handleConnection(conn net.Conn) error {
	defer conn.Close()

	w := NewRW(conn)

	r, err := http.ReadRequest(w.buf.Reader)
	if err != nil {
		return err
	}

	defer func() {
		rc := recover()
		if rc != nil {
			s.ErrorHandler(w, r, rc.(error))
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

func (s *Server) onRequest(ctx context.Context, conn netpoll.Connection) error {
	if err := s.handleConnection(conn); err != nil {
		s.ErrorHandler(nil, nil, err)
	}

	return nil
}

// Use binds middleware functions to the server.
func (s *Server) Use(middleware ...func(http.HandlerFunc) http.HandlerFunc) {
	if s.middleware == nil {
		s.middleware = []func(http.HandlerFunc) http.HandlerFunc{}
	}

	s.middleware = append(s.middleware, middleware...)
}

func (s *Server) ServeFiles(pattern string, root http.FileSystem) {
	if s.mux == nil {
		s.mux = http.NewServeMux()
	}

	s.mux.Handle(pattern, http.StripPrefix(pattern, http.FileServer(root)))
}
