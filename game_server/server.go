package game

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"time"

	"github.com/gunnermanx/simplegameserver/auth"
	"github.com/gunnermanx/simplegameserver/config"
	"github.com/gunnermanx/simplegameserver/datastore"

	sgs_errors "github.com/gunnermanx/simplegameserver/game_server/errors"
	game "github.com/gunnermanx/simplegameserver/game_server/game"
	player "github.com/gunnermanx/simplegameserver/game_server/game/player"

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

// SimpleGameServer encapsulates the functionality of a simple game server
//
// The functionality available are:
//   Game management (creation/deletion)
//   Joining/Leaving games
//   Relay messages between the server and clients
type SimpleGameServer struct {
	config   *config.GameServerConfig
	serveMux *http.ServeMux
	server   *http.Server
	logger   *logrus.Logger

	games        map[string]*game.Game
	gamesMutex   sync.RWMutex
	players      map[string]player.GamePlayer
	playersMutex sync.RWMutex

	datastore    datastore.Datastore
	authProvider auth.AuthProvider

	gameInit game.GameInit
	gameTick game.GameTick
}

func New(
	conf *config.GameServerConfig,
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
		games:        make(map[string]*game.Game),
		players:      make(map[string]player.GamePlayer),
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

func (sgs *SimpleGameServer) WithGameInit(init game.GameInit) {
	sgs.gameInit = init
}

func (sgs *SimpleGameServer) WithGameTick(tick game.GameTick) {
	sgs.gameTick = tick
}

// connect will connect a player to the server
// during connect, the server will fetch player data and cache it on the server
// TODO
func (sgs *SimpleGameServer) connect(playerID string) (err error) {
	// TODO pull from db
	// TODO add some protection

	var exists bool
	sgs.playersMutex.RLock()
	_, exists = sgs.players[playerID]
	sgs.playersMutex.RUnlock()
	if !exists {
		sgs.logger.WithField("playerID", playerID).Infof("player connected to server")
		sgs.playersMutex.Lock()
		sgs.players[playerID] = &player.SGSGamePlayer{
			ID: playerID,
			// TODO some other things, maybe pull once from db to get some info
		}
		sgs.playersMutex.Unlock()
	}
	return
}

// createGame creates and runs a game instance on the server
func (sgs *SimpleGameServer) createGame(numPlayers int, waitForPlayersTimeout int) (g *game.Game, err error) {
	// TODO need some form of protection here later
	g = game.NewGame(sgs.logger, numPlayers)
	sgs.gamesMutex.Lock()
	sgs.games[g.ID] = g
	sgs.gamesMutex.Unlock()

	// Run the game in a separate goroutine
	go func() {
		var wg sync.WaitGroup
		wg.Add(1)

		go func() {
			defer wg.Done()
			g.Run(
				sgs.gameInit,
				sgs.gameTick,
				sgs.config.TickIntervalMS,
				waitForPlayersTimeout,
				nil,
			)
		}()

		wg.Wait()
		sgs.gamesMutex.Lock()
		delete(sgs.games, g.ID)
		sgs.gamesMutex.Unlock()

	}()

	return
}

func (sgs *SimpleGameServer) createPlayer(playerID string, w http.ResponseWriter, r *http.Request) (p *player.SGSGamePlayer, err error) {
	// TODO, check if playerID is connected to server
	// TODO, should bootstrap some info from server player to game player
	p, err = player.NewSGSGamePlayer(playerID, w, r)
	return
}

func (sgs *SimpleGameServer) getGame(gameID string) (g *game.Game, err error) {
	var exists bool
	sgs.gamesMutex.RLock()
	g, exists = sgs.games[gameID]
	sgs.gamesMutex.RUnlock()
	if !exists {
		err = sgs_errors.ErrGameNotFound
	}
	return
}

// joinGame adds a player to an existing game on the server
func (sgs *SimpleGameServer) joinGame(
	gameID string,
	player player.GamePlayer,
) (err error) {
	// Find the game instance with the given ID
	var g *game.Game
	if g, err = sgs.getGame(gameID); err != nil {
		return
	}
	// Check if the game is full
	g.PlayersMutex.RLock()
	currentNumPlayers := len(g.Players)
	g.PlayersMutex.RUnlock()
	if currentNumPlayers >= g.NumPlayers {
		err = sgs_errors.ErrGameFull
		return
	}

	// Add the player to the game
	g.AddPlayer(player)

	return
}
