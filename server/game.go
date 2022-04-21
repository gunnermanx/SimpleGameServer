package server

import (
	"context"
	"errors"
	"fmt"
	"io"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"nhooyr.io/websocket"
	"nhooyr.io/websocket/wsjson"
)

type GameLogic func(context.Context, *Game) error

type Game struct {
	ID      string
	Players map[string]*Player
	Context context.Context
	Logger  *logrus.Entry

	GameMessages chan GameMessage
}

type Player struct {
	ID     string
	WSConn *websocket.Conn
}

func (sgs *SimpleGameServer) createGame() (game *Game, err error) {
	// TODO need some form of protection here later

	game = &Game{
		ID:           uuid.New().String(),
		Players:      make(map[string]*Player),
		GameMessages: make(chan GameMessage),
	}
	sgs.games[game.ID] = game
	game.Logger = sgs.logger.WithFields(logrus.Fields{
		"gameID": game.ID,
	})

	// Create a goroutine to run the gamelogic
	go func() {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		game.Context = ctx
		sgs.gameLogic(ctx, game)
		sgs.logger.Infof("game %s completed", game.ID)
	}()

	return
}

func (sgs *SimpleGameServer) joinGame(
	gameID string,
	playerID string,
	wsconn *websocket.Conn,
) (err error) {
	// find the game instance with the given ID
	var g *Game
	var exists bool
	if g, exists = sgs.games[gameID]; !exists {
		err = fmt.Errorf("failed to join game, gameID: %s not found", gameID)
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
	if p, exists = g.Players[playerID]; !exists {
		// log player creation?
		p = &Player{
			ID: playerID,
		}
		g.Players[playerID] = p
	}
	p.WSConn = wsconn

	// Create a goroutine to handle messages from the player
	go func(game *Game, player *Player) {
		defer func() {
			game.Logger.WithField(
				"playerID", player.ID,
			).Infof("stopped listening on wsconn")
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
					game.Logger.Debugf("received: %v", gamemsg)
					game.GameMessages <- gamemsg
				}
			}
		}
	}(g, p)

	g.GameMessages <- NewPlayerJoinedMessage(p.ID)

	return
}
