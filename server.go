package mahakam

import (
	"fmt"
	"net/http"

	"github.com/seiortech/mahakam/extensions"
)

// Middleware defines a middleware parameter type for the server.
type Middleware = func(http.HandlerFunc) http.HandlerFunc

// Server is a custom HTTP server that uses netpoll for handling connections.
type Server struct {
	Address      string
	mux          *http.ServeMux
	server       NetworkFramework
	middleware   []Middleware
	ErrorHandler func(http.ResponseWriter, *http.Request, error)
}

// NewServer creates a new Server instance with the specified address and HTTP ServeMux.
func NewServer(address string, mux *http.ServeMux) *Server {
	return &Server{
		Address: address,
		mux:     mux,
		server:  NETPOLL,
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
	switch s.server {
	case NETPOLL:
		s := netpoolFramework{
			Address:      s.Address,
			mux:          s.mux,
			middleware:   s.middleware,
			ErrorHandler: s.ErrorHandler,
		}

		return s.listenAndServe()
	case HTTP:
		s := httpFramework{
			Address:      s.Address,
			mux:          s.mux,
			middleware:   s.middleware,
			ErrorHandler: s.ErrorHandler,
		}

		return s.listenAndServe()
	case NET:
		s := netFramework{
			Address:      s.Address,
			mux:          s.mux,
			middleware:   s.middleware,
			ErrorHandler: s.ErrorHandler,
		}

		return s.listenAndServe()
	default:
		return fmt.Errorf("unsupported server framework: %s", s.server)
	}
}

// Use binds middleware functions to the server.
func (s *Server) Use(middleware ...func(http.HandlerFunc) http.HandlerFunc) {
	if s.middleware == nil {
		s.middleware = []func(http.HandlerFunc) http.HandlerFunc{}
	}

	s.middleware = append(s.middleware, middleware...)
}

// ServeFiles serves static files from the specified root directory using the given pattern.
func (s *Server) ServeFiles(pattern string, root http.FileSystem) {
	if s.mux == nil {
		s.mux = http.NewServeMux()
	}

	s.mux.Handle(pattern, http.StripPrefix(pattern, http.FileServer(root)))
}

// Handle binds a handler to a specific pattern in the server's HTTP ServeMux.
func (s *Server) Handle(pattern string, handler http.Handler) {
	if s.mux == nil {
		s.mux = http.NewServeMux()
	}

	s.mux.Handle(pattern, handler)
}

// HandleFunc binds a handler function to a specific pattern in the server's HTTP ServeMux.
func (s *Server) HandleFunc(pattern string, handler http.HandlerFunc) {
	if s.mux == nil {
		s.mux = http.NewServeMux()
	}

	s.mux.HandleFunc(pattern, handler)
}

// Framework sets the network framework for the server. by default it uses NETPOLL.
func (s *Server) Framework(framework NetworkFramework) {
	s.server = framework
}
