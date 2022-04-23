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
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

// The server should handle the following responsibilities
//
// Game management (creation/deletion)
// Joining/Leaving games
// Relay messages between the server and clients

const (
	GRACEFUL_SHUTDOWN_TIME_S = 10
)

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
		serveMux:     http.NewServeMux(),
		games:        make(map[string]*Game),
		players:      make(map[string]*Player),
	}

	s.setupHandlers()
	s.server = &http.Server{
		Handler: s,
	}

	return
}

// Start the game server
func (sgs *SimpleGameServer) Start() (err error) {
	var listener net.Listener
	// TODO remove localhost
	if listener, err = net.Listen("tcp", fmt.Sprintf("localhost:%s", sgs.config.Port)); err != nil {
		err = errors.Wrap(err, "failed to start game server")
		sgs.logger.Error(err)
		return
	}

	// Start the http server
	errc := make(chan error, 1)
	go func() {
		sgs.logger.Infof("Starting game server on: %s", listener.Addr().String())
		errc <- sgs.server.Serve(listener)
	}()

	// Wait for termination or errors
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, os.Interrupt)
	select {
	case err = <-errc:
		sgs.logger.Errorf("failed to serve: %s", err.Error())
	case sig := <-sigs:
		sgs.logger.Errorf("terminating on sig: %v", sig)
	}

	// Gracefully shutdown with timeout of 10s
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*GRACEFUL_SHUTDOWN_TIME_S)
	defer cancel()
	return sgs.server.Shutdown(ctx)
}

// Register game logic for the server
func (sgs *SimpleGameServer) RegisterGameLogic(gamelogic GameLogic) {
	sgs.gameLogic = gamelogic
}

func (sgs *SimpleGameServer) connect(playerID string) (err error) {
	// TODO pull from db
	// TODO add some protection

	var exists bool
	if _, exists = sgs.players[playerID]; !exists {
		sgs.logger.WithField("playerID", playerID).Infof("player connected to server")
		sgs.players[playerID] = &Player{
			ID: playerID,
			// TODO some other things, maybe pull once from db to get some info
		}
	}
	return
}
