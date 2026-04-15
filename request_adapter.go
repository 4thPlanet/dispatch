package dispatch

import (
	"net/http"
)

type RequestAdapter interface {
	Request() *http.Request
}
type typedHandler[T RequestAdapter] func(http.ResponseWriter, T)
type TypedHandler[T RequestAdapter] struct {
	*http.ServeMux
	loader            func(*http.Request) T
	chainedMiddleware func(http.ResponseWriter, T, typedHandler[T])
}

func NewTypedHandler[T RequestAdapter](fn func(*http.Request) T) *TypedHandler[T] {
	return &TypedHandler[T]{
		ServeMux: http.NewServeMux(),
		loader:   fn,
	}
}

type handlerBridge[T RequestAdapter] struct {
	fn           http.HandlerFunc
	typedHandler typedHandler[T]
}

func (h *handlerBridge[T]) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	h.fn(w, r)
}

func (mux *TypedHandler[T]) HandleFunc(route string, handler typedHandler[T]) {
	h := &handlerBridge[T]{
		typedHandler: handler,
	}
	h.fn = func(w http.ResponseWriter, r *http.Request) {
		if mux.chainedMiddleware != nil {
			mux.chainedMiddleware(w, mux.loader(r), h.typedHandler)
		} else {
			h.typedHandler(w, mux.loader(r))
		}
	}
	mux.Handle(route, h)
}
