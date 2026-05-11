package dispatch

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

type teapotChecker struct{}

func (mw *teapotChecker) Enter(w http.ResponseWriter, r *testRequest) (http.ResponseWriter, *testRequest, bool) {
	// if context.teapot is set, return a teapot
	if teapot := r.Request().Context().Value("teapot"); teapot != nil {
		w.WriteHeader(http.StatusTeapot)
		return nil, nil, false
	}
	return w, r, true
}
func (mw *teapotChecker) Exit(w http.ResponseWriter, r *testRequest) {}

type methodChecker struct{}

func (mw *methodChecker) Enter(w http.ResponseWriter, r *testRequest) (http.ResponseWriter, *testRequest, bool) {
	// realistically this would be done by specifying the method in the handler..
	if r.Request().Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return nil, nil, false
	}
	return w, r, true
}
func (mw *methodChecker) Exit(w http.ResponseWriter, r *testRequest) {}

func TestUseMiddleware(t *testing.T) {
	var mw1 Middleware[*testRequest] = new(teapotChecker)
	var mw2 Middleware[*testRequest] = new(methodChecker)

	handler := NewTypedHandler(func(r *http.Request) *testRequest {
		req := new(testRequest)
		req.r = r
		req.PathDepth = strings.Count(r.URL.Path, "/")
		return req
	})
	handler.HandleFunc("/", func(w http.ResponseWriter, r *testRequest) {
		w.WriteHeader(http.StatusOK)
	})
	handler.UseMiddleware(mw1, mw2)

	for _, test := range []struct {
		Teapot *bool
		Method string
		Code   int
	}{
		{Teapot: nil, Method: http.MethodGet, Code: http.StatusOK},
		{Teapot: new(true), Method: http.MethodGet, Code: http.StatusTeapot},
		{Teapot: new(true), Method: http.MethodPost, Code: http.StatusTeapot},
		{Teapot: nil, Method: http.MethodPost, Code: http.StatusMethodNotAllowed},
	} {
		req := httptest.NewRequest(test.Method, "/", nil)
		if test.Teapot != nil {
			req = req.WithContext(context.WithValue(req.Context(), "teapot", *test.Teapot))
		}
		res := httptest.NewRecorder()
		handler.ServeHTTP(res, req)
		if got, want := res.Code, test.Code; got != want {
			t.Errorf("Unexpected response code. Got: %v, Want: %v", got, want)
		}
	}

}
