package game

import (
	"errors"
)

var (
	ErrTimedoutWaitingForPlayers = errors.New("timed out waiting for players")
)
