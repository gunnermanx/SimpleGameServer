package server_test

import (
	"context"
	"testing"

	"github.com/gunnermanx/simplegameserver/server"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
)

func TestGame(t *testing.T) {

	p1_id := "p1_id"
	p2_id := "p2_id"

	logger := logrus.New()
	game := server.NewGame(logger, 2)
	game.Context = context.Background()

	t.Run("create game", func(t *testing.T) {

	})

	t.Run("wait for players", func(t *testing.T) {
		// Have two players join, and see if WaitForPlayers return
		// Also for p1, have the player join and leave, then join again
		t.Run("players joined in time", func(t *testing.T) {
			go func() {
				game.GameMessages <- server.NewPlayerJoinedMessage(p1_id)
				game.GameMessages <- server.NewPlayerLeftMessage(p1_id)
				game.GameMessages <- server.NewPlayerJoinedMessage(p1_id)
			}()
			go func() {
				game.GameMessages <- server.NewPlayerJoinedMessage(p2_id)
			}()
			playerIDs, err := game.WaitForPlayers(5)

			require.NoError(t, err)
			require.ElementsMatch(t, playerIDs, []string{p1_id, p2_id})
		})
		// Set timeout to 1 sec and wait for the timeout
		t.Run("timed out waiting", func(t *testing.T) {
			playerIDs, err := game.WaitForPlayers(1)
			require.ErrorIs(t, err, server.ErrTimedoutWaitingForPlayers)
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
