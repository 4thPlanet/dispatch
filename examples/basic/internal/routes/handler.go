package routes

import (
	"net/http"

	"github.com/4thPlanet/dispatch"
)

type Handler struct {
	r *http.Request
}

func (h *Handler) Request() *http.Request {
	return h.r
}

func SetupRouter() *dispatch.TypedHandler[*Handler] {
	handler := dispatch.NewTypedHandler(func(r *http.Request) *Handler {
		return &Handler{
			r: r,
		}
	})
	handler.HandleFunc("/", func(w http.ResponseWriter, r *Handler) {
		w.Write([]byte("Hello, World!"))
	})
	return handler

}
