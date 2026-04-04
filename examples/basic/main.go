package main

import (
	"context"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/4thPlanet/dispatch"
)

type ListenerConfig struct {
	Protocol string
	Address  string
}

type Config struct {
	Listener ListenerConfig
}

var config = Config{
	Listener: ListenerConfig{
		Protocol: "unix",
		Address:  "./server.sock",
	},
}

func main() {
	server := dispatch.NewServer()
	server.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("Hello, World!"))
	})

	listener, err := net.Listen(config.Listener.Protocol, config.Listener.Address)
	if err != nil {
		log.Fatalf("Unable to listen on unix socket: %v", err)
	}

	c := make(chan os.Signal, 1)
	signal.Notify(c, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-c
		server.Shutdown(context.TODO())
		listener.Close()
		os.Exit(0)
	}()
	server.Serve(listener)
}
