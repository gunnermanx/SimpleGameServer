package game_errors

import (
	"errors"
)

var (
	ErrContextCancelled = errors.New("context cancelled")
)
