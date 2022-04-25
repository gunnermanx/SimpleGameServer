package datastore

import (
	"github.com/gunnermanx/simplegameserver/datastore/model"
)

type Datastore interface {
	FindUser(playerID string) (model.User, error)
	FindMatchmakingData(playerID string) (model.MatchmakingData, error)
}
