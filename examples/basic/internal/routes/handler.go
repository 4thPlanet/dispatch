package routes

import (
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

type TextOutputer interface {
	Text(http.ResponseWriter) error
}
type JsonOutputer interface {
	Json(http.ResponseWriter) error
}
type HtmlOutputer interface {
	Html(http.ResponseWriter) error
}

func must(err error) {
	if err != nil {
		panic("Unexpected error: " + err.Error())
	}
}
func SetupRouter() *dispatch.TypedHandler[*Handler] {
	handler := dispatch.NewTypedHandler(func(r *http.Request) *Handler {
		return &Handler{
			r: r,
		}
	})
	var ctn = dispatch.NewContentTypeNegotiator()
	must(dispatch.RegisterImplementationToNegotiator[TextOutputer](ctn, "text/plain"))
	must(dispatch.RegisterImplementationToNegotiator[HtmlOutputer](ctn, "text/html"))
	must(dispatch.RegisterImplementationToNegotiator[JsonOutputer](ctn, "application/json"))
	greetingHandler(handler, ctn)
	chatHandler(handler)

	// This is an intentionally faulty handler to demonstrate how the error handler middleware behaves.
	handler.HandleFunc("/panic", func(w http.ResponseWriter, r *Handler) {
		var a *int
		*a = *a + 10
	})

	return handler

}
