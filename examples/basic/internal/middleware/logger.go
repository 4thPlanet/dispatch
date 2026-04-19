package middleware

import (
	"bufio"
	"errors"
	"io"
	"log"
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/4thPlanet/dispatch"
	"github.com/4thPlanet/dispatch/examples/basic/internal/routes"
)

// Custom writer to get size of output
type writerLog struct {
	http.ResponseWriter
	length int
	code   int
}

func (wl *writerLog) reset(w http.ResponseWriter) {
	wl.ResponseWriter = w
	wl.length = 0
	wl.code = http.StatusOK
}

func (w *writerLog) Write(out []byte) (int, error) {
	n, err := w.ResponseWriter.Write(out)
	w.length += n
	return n, err
}
func (w *writerLog) WriteHeader(code int) {
	w.code = code
	w.ResponseWriter.WriteHeader(code)
}
func (w *writerLog) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	h, ok := w.ResponseWriter.(http.Hijacker)
	if !ok {
		return nil, nil, errors.New("hijack not supported")
	}
	return h.Hijack()
}
func (w *writerLog) Flush() {
	if f, ok := w.ResponseWriter.(http.Flusher); ok {
		f.Flush()
	}
}

// Custom Reader on request body to get size of input
type bodySizeReader struct {
	size int
	io.ReadCloser
}

func newBodySizeReader(rc io.ReadCloser) *bodySizeReader {
	bsr := new(bodySizeReader)
	bsr.ReadCloser = rc
	return bsr
}
func (bsr *bodySizeReader) Read(buf []byte) (int, error) {
	n, err := bsr.ReadCloser.Read(buf)
	bsr.size += n
	return n, err
}

func (bsr *bodySizeReader) Close() error {
	return nil
}

func Logger() dispatch.Middleware[*routes.Handler] {
	wlPool := sync.Pool{
		New: func() any {
			return new(writerLog)
		},
	}

	return func(w http.ResponseWriter, r *routes.Handler, next dispatch.Middleware[*routes.Handler]) {
		start := time.Now()
		wl := wlPool.Get().(*writerLog)
		defer wlPool.Put(wl)
		wl.reset(w)

		bsr := newBodySizeReader(r.Request().Body)
		defer bsr.ReadCloser.Close()
		r.Request().Body = bsr

		next(wl, r, next)
		io.ReadAll(bsr)
		log.Printf("%s %s %s in: %d out: %d | %d",
			r.Request().Method,
			r.Request().URL.Path,
			time.Since(start),
			bsr.size,
			wl.length,
			wl.code,
		)
	}
}
