package datastore

import (
	"github.com/gunnermanx/simplegameserver/datastore/model"
)

type Datastore interface {
	FindUser() (model.User, error)
}
