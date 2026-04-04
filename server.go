package dispatch

import (
	"net/http"
)

type Server struct {
	*http.ServeMux
	*http.Server
}

func NewServer() *Server {
	server := &Server{
		ServeMux: http.NewServeMux(),
	}
	server.Server = &http.Server{
		Handler: server.ServeMux,
	}
	return server

}
