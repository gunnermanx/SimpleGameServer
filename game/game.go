package game

import (
	"context"
	"errors"
	"fmt"
	"io"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"nhooyr.io/websocket"
	"nhooyr.io/websocket/wsjson"
)

// Game models a game running on the game server
type Game struct {
	ID           string
	Players      map[string]*Player
	PlayersMutex sync.RWMutex
	Context      context.Context
	Logger       *logrus.Entry

	NumPlayers   int
	GameMessages chan GameMessage

	Data interface{}
}

type GameCompletedCallback func(error, ...interface{})

func NewGame(
	logger *logrus.Logger,
	maxPlayers int,
) (game *Game) {
	game = &Game{
		ID:           uuid.New().String(),
		Players:      make(map[string]*Player),
		GameMessages: make(chan GameMessage),
		NumPlayers:   maxPlayers,
	}
	game.Logger = logger.WithFields(logrus.Fields{
		"gameID": game.ID,
	})
	return
}

// run will run the code for a game instance
//
// run will do the following:
//   1. wait until the number of required
//   2. initialize the game instance once players have joined
//   3. start the game loop
func (g *Game) run(
	gameInit GameInit,
	gameTick GameTick,
	tickIntervalMS int,
	waitForPlayersTimeout int,
	callback GameCompletedCallback,
) {
	// Create a new context for the game
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	g.Context = ctx

	// Create GameCompletedArgs
	var results []interface{}
	var err error
	defer func() {
		if callback != nil {
			callback(err, results)
		}
		g.Logger.Info("game completed")
	}()

	// Wait for players before starting gameloop
	var playerIDs []string
	g.Logger.Debug("started waiting for players")
	if playerIDs, err = g.WaitForPlayers(waitForPlayersTimeout); err != nil {
		g.Logger.Errorf("failed waiting for players: %s", err.Error())
		return
	}
	g.Logger.Debugf("finished waiting for players. p1: %s, p2: %s", playerIDs[0], playerIDs[1])

	// Initialize the game instance
	var out map[string][]GameMessage
	if out, err = gameInit(ctx, g, playerIDs); err != nil {
		g.Logger.Errorf("error in gameinit: %s", err.Error())
		cancel()
	}
	if err = g.sendMessagesToPlayers(out); err != nil {
		// TODO maybe
		cancel()
		return
	}

	// simple game loop:
	ticker := time.NewTicker(time.Duration(tickIntervalMS) * time.Millisecond)
	msgs := []GameMessage{}
	for {
		select {
		case <-ticker.C:

			// Run the gameTick
			var out map[string][]GameMessage
			if out, err = gameTick(ctx, g, msgs); err != nil {
				g.Logger.Errorf("error in gametick: %s", err.Error())
				cancel()
				return
			}

			// Send messages back to players
			if err = g.sendMessagesToPlayers(out); err != nil {
				// TODO maybe
				cancel()
				return
			}

			msgs = nil

		case msg := <-g.GameMessages:
			//g.Logger.Infof("colleting msg from channel: %v", msg)
			msgs = append(msgs, msg)
			// TODO parse them
		}
	}
}

func (g *Game) WaitForPlayers(waitForPlayersTimeout int) (playerIDs []string, err error) {
	ctx, cancel := context.WithTimeout(g.Context, time.Duration(waitForPlayersTimeout)*time.Second)
	defer cancel()

	players := map[string]bool{}
	for {
		select {
		case msg := <-g.GameMessages:
			if msg.Code == PLAYER_JOINED {
				g.Logger.Debugf("player joined: %s", msg.Data.(string))
				players[msg.Data.(string)] = true
			} else if msg.Code == PLAYER_LEFT {
				g.Logger.Debugf("player left: %s", msg.Data.(string))
				delete(players, msg.Data.(string))
			}
			if len(players) == g.NumPlayers {
				for p := range players {
					playerIDs = append(playerIDs, p)
				}
				return
			}
		case <-ctx.Done():
			err = ErrTimedoutWaitingForPlayers
			return
		}
	}
}

func (g *Game) addPlayer(playerID string, wsconn *websocket.Conn) (err error) {

	// TODO handle players connected to server only
	// create or get the player in the game
	var p *Player
	var exists bool

	g.PlayersMutex.Lock()
	if p, exists = g.Players[playerID]; !exists {
		// log player creation?
		p = &Player{
			ID: playerID,
		}
		g.Players[playerID] = p
	}
	g.PlayersMutex.Unlock()
	p.WSConn = wsconn

	// Create a goroutine to handle messages from the player
	go func(game *Game, player *Player) {
		defer func() {
			game.Logger.WithField(
				"playerID", player.ID,
			).Infof("stopped listening on wsconn")

			// TODO verify this works correct
			game.PlayersMutex.Lock()
			delete(game.Players, player.ID)
			game.PlayersMutex.Unlock()

			game.GameMessages <- NewPlayerLeftMessage(player.ID)
		}()

		for {
			var err error
			var gamemsg GameMessage

			select {
			case <-game.Context.Done():
				game.Logger.WithField(
					"playerID", player.ID,
				).Debug("closing wsconn on game completion")
				wsconn.Close(websocket.StatusNormalClosure, "game completion")
				return
			default:
				err = wsjson.Read(game.Context, wsconn, &gamemsg)
				if err != nil {
					if errors.Is(err, context.Canceled) {
						game.Logger.Debugf("context cancelled")
						wsconn.Close(websocket.StatusNormalClosure, "context cancelled")
						return
					} else if errors.Is(err, io.EOF) {
						game.Logger.Debugf("socket closed")
						wsconn.Close(websocket.StatusNormalClosure, "socket closed")
						return
					} else {
						game.Logger.Errorf("err reading message: %s", err)
						wsconn.Close(websocket.StatusUnsupportedData, "bad msg")
						return
					}
				} else {
					//game.Logger.Debugf("received: %v", gamemsg)
					game.GameMessages <- gamemsg
				}
			}
		}
	}(g, p)

	g.GameMessages <- NewPlayerJoinedMessage(p.ID)

	return
}

func (g *Game) sendMessagesToPlayers(out map[string][]GameMessage) (err error) {
	var p *Player
	var exists bool
	for playerID, msgs := range out {
		g.PlayersMutex.RLock()
		p, exists = g.Players[playerID]
		g.PlayersMutex.RUnlock()
		if !exists {
			err = fmt.Errorf("no player in game with ID: %s", playerID)
			return
		}

		for _, msg := range msgs {
			if err = wsjson.Write(g.Context, p.WSConn, &msg); err != nil {
				// TODO
				g.Logger.Info("####### 22222")
				return
			}
		}
	}
	return
}
