package routes

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/4thPlanet/dispatch"
)

type Greeting struct {
	Name        string `json:"Hello"`
	VisitNumber uint32
}

func (g *Greeting) Text(w http.ResponseWriter) error {
	_, err := fmt.Fprintf(w, "Hello, %s. You are visit number %d\n", g.Name, g.VisitNumber)
	return err
}
func (g *Greeting) Json(w http.ResponseWriter) error {
	if buf, err := json.Marshal(g); err != nil {
		return err
	} else {
		_, err := w.Write(append(buf, '\n'))
		return err
	}
}
func (g *Greeting) Html(w http.ResponseWriter) error {
	_, err := fmt.Fprintf(w, "<html><body><p>Hello, %s. You are visit number %d</p></body></html>\n", g.Name, g.VisitNumber)
	return err
}

func greetingHandler(handler *dispatch.TypedHandler[*Handler], ctn *dispatch.ContentTypeNegotiator) {
	var greetingFunc dispatch.ContentTypeHandler[*Handler, *Greeting] = func(r *Handler) (*Greeting, error) {
		name := r.Request().PathValue("name")
		if name == "" {
			name = "World"
		}
		return &Greeting{
			Name:        name,
			VisitNumber: r.VisitNumber,
		}, nil
	}
	typedHandler := greetingFunc.AsTypedHandler(ctn, log.Default().Writer())
	handler.HandleFunc("/{$}", typedHandler)
	handler.HandleFunc("/{name}", typedHandler)
}
