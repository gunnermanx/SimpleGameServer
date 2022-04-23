package server

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/gunnermanx/simplegameserver/auth"
	"github.com/gunnermanx/simplegameserver/config"
	"github.com/gunnermanx/simplegameserver/datastore"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"nhooyr.io/websocket"
)

// The server should handle the following responsibilities
//
// Game management (creation/deletion)
// Joining/Leaving games
// Relay messages between the server and clients

const (
	GRACEFUL_SHUTDOWN_TIME_S = 10
)

// GameInit is called when a game instance on the server is created
// This should be implemented by a concrete game server and added to the server using WithGameInit
type GameInit func(ctx context.Context, g *Game, playerIDs []string) (map[string][]GameMessage, error)

// GameTick is called once every server tick and defines the behavior of the game server
// This should be implemented by a concrete game server and added to the server using WithGameTick
type GameTick func(ctx context.Context, g *Game, msgs []GameMessage) (map[string][]GameMessage, error)

// SimpleGameServer encapsulates the functionality of a simple game server
//
// The functionality available are:
//   Game management (creation/deletion)
//   Joining/Leaving games
//   Relay messages between the server and clients
type SimpleGameServer struct {
	sync.Mutex

	config   *config.ServerConfig
	serveMux *http.ServeMux
	server   *http.Server
	logger   *logrus.Logger

	games   map[string]*Game
	players map[string]*Player

	datastore    datastore.Datastore
	authProvider auth.AuthProvider

	gameInit GameInit
	gameTick GameTick
}

func New(
	conf *config.ServerConfig,
	logger *logrus.Logger,
	ap auth.AuthProvider,
	ds datastore.Datastore,
) (s *SimpleGameServer) {

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
	if listener, err = net.Listen("tcp", fmt.Sprintf(":%s", sgs.config.Port)); err != nil {
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

func (sgs *SimpleGameServer) WithGameInit(init GameInit) {
	sgs.gameInit = init
}

func (sgs *SimpleGameServer) WithGameTick(tick GameTick) {
	sgs.gameTick = tick
}

// connect will connect a player to the server
// during connect, the server will fetch player data and cache it on the server
// TODO
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

// createGame creates and runs a game instance on the server
func (sgs *SimpleGameServer) createGame(numPlayers int, waitForPlayersTimeout int) (game *Game, err error) {
	// TODO need some form of protection here later
	game = &Game{
		ID:           uuid.New().String(),
		Players:      make(map[string]*Player),
		GameMessages: make(chan GameMessage),
		NumPlayers:   numPlayers,
	}
	game.Logger = sgs.logger.WithFields(logrus.Fields{
		"gameID": game.ID,
	})
	sgs.Lock()
	sgs.games[game.ID] = game
	sgs.Unlock()

	// Run the game in a separate goroutine
	go game.run(sgs.gameInit, sgs.gameTick, sgs.config.TickIntervalMS, waitForPlayersTimeout)

	return
}

// joinGame adds a player to an existing game on the server
func (sgs *SimpleGameServer) joinGame(
	gameID string,
	playerID string,
	wsconn *websocket.Conn,
) (err error) {
	// find the game instance with the given ID
	var g *Game
	var exists bool
	sgs.Lock()
	g, exists = sgs.games[gameID]
	sgs.Unlock()
	if !exists {
		err = fmt.Errorf("failed to join game, gameID: %s. gameID not found", gameID)
		wsconn.Close(WS_STATUS_INVALID_PARAMETERS, err.Error())
		return
	}

	g.Lock()
	currentNumPlayers := len(g.Players)
	g.Unlock()
	if currentNumPlayers >= g.NumPlayers {
		err = fmt.Errorf("failed to join game, gameID: %s. game is full", gameID)
		wsconn.Close(WS_STATUS_INVALID_PARAMETERS, err.Error())
		return
	}

	// Add the player to the game
	return g.addPlayer(playerID, wsconn)
}
