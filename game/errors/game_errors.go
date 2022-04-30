package game_errors

import (
	"errors"
)

var (
	ErrGameTimedOutWaitingForPlayers = errors.New("timed out waiting for players")
	ErrGameFull                      = errors.New("game is full")
	ErrGamePlayerAlreadyExists       = errors.New("player is already in the game")
)
