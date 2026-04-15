package dispatch

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

type testRequest struct {
	r         *http.Request
	PathDepth int
}

func (r *testRequest) Request() *http.Request {
	return r.r
}

func TestTypedHandler(t *testing.T) {

	var req *http.Request

	handler := NewTypedHandler(func(r *http.Request) *testRequest {
		req := new(testRequest)
		req.r = r
		req.PathDepth = strings.Count(r.URL.Path, "/")
		return req
	})
	handler.HandleFunc("/", func(w http.ResponseWriter, r *testRequest) {
		if got, want := r.Request(), req; got != want {
			t.Errorf("Unexpected request. Got: %v, Want: %v", got, want)
		}
		// count number of / in url path
		if got, want := strings.Count(r.Request().URL.Path, "/"), r.PathDepth; got != want {
			t.Errorf("Unexpected Path Depth. Got: %v, Want: %v", got, want)
		}
	})

	mux := http.NewServeMux()
	mux.Handle("/", handler)

	req = httptest.NewRequest(http.MethodGet, "/foo/bar/baz", nil)
	res := httptest.NewRecorder()
	mux.ServeHTTP(res, req)

}
