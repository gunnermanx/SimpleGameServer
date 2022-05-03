package game_errors

import (
	"errors"
)

var (
	ErrGameNotFound                  = errors.New("game with ID not found")
	ErrGameTimedOutWaitingForPlayers = errors.New("timed out waiting for players")
	ErrGameFull                      = errors.New("game is full")
	ErrGamePlayerAlreadyExists       = errors.New("player is already in the game")
)
