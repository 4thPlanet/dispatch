package dispatch

import (
	"cmp"
	"net/http"
	"reflect"
	"slices"
	"strconv"
	"strings"
	"sync"
)

type cache[K comparable, V any] struct {
	cache map[K]V
	mu    sync.RWMutex
}

func createCache[K comparable, V any]() *cache[K, V] {
	c := new(cache[K, V])
	c.cache = map[K]V{}
	return c
}
func (c *cache[K, V]) Load(key K) (V, bool) {
	var val V
	var isset bool
	c.mu.RLock()
	defer c.mu.RUnlock()
	val, isset = c.cache[key]
	return val, isset
}
func (c *cache[K, V]) Store(key K, val V) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.cache[key] = val
}

// type ContentTypeNegotiator map[string]map[string]reflect.Type
type ContentTypeNegotiator struct {
	table       map[string]map[string]reflect.Type
	offersCache *cache[reflect.Type, map[string]map[string]reflect.Type]
}

func NewContentTypeNegotiator() *ContentTypeNegotiator {

	return &ContentTypeNegotiator{
		table:       make(map[string]map[string]reflect.Type),
		offersCache: createCache[reflect.Type, map[string]map[string]reflect.Type](),
	}
}

var contentWriterType = reflect.TypeFor[func(http.ResponseWriter) error]()

type NegotiatorImplementationError byte

const (
	_ NegotiatorImplementationError = iota
	InvalidContentType
	InvalidImplementation
	InvalidInterfaceDefinition
)

func (e NegotiatorImplementationError) Error() string {
	switch e {
	case InvalidContentType:
		return "contentType does not match form form type/subtype"
	case InvalidImplementation:
		return "type is not an interface"
	case InvalidInterfaceDefinition:
		return "type must implement a single method with the signature func(http.ResponseWriter) error"
	default:
		panic("invalid error")
	}

}

func RegisterImplementationToNegotiator[O any](ctn *ContentTypeNegotiator, contentType string) error {
	contentTypeParts := strings.Split(contentType, "/")
	switch len(contentTypeParts) {
	case 1:
		contentTypeParts = append(contentTypeParts, "*")
	case 2:
	default:
		return InvalidContentType
	}
	t, st := contentTypeParts[0], contentTypeParts[1]

	// O must be an interface with a single method
	dataType := reflect.TypeFor[O]()
	if dataType.Kind() != reflect.Interface {
		return InvalidImplementation
	}

	if dataType.NumMethod() != 1 {
		return InvalidInterfaceDefinition
	}
	method := dataType.Method(0).Type
	if method != contentWriterType {
		return InvalidInterfaceDefinition
	}

	if _, isset := ctn.table[t]; !isset {
		ctn.table[t] = map[string]reflect.Type{}
	}
	ctn.table[t][st] = dataType
	return nil
}

func (ctn *ContentTypeNegotiator) negotiateContentType(acceptHeader string, dataType reflect.Type) (reflect.Type, string) {
	if len(acceptHeader) == 0 {
		acceptHeader = "*/*"
	}

	// offers contains a list of content types which are available to deliver to the user
	offers, ok := ctn.offersCache.Load(dataType)
	if !ok {
		offers = map[string]map[string]reflect.Type{}
		for contentType, subtypes := range ctn.table {
			offers[contentType] = map[string]reflect.Type{}
			for subtype, implements := range subtypes {
				if dataType.Implements(implements) {
					offers[contentType][subtype] = implements
				}
			}
		}
		ctn.offersCache.Store(dataType, offers)
	}

	type acceptedContentType struct {
		implementation reflect.Type
		contentType    string
		specificity    byte
		weight         float64
	}
	acceptHeaderSplit := strings.Split(acceptHeader, ",")
	acceptedContentTypes := make([]acceptedContentType, 0, len(acceptHeaderSplit))
	for _, contentType := range acceptHeaderSplit {
		qualitySplit := strings.Split(strings.TrimSpace(contentType), ";q=")
		contentTypeParts := strings.Split(qualitySplit[0], "/")
		switch len(contentTypeParts) {
		case 1:
			contentTypeParts = append(contentTypeParts, "*")
		case 2:
		default:
			continue // Invalid content type
		}
		t, st := contentTypeParts[0], contentTypeParts[1]
		var specificity byte
		if t == "*" {
			specificity = 0
		} else if st == "*" {
			specificity = 1
		} else {
			specificity = 2
		}
		found := false
		var acceptedType = acceptedContentType{specificity: specificity, contentType: t + "/" + st}

		if subtypes, isset := offers[t]; isset {
			if implements, isset := offers[t][st]; isset {
				acceptedType.implementation = implements
				found = true
			} else if st == "*" {
				for _, implements := range subtypes {
					acceptedType.implementation = implements
					break
				}
				found = true
			}
		} else if t == "*" {
		WildcardMatchLoop:
			for _, subtypes := range offers {
				for _, implements := range subtypes {
					acceptedType.implementation = implements
					break WildcardMatchLoop
				}
			}
			found = true
		}

		if !found {
			continue
		}
		acceptedType.weight = 1.0

		if len(qualitySplit) > 1 {
			if weight, err := strconv.ParseFloat(qualitySplit[1], 64); err != nil {
				continue // Invalid weight
			} else {
				acceptedType.weight = weight
			}
		}
		if acceptedType.weight == 1 && acceptedType.specificity == 2 {
			// short-circuit the result, this is as good as you'll get
			return acceptedType.implementation, acceptedType.contentType
		} else {
			acceptedContentTypes = append(acceptedContentTypes, acceptedType)
		}
	}

	if len(acceptedContentTypes) == 0 {
		return nil, ""
	}

	// get max based off of weight
	highestWeighted := slices.MaxFunc(acceptedContentTypes, func(a, b acceptedContentType) int {
		if s := cmp.Compare(a.specificity, b.specificity); s != 0 {
			return s
		}
		return cmp.Compare(a.weight, b.weight)
	})
	return highestWeighted.implementation, highestWeighted.contentType
}

type ContentTypeHandler[R RequestAdapter, O any] func(r R) (O, error)
type HttpError interface {
	error
	ErrorCode() int
}

func (fn ContentTypeHandler[R, O]) AsTypedHandler(ctn *ContentTypeNegotiator, logger Printer) typedHandler[R] {
	return func(w http.ResponseWriter, r R) {
		implementation, contentType := ctn.negotiateContentType(r.Request().Header.Get("Accept"), reflect.TypeFor[O]())
		if implementation == nil {
			// You could include a list of available content types, either in the response body or its headers
			w.WriteHeader(http.StatusNotAcceptable)
			return
		}

		// Assuming it succeeds:
		out, err := fn(r)
		if err != nil {
			if httpError, ok := err.(HttpError); ok {
				w.WriteHeader(httpError.ErrorCode())
			} else {
				w.WriteHeader(http.StatusInternalServerError)
			}
			return
		}
		w.Header().Set("Content-Type", contentType)

		transformer := reflect.ValueOf(out).Convert(implementation).Method(0).Interface().(func(http.ResponseWriter) error)
		if err := transformer(w); err != nil {
			logger.Print(err)
			w.WriteHeader(http.StatusInternalServerError)
		}

	}
}
