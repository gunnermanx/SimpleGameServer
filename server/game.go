package server

import (
	"context"

	"github.com/google/uuid"
	"nhooyr.io/websocket"
)

type Game struct {
	ID      string
	Players []*Player
}

type Player struct {
	ID   string
	conn *websocket.Conn
}

func (mm *SimpleGameServer) CreateGame(ctx context.Context) (game *Game, err error) {
	// TODO need some form of protection here later

	game = &Game{
		ID: uuid.New().String(),
	}

	mm.games[game.ID] = game

	// need to start the game

	return
}

func (mm *SimpleGameServer) JoinGame(ctx context.Context) {

}

func (g *Game) Start() {

}
