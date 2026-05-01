package dispatch

import (
	"errors"
	"log"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"sync"
	"testing"
)

type HtmlOutputer interface {
	Html(http.ResponseWriter) error
}
type CsvOutputer interface {
	Csv(http.ResponseWriter) error
}
type JsonOutputer interface {
	Json(http.ResponseWriter) error
}

var ctn = NewContentTypeNegotiator()

func init() {
	RegisterImplementationToNegotiator[HtmlOutputer](ctn, "text/html")
	RegisterImplementationToNegotiator[CsvOutputer](ctn, "text/csv")
	RegisterImplementationToNegotiator[JsonOutputer](ctn, "application/json")
}

func TestRegisterImplementation(t *testing.T) {
	// The registrations in init() above handle the standard, happy paths
	// This unit test will handle the not-so-happy
	negotiator := NewContentTypeNegotiator()
	t.Run("missing_subtype", func(t *testing.T) {
		if err := RegisterImplementationToNegotiator[HtmlOutputer](negotiator, "foo"); err != nil {
			t.Errorf("Unexpected error registering content type with * subtype: %v", err)
		}
	})
	t.Run("invalid_content_type", func(t *testing.T) {
		if got, want := RegisterImplementationToNegotiator[HtmlOutputer](negotiator, "foo/bar/baz"), InvalidContentType; got != want {
			t.Errorf("Call to RegisterImplementationToNegotiation did not result in expected error. Got: %v, Want: %v", got, want)
		}
	})
	t.Run("InvalidImplementation", func(t *testing.T) {
		type i int
		if got, want := RegisterImplementationToNegotiator[i](negotiator, "foo/bar"), InvalidImplementation; got != want {
			t.Errorf("Call to RegisterImplementationToNegotiation did not result in expected error. Got: %v, Want: %v", got, want)
		}
	})
	t.Run("InvalidInterfaceDefinition", func(t *testing.T) {
		type i interface {
			Foo()
			Bar(http.ResponseWriter) error
		}
		if got, want := RegisterImplementationToNegotiator[i](negotiator, "foo/bar"), InvalidInterfaceDefinition; got != want {
			t.Errorf("Call to RegisterImplementationToNegotiation did not result in expected error. Got: %v, Want: %v", got, want)
		}
		type j interface {
			Foo()
		}
		if got, want := RegisterImplementationToNegotiator[j](negotiator, "foo/bar"), InvalidInterfaceDefinition; got != want {
			t.Errorf("Call to RegisterImplementationToNegotiation did not result in expected error. Got: %v, Want: %v", got, want)
		}
	})

}

func TestImplementationErrors(t *testing.T) {
	var _ error = NegotiatorImplementationError(0)
	// Confirm these constants have a non-empty error string
	if got := InvalidContentType.Error(); got == "" {
		t.Errorf("InvalidContentType error message empty.")
	}
	if got := InvalidImplementation.Error(); got == "" {
		t.Errorf("InvalidImplementation error message empty.")
	}
	if got := InvalidInterfaceDefinition.Error(); got == "" {
		t.Errorf("InvalidInterfaceDefinition error message empty.")
	}
	// Confirm 0 panics
	wg := sync.WaitGroup{}
	wg.Go(func() {
		defer func() {
			if r := recover(); r != "invalid error" {
				t.Errorf("NegotiatorImplementationError 0 (invalid error value) did not panic on call to String()")
			}
		}()
		NegotiatorImplementationError(0).Error()
	})
	wg.Wait()

}

type Foo struct{}

func (f *Foo) Html(w http.ResponseWriter) error {
	w.Write([]byte(`<html></html>`))
	return nil
}
func (f *Foo) Json(w http.ResponseWriter) error {
	w.Write([]byte(`{"foo":"bar"}`))
	return nil
}

