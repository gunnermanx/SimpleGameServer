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

	"github.com/gunnermanx/simplegameserver/auth"
	"github.com/gunnermanx/simplegameserver/config"
	"github.com/gunnermanx/simplegameserver/datastore"
)

// The server should handle the following responsibilities
//
// Game management (creation/deletion)
// Joining/Leaving games
// Relay messages between the server and clients

type SimpleGameServer struct {
	config   config.ServerConfig
	serveMux http.ServeMux
	server   http.Server

	games map[string]*Game

	datastore    datastore.Datastore
	authProvider auth.AuthProvider
}

func New(
	conf config.ServerConfig,
	ds datastore.Datastore,
	ap auth.AuthProvider,
) (s *SimpleGameServer, err error) {

	s = &SimpleGameServer{
		config:       conf,
		datastore:    ds,
		authProvider: ap,
	}

	s.SetupHandlers()

	s.server = http.Server{
		Handler: s,
	}

	s.games = make(map[string]*Game)

	return
}

func (ss *SimpleGameServer) Connect() (err error) {
	// connect to the server,
	// var user model.User
	// if user, err = ss.datastore.FindUser(); err != nil {
	// 	return
	// }

	return
}

func (ss *SimpleGameServer) Start() (err error) {
	var listener net.Listener
	if listener, err = net.Listen("tcp", fmt.Sprintf(":%s", ss.config.Port)); err != nil {
		return
	}

	errc := make(chan error, 1)
	go func() {
		log.Println("Starting game server")
		errc <- ss.server.Serve(listener)
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
	return ss.server.Shutdown(ctx)
}
