package mahakam

import (
	"context"
	"fmt"
	"net"
	"net/http"

	"github.com/cloudwego/netpoll"
)

type netpoolFramework struct {
	Address      string
	mux          *http.ServeMux
	middleware   []Middleware
	ErrorHandler func(http.ResponseWriter, *http.Request, error)
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
