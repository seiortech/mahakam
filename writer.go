package mahakam

import (
	"bufio"
	"fmt"
	"net"
	"net/http"
)

type RW struct {
	conn       net.Conn
	headers    http.Header
	statusCode int
	written    bool
	hijacked   bool
	buf        *bufio.ReadWriter
}

func NewRW(conn net.Conn) *RW {
	return &RW{
		conn:       conn,
		headers:    make(http.Header),
		statusCode: http.StatusOK,
		buf:        bufio.NewReadWriter(bufio.NewReader(conn), bufio.NewWriter(conn)),
	}
}

func (w *RW) Reader() *bufio.Reader {
	return w.buf.Reader
}

func (w *RW) Writer() *bufio.Writer {
	return w.buf.Writer
}

func (w *RW) Write(data []byte) (int, error) {
	if w.hijacked {
		return 0, http.ErrHijacked
	}

	if !w.written {
		w.WriteHeader(w.statusCode)
	}

	n, err := w.buf.Write(data)
	flushErr := w.buf.Flush()
	if flushErr != nil && err == nil {
		err = flushErr
	}

	return n, err
}

func (w *RW) Header() http.Header {
	return w.headers
}

func (w *RW) WriteHeader(statusCode int) {
	if w.written {
		return
	}

	w.statusCode = statusCode
	w.written = true

	fmt.Fprintf(w.conn, "HTTP/1.1 %d %s\r\n", statusCode, http.StatusText(statusCode))

	for key, values := range w.headers {
		for _, value := range values {
			fmt.Fprintf(w.conn, "%s: %s\r\n", key, value)
		}
	}

	fmt.Fprintf(w.conn, "\r\n")
	w.buf.Flush()
}

func (w *RW) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	if w.hijacked {
		return nil, nil, http.ErrHijacked
	}

	if w.written {
		return nil, nil, http.ErrBodyNotAllowed
	}

	w.hijacked = true

	return w.conn, w.buf, nil
}
