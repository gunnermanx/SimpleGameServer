package game_player

import (
	"context"

	messages "github.com/gunnermanx/simplegameserver/game/game_instance/messages"
)

type GamePlayer interface {
	GetID() string
	GetContext() context.Context
	Read() (messages.GameMessage, error)
	Write(messages.GameMessage) error
	CloseConnection()
	CloseConnectionWithError(error)
}
