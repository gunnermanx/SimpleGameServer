package server

import (
	"context"
	"errors"
	"fmt"
	"io"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"nhooyr.io/websocket"
	"nhooyr.io/websocket/wsjson"
)

type GameLogic func(ctx context.Context, g *Game, playerIDs []string) error

type Game struct {
	sync.Mutex

	ID      string
	Players map[string]*Player
	Context context.Context
	Logger  *logrus.Entry

	NumPlayers   int
	GameMessages chan GameMessage
}

type Player struct {
	ID     string
	WSConn *websocket.Conn
}

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

	// Create a goroutine to run the gamelogic
	go func(g *Game) {
		// Create a new context for the game
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		g.Context = ctx

		// Wait for players before starting game logic
		var playerIDs []string
		var err error
		sgs.logger.Debug("started waiting for players")
		if playerIDs, err = waitForPlayers(g, waitForPlayersTimeout); err != nil {
			sgs.logger.Errorf("failed waiting for players: %s", err.Error())
			cancel()
			return
		}
		sgs.logger.Debugf("finished waiting for players. p1: %s, p2: %s", playerIDs[0], playerIDs[1])

		// Send GameReady message to all players

		// Run game logic
		sgs.gameLogic(ctx, g, playerIDs)
		sgs.logger.Infof("game %s completed", g.ID)
	}(game)

	return
}

func waitForPlayers(g *Game, waitForPlayersTimeout int) (playerIDs []string, err error) {
	ctx, cancel := context.WithTimeout(g.Context, time.Duration(waitForPlayersTimeout)*time.Second)
	defer cancel()

	players := map[string]bool{}
	for {
		select {
		case msg := <-g.GameMessages:
			if msg.Code == PLAYER_JOINED {
				g.Logger.Debugf("player joined: %s", msg.Data.(string))
				players[msg.Data.(string)] = true
			} else if msg.Code == PLAYER_LEFT {
				g.Logger.Debugf("player left: %s", msg.Data.(string))
				delete(players, msg.Data.(string))
			}
			if len(players) == g.NumPlayers {
				for p := range players {
					playerIDs = append(playerIDs, p)
				}
				return
			}
		case <-ctx.Done():
			err = fmt.Errorf("game ended due to lack of players")
			return
		}
	}
}

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

	return g.addPlayer(playerID, wsconn)
}

func (g *Game) addPlayer(playerID string, wsconn *websocket.Conn) (err error) {

	// TODO handle players connected to server only
	// create or get the player in the game
	var p *Player
	var exists bool

	g.Lock()
	if p, exists = g.Players[playerID]; !exists {
		// log player creation?
		p = &Player{
			ID: playerID,
		}
		g.Players[playerID] = p
	}
	g.Unlock()
	p.WSConn = wsconn

	// Create a goroutine to handle messages from the player
	go func(game *Game, player *Player) {
		defer func() {
			game.Logger.WithField(
				"playerID", player.ID,
			).Infof("stopped listening on wsconn")

			// TODO verify this works correct
			delete(game.Players, player.ID)

			game.GameMessages <- NewPlayerLeftMessage(player.ID)
		}()

		for {
			var err error
			var gamemsg GameMessage

			select {
			case <-game.Context.Done():
				game.Logger.WithField(
					"playerID", player.ID,
				).Debug("closing wsconn on game completion")
				wsconn.Close(websocket.StatusNormalClosure, "game completion")
				return
			default:
				err = wsjson.Read(game.Context, wsconn, &gamemsg)
				if err != nil {
					if errors.Is(err, context.Canceled) {
						game.Logger.Debugf("context cancelled")
						wsconn.Close(websocket.StatusNormalClosure, "context cancelled")
						return
					} else if errors.Is(err, io.EOF) {
						game.Logger.Debugf("socket closed")
						wsconn.Close(websocket.StatusNormalClosure, "socket closed")
						return
					} else {
						game.Logger.Errorf("err reading message: %w", err)
						wsconn.Close(websocket.StatusUnsupportedData, "bad msg")
						return
					}
				} else {
					//game.Logger.Debugf("received: %v", gamemsg)
					game.GameMessages <- gamemsg
				}
			}
		}
	}(g, p)

	g.GameMessages <- NewPlayerJoinedMessage(p.ID)

	return
}
