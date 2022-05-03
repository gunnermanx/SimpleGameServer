package game_instance

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	errors "github.com/gunnermanx/simplegameserver/game/errors"
	messages "github.com/gunnermanx/simplegameserver/game/game_instance/messages"
	mocks "github.com/gunnermanx/simplegameserver/mocks"
	"github.com/stretchr/testify/require"

	"github.com/sirupsen/logrus"
)

func TestRunGame(t *testing.T) {
	p1_id := "p1_id"
	p2_id := "p2_id"

	logger := logrus.New()
	var g *Game

	t.Run("run game", func(t *testing.T) {
		g = NewGame(logger, 2)

		mockCtrl := gomock.NewController(t)
		defer mockCtrl.Finish()
		mockPlayer1 := mocks.NewMockGamePlayer(mockCtrl)
		mockPlayer2 := mocks.NewMockGamePlayer(mockCtrl)

		playerMsg := messages.GameMessage{
			Code: 123,
			Data: "foo",
		}

		mockPlayer1.EXPECT().GetID().Return(p1_id).AnyTimes()
		mockPlayer2.EXPECT().GetID().Return(p2_id).AnyTimes()

		player1Ctx, _ := context.WithCancel(context.Background())
		mockPlayer1.EXPECT().GetContext().Return(player1Ctx).AnyTimes()
		player2Ctx, _ := context.WithCancel(context.Background())
		mockPlayer2.EXPECT().GetContext().Return(player2Ctx).AnyTimes()

		mockPlayer1.EXPECT().Read().Return(playerMsg, nil).AnyTimes()
		mockPlayer2.EXPECT().Read().Return(playerMsg, nil).AnyTimes()

		gameInit := func(
			ctx context.Context, g *Game, playerIDs []string,
		) (out map[string][]messages.GameMessage, err error) {
			gameData := make(map[string]interface{})
			gameData["counter"] = 0
			gameData["foo"] = "bar"
			g.Data = gameData
			return
		}
		gameTick := func(
			ctx context.Context, g *Game, msgs []messages.GameMessage,
		) (complete bool, out map[string][]messages.GameMessage, err error) {
			gameData := g.Data.(map[string]interface{})
			counter := gameData["counter"].(int) + 1
			gameData["counter"] = counter
			complete = counter > 10
			return
		}

		go g.AddPlayer(mockPlayer1)
		go g.AddPlayer(mockPlayer2)

		g.Run(gameInit, gameTick, 50, 5, nil)

		gameData := g.Data.(map[string]interface{})
		require.Equal(t, 11, gameData["counter"])
		require.Equal(t, "bar", gameData["foo"])
	})

	t.Run("game init errors out", func(t *testing.T) {
		g = NewGame(logger, 2)

		mockCtrl := gomock.NewController(t)
		defer mockCtrl.Finish()
		mockPlayer1 := mocks.NewMockGamePlayer(mockCtrl)
		mockPlayer2 := mocks.NewMockGamePlayer(mockCtrl)

		playerMsg := messages.GameMessage{
			Code: 123,
			Data: "foo",
		}

		mockPlayer1.EXPECT().GetID().Return(p1_id).AnyTimes()
		mockPlayer2.EXPECT().GetID().Return(p2_id).AnyTimes()

		player1Ctx, _ := context.WithCancel(context.Background())
		mockPlayer1.EXPECT().GetContext().Return(player1Ctx).AnyTimes()
		player2Ctx, _ := context.WithCancel(context.Background())
		mockPlayer2.EXPECT().GetContext().Return(player2Ctx).AnyTimes()

		mockPlayer1.EXPECT().Read().Return(playerMsg, nil).AnyTimes()
		mockPlayer2.EXPECT().Read().Return(playerMsg, nil).AnyTimes()

		gameInit := func(
			ctx context.Context, g *Game, playerIDs []string,
		) (out map[string][]messages.GameMessage, err error) {
			err = fmt.Errorf("some error in gameinit")
			return
		}
		gameTick := func(
			ctx context.Context, g *Game, msgs []messages.GameMessage,
		) (complete bool, out map[string][]messages.GameMessage, err error) {
			return
		}

		go g.AddPlayer(mockPlayer1)
		go g.AddPlayer(mockPlayer2)

		g.Run(gameInit, gameTick, 50, 5, nil)

		<-g.Context.Done()
	})
}

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

	t.Run("send messages to players", func(t *testing.T) {

		t.Run("two players write messages", func(t *testing.T) {
			g = NewGame(logger, 2)

			mockCtrl := gomock.NewController(t)
			defer mockCtrl.Finish()
			mockPlayer1 := mocks.NewMockGamePlayer(mockCtrl)
			mockPlayer2 := mocks.NewMockGamePlayer(mockCtrl)

			g.Players[p1_id] = mockPlayer1
			g.Players[p2_id] = mockPlayer2

			msg1 := messages.GameMessage{
				Code: 123,
				Data: 1,
			}
			msg2 := messages.GameMessage{
				Code: 123,
				Data: 2,
			}

			msgsToSend := make(map[string][]messages.GameMessage)
			msgsToSend[p1_id] = append(msgsToSend[p1_id], msg1, msg2)
			msgsToSend[p2_id] = append(msgsToSend[p2_id], msg1)

			mockPlayer1.EXPECT().Write(msg1).Return(nil).Times(1)
			mockPlayer1.EXPECT().Write(msg2).Return(nil).Times(1)
			mockPlayer2.EXPECT().Write(msg1).Return(nil).Times(1)
			mockPlayer2.EXPECT().Write(msg2).Return(nil).Times(0)

			err := g.sendMessagesToPlayers(msgsToSend)
			require.NoError(t, err)
		})

		t.Run("no player in game with ID exists", func(t *testing.T) {
			g = NewGame(logger, 2)

			msg1 := messages.GameMessage{
				Code: 123,
				Data: 1,
			}

			msgsToSend := make(map[string][]messages.GameMessage)
			msgsToSend[p1_id] = append(msgsToSend[p1_id], msg1)
			expectedErr := fmt.Errorf("no player in game with ID: %s", p1_id)

			err := g.sendMessagesToPlayers(msgsToSend)
			require.EqualError(t, err, expectedErr.Error())
		})

		t.Run("player.Write returns error", func(t *testing.T) {
			g = NewGame(logger, 2)

			mockCtrl := gomock.NewController(t)
			defer mockCtrl.Finish()
			mockPlayer := mocks.NewMockGamePlayer(mockCtrl)
			mockPlayer.EXPECT().GetID().Return(p1_id).AnyTimes()

			g.Players[p1_id] = mockPlayer

			msg1 := messages.GameMessage{
				Code: 123,
				Data: 1,
			}
			msg2 := messages.GameMessage{
				Code: 123,
				Data: 2,
			}

			msgsToSend := make(map[string][]messages.GameMessage)
			msgsToSend[p1_id] = append(msgsToSend[p1_id], msg1, msg2)
			expectedErr := fmt.Errorf("some error")

			mockPlayer.EXPECT().Write(msg1).Return(expectedErr).Times(1)

			err := g.sendMessagesToPlayers(msgsToSend)
			require.ErrorIs(t, err, expectedErr)
		})
	})
}
