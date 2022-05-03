package game_player

import (
	"context"

	messages "github.com/gunnermanx/simplegameserver/game_server/game/messages"
)

//go:generate mockgen -destination=../../../mocks/mock_player.go -package=mocks github.com/gunnermanx/simplegameserver/game_server/game/player GamePlayer

type GamePlayer interface {
	GetID() string
	GetContext() context.Context
	Read() (messages.GameMessage, error)
	Write(messages.GameMessage) error
	CloseConnection()
	CloseConnectionWithError(error)
}
