package game

import (
	"context"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/gunnermanx/simplegameserver/config"
	game "github.com/gunnermanx/simplegameserver/game_server/game"
	messages "github.com/gunnermanx/simplegameserver/game_server/game/messages"
	game_player "github.com/gunnermanx/simplegameserver/game_server/game/player"
	mocks "github.com/gunnermanx/simplegameserver/mocks"

	sgs_errors "github.com/gunnermanx/simplegameserver/game_server/errors"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
)

func TestServer(t *testing.T) {

	p1_id := "p1_id"
	// p2_id := "p2_id"

	game1_id := "game1_id"

	logger := logrus.New()

	config := &config.GameServerConfig{}

	var g *game.Game

	t.Run("join game", func(t *testing.T) {

		t.Run("game exists and isnt full", func(t *testing.T) {
			g = game.NewGame(logger, 2)

			mockCtrl := gomock.NewController(t)
			defer mockCtrl.Finish()
			mockAuthProvider := mocks.NewMockAuthProvider(mockCtrl)
			mockDatastore := mocks.NewMockDatastore(mockCtrl)
			mockPlayer := mocks.NewMockGamePlayer(mockCtrl)

			playerCtx, cancel := context.WithCancel(context.Background())
			mockPlayer.EXPECT().GetID().Return(p1_id).AnyTimes()
			mockPlayer.EXPECT().GetContext().Return(playerCtx).Times(1)

			s := New(
				config,
				logger,
				mockAuthProvider,
				mockDatastore,
			)

			s.games[game1_id] = g

			// AddPlayer starts listening for messages from the player,
			// but we will cancel early to avoid mocking mockPlayer.Read calls for cleanliness
			cancel()
			// AddPlayer will send player added message, blocking the goroutine
			go func() {
				msg := <-g.GameMessages
				require.Equal(t, msg.Code, messages.PLAYER_JOINED)
				require.Equal(t, msg.Data, p1_id)
			}()
			err := s.joinGame(game1_id, mockPlayer)
			require.NoError(t, err)
		})

		t.Run("game exists but is full", func(t *testing.T) {
			g = game.NewGame(logger, 1)

			mockCtrl := gomock.NewController(t)
			defer mockCtrl.Finish()
			mockAuthProvider := mocks.NewMockAuthProvider(mockCtrl)
			mockDatastore := mocks.NewMockDatastore(mockCtrl)
			mockPlayer := mocks.NewMockGamePlayer(mockCtrl)

			s := New(
				config,
				logger,
				mockAuthProvider,
				mockDatastore,
			)

			s.games[game1_id] = g

			// Add an entry into the Players map
			g.Players["some_guy"] = &game_player.SGSGamePlayer{}

			err := s.joinGame(game1_id, mockPlayer)
			require.ErrorIs(t, err, sgs_errors.ErrGameFull)
		})

		t.Run("game doesn't exist", func(t *testing.T) {
			g = game.NewGame(logger, 1)

			mockCtrl := gomock.NewController(t)
			defer mockCtrl.Finish()
			mockAuthProvider := mocks.NewMockAuthProvider(mockCtrl)
			mockDatastore := mocks.NewMockDatastore(mockCtrl)
			mockPlayer := mocks.NewMockGamePlayer(mockCtrl)

			s := New(
				config,
				logger,
				mockAuthProvider,
				mockDatastore,
			)

			err := s.joinGame(game1_id, mockPlayer)
			require.ErrorIs(t, err, sgs_errors.ErrGameNotFound)
		})
	})
}
