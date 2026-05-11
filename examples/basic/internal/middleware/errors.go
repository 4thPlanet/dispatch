package middleware

import (
	"log"
	"net/http"

	"github.com/4thPlanet/dispatch"
	"github.com/4thPlanet/dispatch/examples/basic/internal/routes"
)

type ErrorMW struct{}

func (mw *ErrorMW) Enter(w http.ResponseWriter, r *routes.Handler) (http.ResponseWriter, *routes.Handler, bool) {
	return w, r, true
}
func (mw *ErrorMW) Exit(w http.ResponseWriter, r *routes.Handler) {
	if r := recover(); r != nil {
		log.Printf("A panic occurred while processing the request! %v", r)
		w.WriteHeader(http.StatusInternalServerError)
	}
}

func Errors() dispatch.Middleware[*routes.Handler] {
	return new(ErrorMW)
}
