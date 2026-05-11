package middleware

import (
	"net/http"
	"sync/atomic"

	"github.com/4thPlanet/dispatch/examples/basic/internal/routes"
)

type CounterMW struct {
	visits atomic.Uint32
}

func (mw *CounterMW) Enter(w http.ResponseWriter, r *routes.Handler) (http.ResponseWriter, *routes.Handler, bool) {
	r.VisitNumber = mw.visits.Add(1)
	return w, r, true
}
func (mw *CounterMW) Exit(w http.ResponseWriter, r *routes.Handler) {}

func Counter() *CounterMW {
	return new(CounterMW)
}
