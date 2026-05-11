package dispatch

import "net/http"

type Middleware[T RequestAdapter] interface {
	Enter(http.ResponseWriter, T) (http.ResponseWriter, T, bool)
	Exit(http.ResponseWriter, T)
}

func (mux *TypedHandler[T]) UseMiddleware(mws ...Middleware[T]) {
	mux.mws = mws
}
