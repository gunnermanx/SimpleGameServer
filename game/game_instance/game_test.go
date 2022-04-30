package game_instance

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	errors "github.com/gunnermanx/simplegameserver/game/errors"
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
	var g *Game

	t.Run("add player and remove player", func(t *testing.T) {
		g = NewGame(logger, 2)
		playerCtx, cancel := context.WithCancel(context.Background())

		mockCtrl := gomock.NewController(t)
		defer mockCtrl.Finish()
		mockPlayer := mocks.NewMockGamePlayer(mockCtrl)

		mockPlayer.EXPECT().GetID().Return(p1_id).AnyTimes()
		mockPlayer.EXPECT().GetContext().Return(playerCtx).Times(1)
		// AddPlayer starts listening for messages from the player,
		// but we will cancel early to avoid mocking mockPlayer.Read calls for cleanliness
		cancel()

		// AddPlayer will send player added message, blocking the goroutine
		go func() {
			msg := <-g.GameMessages
			require.Equal(t, msg.Code, messages.PLAYER_JOINED)
			require.Equal(t, msg.Data, p1_id)
		}()
		g.AddPlayer(mockPlayer)
		require.Contains(t, g.Players, p1_id)

		// RemovePlayer will send player removed message, blocking the goroutine
		mockPlayer.EXPECT().CloseConnection().Times(1)

		go func() {
			msg := <-g.GameMessages
			require.Equal(t, msg.Code, messages.PLAYER_LEFT)
			require.Equal(t, msg.Data, p1_id)
		}()
		g.RemovePlayer(mockPlayer)
		require.NotContains(t, g.Players, p1_id)

	})

	t.Run("wait for players", func(t *testing.T) {
		g = NewGame(logger, 4)

		// Have 4 players join, and see if WaitForPlayers return
		// Also for p1, have the player join and leave, then join again
		t.Run("players joined in time", func(t *testing.T) {
			go func() {
				g.GameMessages <- messages.NewPlayerJoinedMessage(p1_id)
				g.GameMessages <- messages.NewPlayerLeftMessage(p1_id)
				g.GameMessages <- messages.NewPlayerJoinedMessage(p1_id)
				g.GameMessages <- messages.NewPlayerJoinedMessage(p2_id)
				g.GameMessages <- messages.NewPlayerJoinedMessage(p3_id)
				g.GameMessages <- messages.NewPlayerJoinedMessage(p4_id)
			}()
			playerIDs, err := g.waitForPlayers(5)

			require.NoError(t, err)
			require.ElementsMatch(t, playerIDs, []string{p1_id, p2_id, p3_id, p4_id})
		})

		// // Set timeout to 1 sec and wait for the timeout
		t.Run("timed out waiting", func(t *testing.T) {
			g = NewGame(logger, 2)

			playerIDs, err := g.waitForPlayers(1)
			require.ErrorIs(t, err, errors.ErrGameTimedOutWaitingForPlayers)
			require.ElementsMatch(t, playerIDs, []string{})
		})
	})

	t.Run("listen to player", func(t *testing.T) {
		playerMsg := messages.GameMessage{
			Code: 123,
			Data: "foo",
		}

		t.Run("player context cancelled", func(t *testing.T) {
			g = NewGame(logger, 2)

			mockCtrl := gomock.NewController(t)
			defer mockCtrl.Finish()
			mockPlayer := mocks.NewMockGamePlayer(mockCtrl)

			mockPlayer.EXPECT().GetID().Return(p1_id).AnyTimes()
			mockPlayer.EXPECT().Read().Return(playerMsg, nil).AnyTimes()
			playerCtx, playerCtxCancel := context.WithCancel(context.Background())
			mockPlayer.EXPECT().GetContext().Return(playerCtx).AnyTimes()

			// Keep reading messages to unblock player.Read calls
			go func() {
				for msg := range g.GameMessages {
					require.Equal(t, msg.Code, playerMsg.Code)
					require.Equal(t, msg.Data, playerMsg.Data)
				}
			}()
			// After 500ms, cancel the player context so the listen loop ends
			go func() {
				time.Sleep(time.Duration(100) * time.Millisecond)
				playerCtxCancel()
			}()

			// listenToPlayer gets blocked on player.Read and on pushing to game.GameMessages
			g.listenToPlayer(mockPlayer)
		})

		t.Run("game context cancelled", func(t *testing.T) {
			g = NewGame(logger, 2)

			mockCtrl := gomock.NewController(t)
			defer mockCtrl.Finish()
			mockPlayer := mocks.NewMockGamePlayer(mockCtrl)

			mockPlayer.EXPECT().GetID().Return(p1_id).AnyTimes()
			mockPlayer.EXPECT().Read().Return(playerMsg, nil).AnyTimes()
			mockPlayer.EXPECT().GetContext().Return(context.Background()).AnyTimes()

			// Keep reading messages to unblock player.Read calls
			go func() {
				for msg := range g.GameMessages {
					require.Equal(t, msg.Code, playerMsg.Code)
					require.Equal(t, msg.Data, playerMsg.Data)
				}
			}()
			// After 500ms, cancel the game context so the listen loop ends
			go func() {
				time.Sleep(time.Duration(100) * time.Millisecond)
				g.Cancel()
			}()

			// listenToPlayer gets blocked on player.Read and on pushing to game.GameMessages
			g.listenToPlayer(mockPlayer)
		})

		t.Run("player.Read returns error", func(t *testing.T) {
			g = NewGame(logger, 2)

			mockCtrl := gomock.NewController(t)
			defer mockCtrl.Finish()
			mockPlayer := mocks.NewMockGamePlayer(mockCtrl)

			mockPlayer.EXPECT().GetID().Return(p1_id).AnyTimes()
			mockPlayer.EXPECT().Read().Return(playerMsg, fmt.Errorf("some error")).AnyTimes()
			mockPlayer.EXPECT().GetContext().Return(context.Background()).AnyTimes()

			// listenToPlayer will exit immediately since player.Read returned an error
			g.listenToPlayer(mockPlayer)
		})

	})
}
