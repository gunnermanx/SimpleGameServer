package game_errors

import (
	"errors"
)

var (
	ErrPlayerConnectionClosed = errors.New("player connection closed")
	ErrPlayerBadGameMessage   = errors.New("player sent bad game message")
)
