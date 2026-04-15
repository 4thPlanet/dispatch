package dispatch

import "net/http"

type Middleware[T RequestAdapter] func(w http.ResponseWriter, r T, next Middleware[T])

func (mux *TypedHandler[T]) UseMiddleware(mws ...Middleware[T]) {

	count := len(mws)
	if count == 0 {
		return
	}
	lastMwIdx := count - 1

	chain := func(w http.ResponseWriter, r T, finalHandler typedHandler[T]) {
		mws[lastMwIdx](w, r, func(w http.ResponseWriter, r T, _ Middleware[T]) {
			finalHandler(w, r)
		})
	}

	for mdx := lastMwIdx - 1; mdx >= 0; mdx-- {
		prevLink := chain
		chain = func(w http.ResponseWriter, r T, finalHandler typedHandler[T]) {
			mws[mdx](w, r, func(w http.ResponseWriter, r T, _ Middleware[T]) {
				prevLink(w, r, finalHandler)
			})
		}
	}
	mux.chainedMiddleware = chain
}
