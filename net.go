package mahakam

import (
	"net/http"
)

// netFramework is a structure that holds the address, HTTP multiplexer, middleware, and error handler for a web server.
type netFramework struct {
	Address      string
	mux          *http.ServeMux
	middleware   []Middleware
	ErrorHandler func(http.ResponseWriter, *http.Request, error)
}
