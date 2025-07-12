package mahakam

import (
	"fmt"
	"net/http"
)

type httpFramework struct {
	Address         string
	mux             *http.ServeMux
	middleware      []func(http.HandlerFunc) http.HandlerFunc
	ErrorHandler    func(http.ResponseWriter, *http.Request, error)
	certificatePath string
	keyPath         string
}

func (s *httpFramework) listenAndServe() error {
	if s.mux == nil {
		s.mux = http.NewServeMux()
	}

	var handler http.Handler = s.mux
	if len(s.middleware) > 0 {
		handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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

			h := func(w http.ResponseWriter, r *http.Request) {
				s.mux.ServeHTTP(w, r)
			}

			for i := len(s.middleware) - 1; i >= 0; i-- {
				h = s.middleware[i](h)
			}

			h(w, r)
		})
	}

	return http.ListenAndServe(s.Address, handler)
}

func (s *httpFramework) listenAndServeTLS() error {
	if s.mux == nil {
		s.mux = http.NewServeMux()
	}

	var handler http.Handler = s.mux
	if len(s.middleware) > 0 {
		handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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

			h := func(w http.ResponseWriter, r *http.Request) {
				s.mux.ServeHTTP(w, r)
			}

			for i := len(s.middleware) - 1; i >= 0; i-- {
				h = s.middleware[i](h)
			}

			h(w, r)
		})
	}

	return http.ListenAndServeTLS(s.Address, s.certificatePath, s.keyPath, handler)
}
