package server

import (
	"context"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"time"

	"github.com/gunnermanx/simplegameserver/config"
)

type SimpleGameServer struct {
	config   config.ServerConfig
	serveMux http.ServeMux
	server   http.Server
}

func New(conf config.ServerConfig) (s *SimpleGameServer, err error) {
	s = &SimpleGameServer{
		config: conf,
	}

	s.SetupHandlers()

	s.server = http.Server{
		Handler: s,
	}

	return
}

func (mm *SimpleGameServer) Start() (err error) {
	var listener net.Listener
	if listener, err = net.Listen("tcp", fmt.Sprintf(":%s", mm.config.Port)); err != nil {
		return
	}

	errc := make(chan error, 1)
	go func() {
		errc <- mm.server.Serve(listener)
	}()

	// Wait for termination or errors
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, os.Interrupt)
	select {
	case err := <-errc:
		log.Printf("failed to serve: %v", err)
	case sig := <-sigs:
		log.Printf("terminating: %v", sig)
	}

	// Gracefully shutdown with timeout of 10s
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()
	return mm.server.Shutdown(ctx)
}
