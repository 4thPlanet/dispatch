package middleware

import (
	"net/http"
	"sync/atomic"

	"github.com/4thPlanet/dispatch"
	"github.com/4thPlanet/dispatch/examples/basic/internal/routes"
)

var siteVisits atomic.Uint32

func Counter() dispatch.Middleware[*routes.Handler] {
	return func(w http.ResponseWriter, r *routes.Handler, next dispatch.Middleware[*routes.Handler]) {
		// count visits to the site
		r.VisitNumber = siteVisits.Add(1)
		next(w, r, next)
	}
}
