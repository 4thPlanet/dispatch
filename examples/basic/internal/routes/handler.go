package routes

import (
	"fmt"
	"net/http"

	"github.com/4thPlanet/dispatch"
)

type Handler struct {
	r           *http.Request
	VisitNumber uint32
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
		fmt.Fprintf(w, "Hello, World! You are visit number %d\n", r.VisitNumber)
	})
	// This is an intentionally faulty handler to demonstrate how the error handler middleware behaves.
	handler.HandleFunc("/panic", func(w http.ResponseWriter, r *Handler) {
		var a *int
		*a = *a + 10
	})
	return handler

}
