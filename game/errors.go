package server

import (
	"errors"
)

var (
	ErrTimedoutWaitingForPlayers = errors.New("timed out waiting for players")
)
