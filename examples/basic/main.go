package main

import (
	"context"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

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
		Protocol: "tcp",
		Address:  "localhost:8080",
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
		log.Fatalf("Unable to listen on %s socket: %v", config.Listener.Protocol, err)
	}

	c := make(chan os.Signal, 1)
	signal.Notify(c, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-c
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		if err := server.Shutdown(ctx); err != nil {
			log.Println("Error performing graceful shutdown: ", err)
		}
		listener.Close() // #nosec G104 - Best Effort shutdown of the listener
		os.Exit(0)
	}()
	if err := server.Serve(listener); err != http.ErrServerClosed {
		log.Fatalln("Error serving: ", err)
	}
	select {}
}
