package game_instance_test

import (
	"context"
	"testing"

	"github.com/golang/mock/gomock"
	errors "github.com/gunnermanx/simplegameserver/game/errors"
	game "github.com/gunnermanx/simplegameserver/game/game_instance"
	messages "github.com/gunnermanx/simplegameserver/game/game_instance/messages"
	"github.com/gunnermanx/simplegameserver/game/mocks"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
)

func TestGame(t *testing.T) {

	p1_id := "p1_id"
	p2_id := "p2_id"
	p3_id := "p3_id"
	p4_id := "p4_id"

	logger := logrus.New()
	var g *game.Game

	t.Run("create game", func(t *testing.T) {

	})

	t.Run("add player and remove player", func(t *testing.T) {
		g = game.NewGame(logger, 2)
		g.Context = context.Background()

		mockCtrl := gomock.NewController(t)
		defer mockCtrl.Finish()
		mockPlayer := mocks.NewMockGamePlayer(mockCtrl)

		mockPlayer.EXPECT().GetID().Return(p1_id).AnyTimes()
		mockPlayer.EXPECT().CloseConnection().Times(1)

		var msg messages.GameMessage

		// AddPlayer will send player added message, blocking the goroutine
		go func() {
			msg = <-g.GameMessages
			require.Equal(t, msg.Code, messages.PLAYER_JOINED)
			require.Equal(t, msg.Data, p1_id)
		}()
		g.AddPlayer(mockPlayer)
		require.Contains(t, g.Players, p1_id)

		// RemovePlayer will send player removed message, blocking the goroutine
		go func() {
			msg = <-g.GameMessages
			require.Equal(t, msg.Code, messages.PLAYER_LEFT)
			require.Equal(t, msg.Data, p1_id)
		}()
		g.RemovePlayer(mockPlayer)
		require.NotContains(t, g.Players, p1_id)

	})

	t.Run("wait for players", func(t *testing.T) {
		g = game.NewGame(logger, 4)
		g.Context = context.Background()

		// Have 4 players join, and see if WaitForPlayers return
		// Also for p1, have the player join and leave, then join again
		t.Run("players joined in time", func(t *testing.T) {
			go func() {
				g.GameMessages <- messages.NewPlayerJoinedMessage(p1_id)
				g.GameMessages <- messages.NewPlayerLeftMessage(p1_id)
				g.GameMessages <- messages.NewPlayerJoinedMessage(p1_id)
			}()
			go func() {
				g.GameMessages <- messages.NewPlayerJoinedMessage(p2_id)
			}()
			go func() {
				g.GameMessages <- messages.NewPlayerJoinedMessage(p3_id)
			}()
			go func() {
				g.GameMessages <- messages.NewPlayerJoinedMessage(p4_id)
			}()
			playerIDs, err := g.WaitForPlayers(5)

			require.NoError(t, err)
			require.ElementsMatch(t, playerIDs, []string{p1_id, p2_id, p3_id, p4_id})
		})

		// Set timeout to 1 sec and wait for the timeout
		t.Run("timed out waiting", func(t *testing.T) {
			g = game.NewGame(logger, 2)
			g.Context = context.Background()

			playerIDs, err := g.WaitForPlayers(1)
			require.ErrorIs(t, err, errors.ErrGameTimedOutWaitingForPlayers)
			require.ElementsMatch(t, playerIDs, []string{})
		})

	})

	// t.Run("add player", func(t *testing.T) {

	// 	s := httptest.NewServer(nil)

	// 	c, _, err := websocket.Dial(context.Background(), s.URL, nil)

	// 	_ = c
	// 	_ = err
	// })
}
