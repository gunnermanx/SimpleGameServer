package game

import (
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/gunnermanx/simplegameserver/config"
	mocks "github.com/gunnermanx/simplegameserver/mocks"
	"github.com/sirupsen/logrus"
)

func TestServer(t *testing.T) {

	// p1_id := "p1_id"
	// p2_id := "p2_id"

	logger := logrus.New()

	config := &config.GameServerConfig{}

	//var g *game.Game

	t.Run("join game", func(t *testing.T) {

		t.Run("game exists and isnt full", func(t *testing.T) {
			//g = game.NewGame(logger, 2)

			mockCtrl := gomock.NewController(t)
			defer mockCtrl.Finish()
			mockAuthProvider := mocks.NewMockAuthProvider(mockCtrl)
			mockDatastore := mocks.NewMockDatastore(mockCtrl)

			s := New(
				config,
				logger,
				mockAuthProvider,
				mockDatastore,
			)

			s.joinGame("someID", nil)

		})

	})
}
