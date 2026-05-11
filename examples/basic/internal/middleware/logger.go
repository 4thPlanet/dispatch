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

var wlPool sync.Pool

type loggerData struct {
	start time.Time
	wl    *writerLog
	bsr   *bodySizeReader
}
type LoggerMW struct {
	data map[*routes.Handler]loggerData
	mu   sync.Mutex
}

func (mw *LoggerMW) Enter(w http.ResponseWriter, r *routes.Handler) (http.ResponseWriter, *routes.Handler, bool) {
	var data loggerData

	data.start = time.Now()
	data.wl = wlPool.Get().(*writerLog)
	data.wl.reset(w)

	data.bsr = newBodySizeReader(r.Request().Body)

	r.Request().Body = data.bsr
	mw.mu.Lock()
	defer mw.mu.Unlock()
	mw.data[r] = data
	return data.wl, r, true
}
func (mw *LoggerMW) Exit(w http.ResponseWriter, r *routes.Handler) {
	mw.mu.Lock()
	data := mw.data[r]
	delete(mw.data, r)
	mw.mu.Unlock()

	defer wlPool.Put(data.wl)
	defer data.bsr.ReadCloser.Close()
	if _, err := io.ReadAll(data.bsr); err != nil {
		log.Printf("Error reading remainder of request body: %v", err)
	}

	log.Printf("%s %s %s in: %d out: %d | %d",
		r.Request().Method,
		r.Request().URL.Path,
		time.Since(data.start),
		data.bsr.size,
		data.wl.length,
		data.wl.code,
	)
}

func Logger() dispatch.Middleware[*routes.Handler] {
	wlPool = sync.Pool{
		New: func() any {
			return new(writerLog)
		},
	}
	return &LoggerMW{
		data: map[*routes.Handler]loggerData{},
	}
}