func TestNegotiateContentType(t *testing.T) {
	fooType := reflect.TypeFor[*Foo]()

	for _, test := range []struct {
		HeaderValue string
		Output      reflect.Type
		ContentType string
	}{
		{HeaderValue: "application/xhtml", Output: nil, ContentType: ""},
		{HeaderValue: "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8", Output: reflect.TypeFor[HtmlOutputer](), ContentType: "text/html"},
		{HeaderValue: "application/xhtml+xml,application/json;q=0.7,text/html;q=0.8", Output: reflect.TypeFor[HtmlOutputer](), ContentType: "text/html"},
		{HeaderValue: "application/xhtml+xml,application/json;q=0.7,text/*;q=0.8", Output: reflect.TypeFor[JsonOutputer](), ContentType: "application/json"},
		{HeaderValue: "application/json;q=0.9, text/*", Output: reflect.TypeFor[JsonOutputer](), ContentType: "application/json"},
	} {
		implements, contentType := ctn.negotiateContentType(test.HeaderValue, fooType)
		if got, want := implements, test.Output; got != want {
			t.Errorf("Unexpected implementation returned for header value [%s]. Got: %v, Want: %v", test.HeaderValue, got, want)
		}
		if got, want := contentType, test.ContentType; got != want {
			t.Errorf("Unexpected content type returned from negotiation for header value [%s]. Got: %v, Want: %v", test.HeaderValue, got, want)
		}
	}

	// special case for */* - the content type returned is not a given
	if got, _ := ctn.negotiateContentType("application/xhtml+xml,text/csv;q=0.7,*/*;q=0.1,", fooType); got != reflect.TypeFor[HtmlOutputer]() && got != reflect.TypeFor[JsonOutputer]() {
		t.Errorf("Header value [*/*] returned nil implementation. Expected either HtmlOutputer or JsonOutputer")
	}
	if got, _ := ctn.negotiateContentType("", fooType); got != reflect.TypeFor[HtmlOutputer]() && got != reflect.TypeFor[JsonOutputer]() {
		t.Errorf("Header value [] returned nil implementation. Expected either HtmlOutputer or JsonOutputer")
	}
}

type TeapotError struct{}

func (err TeapotError) Error() string  { return "I'm a little teapot, short and stout!" }
func (err TeapotError) ErrorCode() int { return http.StatusTeapot }

var fooView ContentTypeHandler[*testRequest, *Foo] = func(r *testRequest) (*Foo, error) {
	switch r.PathDepth {
	case 1:
		return &Foo{}, nil
	case 2:
		return nil, errors.New("test error")
	default:
		return nil, TeapotError{}
	}

}

func TestAsTypedHandler(t *testing.T) {
	handler := NewTypedHandler(func(r *http.Request) *testRequest {
		req := new(testRequest)
		req.r = r
		req.PathDepth = strings.Count(r.URL.Path, "/")
		return req
	})
	handler.HandleFunc("/", fooView.AsTypedHandler(ctn, log.Default().Writer()))
	foo := new(Foo)
	for _, test := range []struct {
		Path       string
		Accept     string
		Code       int
		BodyWriter func(http.ResponseWriter) error
	}{
		{Path: "/", Accept: "text/html", Code: http.StatusOK, BodyWriter: foo.Html},
		{Path: "/", Accept: "application/json", Code: http.StatusOK, BodyWriter: foo.Json},
		{Path: "/", Accept: "text/csv", Code: http.StatusNotAcceptable, BodyWriter: foo.Json},
		{Path: "/foo/bar", Accept: "text/html", Code: http.StatusInternalServerError},
		{Path: "/foo/bar/baz", Accept: "text/html", Code: http.StatusTeapot},
	} {
		t.Run(test.Path+"_"+test.Accept, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, test.Path, nil)
			req.Header.Set("Accept", test.Accept)
			res := httptest.NewRecorder()
			handler.ServeHTTP(res, req)

			if got, want := res.Code, test.Code; got != want {
				t.Errorf("Unexpected code returned, got: %v, want: %v", got, want)
			}

			if res.Code == http.StatusOK {
				if got, want := res.Header().Get("Content-Type"), test.Accept; got != want {
					t.Errorf("Unexpected code returned, got: %v, want: %v", got, want)
				}
				buf := httptest.NewRecorder()
				if err := test.BodyWriter(buf); err != nil {
					t.Fatalf("Error returned by body writer: %v", err)
				}
				if got, want := res.Body.Bytes(), buf.Body.Bytes(); !reflect.DeepEqual(got, want) {
					t.Errorf("Unexpected body returned, got: %s, want: %s", got, want)
				}

			}
		})

	}

}
