package mahakam

import (
	"fmt"
	"net"
	"net/http"
)

type netFramework struct {
	Address      string
	mux          *http.ServeMux
	middleware   []Middleware
	ErrorHandler func(http.ResponseWriter, *http.Request, error)
}

func (s *netFramework) listenAndServe() error {
	listener, err := net.Listen("tcp", s.Address)
	if err != nil {
		return err
	}
	defer listener.Close()

	for {
		conn, err := listener.Accept()
		if err != nil {
			return err
		}

		go func() {
			if err := s.handleConnection(conn); err != nil {
				if s.ErrorHandler != nil {
					s.ErrorHandler(nil, nil, err)
				}
			}
		}()
	}
}

func (s *netFramework) handleConnection(conn net.Conn) error {
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
