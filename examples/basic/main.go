package main

import (
	"context"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"

	"github.com/4thPlanet/dispatch"
	"github.com/4thPlanet/dispatch/examples/basic/internal/middleware"
	"github.com/4thPlanet/dispatch/examples/basic/internal/routes"
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

	router := routes.SetupRouter()
	router.UseMiddleware(
		middleware.Logger(),
		middleware.Errors(),
		middleware.Counter(),
	)

	server.Handle("/", router)

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
