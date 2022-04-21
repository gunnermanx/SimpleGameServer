package server

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"time"

	"github.com/gunnermanx/simplegameserver/auth"
	"github.com/gunnermanx/simplegameserver/config"
	"github.com/gunnermanx/simplegameserver/datastore"
	"github.com/sirupsen/logrus"
)

// The server should handle the following responsibilities
//
// Game management (creation/deletion)
// Joining/Leaving games
// Relay messages between the server and clients

type SimpleGameServer struct {
	config   *config.ServerConfig
	serveMux *http.ServeMux
	server   *http.Server
	logger   *logrus.Logger

	games   map[string]*Game
	players map[string]*Player

	datastore    datastore.Datastore
	authProvider auth.AuthProvider

	gameLogic GameLogic
}

func New(
	conf *config.ServerConfig,
	logger *logrus.Logger,
	ap auth.AuthProvider,
	ds datastore.Datastore,
) (s *SimpleGameServer, err error) {

	s = &SimpleGameServer{
		config:       conf,
		logger:       logger,
		authProvider: ap,
		datastore:    ds,
		serveMux:     &http.ServeMux{},
		games:        make(map[string]*Game),
		players:      make(map[string]*Player),
	}

	s.setupHandlers()
	s.server = &http.Server{
		Handler: s,
	}

	return
}

func (sgs *SimpleGameServer) Start() (err error) {
	var listener net.Listener
	if listener, err = net.Listen("tcp", fmt.Sprintf(":%s", sgs.config.Port)); err != nil {
		sgs.logger.Errorf("Failed to start game server: %s", err.Error())
		return
	}

	errc := make(chan error, 1)
	go func() {
		sgs.logger.Infof("Starting game server on: %s", listener.Addr().String())
		errc <- sgs.server.Serve(listener)
	}()

	// Wait for termination or errors
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, os.Interrupt)
	select {
	case err := <-errc:
		sgs.logger.Errorf("Failed to serve: %s", err.Error())
	case sig := <-sigs:
		sgs.logger.Errorf("Terminating on sig: %v", sig)
	}

	// Gracefully shutdown with timeout of 10s
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()
	return sgs.server.Shutdown(ctx)
}

func (sgs *SimpleGameServer) RegisterGameLogic(gamelogic GameLogic) {
	sgs.gameLogic = gamelogic
}

func (sgs *SimpleGameServer) connect(playerID string) (err error) {
	// TODO pull from db
	// TODO add some protection

	var exists bool
	if _, exists = sgs.players[playerID]; !exists {
		// log player creation/connection?
		sgs.logger.Infof("player connected: %s", playerID)

		sgs.players[playerID] = &Player{
			ID: playerID,
			// some other things, maybe pull once from db to get some info
		}
	}
	return
}
