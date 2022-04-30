package game_instance_test

import (
	"context"
	"testing"

	errors "github.com/gunnermanx/simplegameserver/game/errors"
	game "github.com/gunnermanx/simplegameserver/game/game_instance"
	messages "github.com/gunnermanx/simplegameserver/game/game_instance/messages"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
)

// type MockGamePlayer struct {
// 	ID string
// }

// func (m *MockGamePlayer) GetID() string {
// 	return m.ID
// }
// func (m *MockGamePlayer) GetContext() context.Context {
// 	return context.Background()
// }
// func (m *MockGamePlayer) Read() (messages.GameMessage, error) {
// 	return messages.GameMessage{
// 		Code: 1234,
// 		Data: "test",
// 	}, nil
// }
// func (m *MockGamePlayer) Write(messages.GameMessage) error {

// }
// func (m *MockGamePlayer) CloseConnection()
// func (m *MockGamePlayer) CloseConnectionWithError(error)

func TestGame(t *testing.T) {

	p1_id := "p1_id"
	p2_id := "p2_id"
	p3_id := "p3_id"
	p4_id := "p4_id"

	logger := logrus.New()
	var g *game.Game

	t.Run("create game", func(t *testing.T) {

	})

	t.Run("add player", func(t *testing.T) {
		g = game.NewGame(logger, 2)
		g.Context = context.Background()

		//g.AddPlayer()

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
