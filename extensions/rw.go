package extensions

import (
	"bufio"
	"bytes"
	"errors"
	"net"
	"net/http"
)

// CustomResponseWriter is used for custom rw behavior in HTTP handlers. It used in some middlewares.
type CustomResponseWriter struct {
	http.ResponseWriter
	StatusCode int
	Body       *bytes.Buffer
}

func NewCustomResponseWriter(w http.ResponseWriter) *CustomResponseWriter {
	return &CustomResponseWriter{
		ResponseWriter: w,
		StatusCode:     http.StatusOK,
		Body:           new(bytes.Buffer),
	}
}

func (rw *CustomResponseWriter) WriteHeader(code int) {
	rw.StatusCode = code
}

func (rw *CustomResponseWriter) Write(b []byte) (int, error) {
	return rw.Body.Write(b)
}

func (rw *CustomResponseWriter) Flush() {
	rw.ResponseWriter.WriteHeader(rw.StatusCode)

	if rw.Body.Len() > 0 {
		rw.ResponseWriter.Write(rw.Body.Bytes())
	}
}

func (rw *CustomResponseWriter) Header() http.Header {
	if rw.ResponseWriter == nil {
		return http.Header{}
	}

	return rw.ResponseWriter.Header()
}

func (rw *CustomResponseWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	hijacker, ok := rw.ResponseWriter.(http.Hijacker)
	if !ok {
		return nil, nil, errors.New("rw does not support hijacking")
	}

	return hijacker.Hijack()
}
