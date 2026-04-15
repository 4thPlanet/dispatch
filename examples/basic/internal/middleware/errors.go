package middleware

import (
	"log"
	"net/http"

	"github.com/4thPlanet/dispatch"
	"github.com/4thPlanet/dispatch/examples/basic/internal/routes"
)

func Errors() dispatch.Middleware[*routes.Handler] {
	return func(w http.ResponseWriter, r *routes.Handler, next dispatch.Middleware[*routes.Handler]) {
		// count visits to the site
		defer func() {
			if r := recover(); r != nil {
				log.Printf("A panic occurred while processing the request! %v", r)
				w.WriteHeader(http.StatusInternalServerError)
			}
		}()
		next(w, r, next)
	}
}
