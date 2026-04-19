package dispatch

import (
	"net/http"
	"time"
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
		Handler:           server.ServeMux,
		ReadHeaderTimeout: time.Second * 5,
	}
	return server

}
