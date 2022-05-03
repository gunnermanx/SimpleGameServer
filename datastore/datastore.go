package datastore

import (
	"github.com/gunnermanx/simplegameserver/datastore/model"
)

//go:generate mockgen -destination=../mocks/mock_datastore.go -package=mocks github.com/gunnermanx/simplegameserver/datastore Datastore

type Datastore interface {
	FindUser(playerID string) (model.User, error)
	FindMatchmakingData(playerID string) (model.MatchmakingData, error)
}
