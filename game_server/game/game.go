package game_instance

import (
	"context"
	"fmt"
	"sync"
	"time"

	errors "github.com/gunnermanx/simplegameserver/game_server/errors"
	messages "github.com/gunnermanx/simplegameserver/game_server/game/messages"
	player "github.com/gunnermanx/simplegameserver/game_server/game/player"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

// GameInit is called when a game instance on the server is created
// This should be implemented by a concrete game server and added to the server using WithGameInit
type GameInit func(
	ctx context.Context,
	g *Game,
	playerIDs []string,
) (map[string][]messages.GameMessage, error)

// GameTick is called once every server tick and defines the behavior of the game server
// This should be implemented by a concrete game server and added to the server using WithGameTick
type GameTick func(
	ctx context.Context,
	g *Game,
	msgs []messages.GameMessage,
) (bool, map[string][]messages.GameMessage, error)

// Game models a game running on the game server
type Game struct {
	Logger  *logrus.Entry
	Context context.Context
	Cancel  context.CancelFunc

	ID         string
	NumPlayers int

	Players      map[string]player.GamePlayer
	PlayersMutex sync.RWMutex
	GameMessages chan messages.GameMessage

	Data interface{}
}

type GameCompletedCallback func(error, ...interface{})

func NewGame(
	logger *logrus.Logger,
	maxPlayers int,
) (game *Game) {
	game = &Game{
		ID:           uuid.New().String(),
		Players:      make(map[string]player.GamePlayer),
		GameMessages: make(chan messages.GameMessage),
		NumPlayers:   maxPlayers,
	}
	game.Logger = logger.WithFields(logrus.Fields{
		"gameID": game.ID,
	})
	game.Context, game.Cancel = context.WithCancel(context.Background())
	return
}

// Run will Run the code for a game instance
//
// Run will do the following:
//   1. wait until the number of required
//   2. initialize the game instance once players have joined
//   3. start the game loop
func (g *Game) Run(
	gameInit GameInit,
	gameTick GameTick,
	tickIntervalMS int,
	waitForPlayersTimeout int,
	callback GameCompletedCallback,
) {
	defer g.Cancel()

	var err error

	// Create GameCompletedArgs
	var results []interface{}

	defer func() {
		if callback != nil {
			callback(err, results)
		}
		g.Logger.Info("game completed")
	}()

	// Wait for players before starting gameloop
	var playerIDs []string
	if playerIDs, err = g.waitForPlayers(waitForPlayersTimeout); err != nil {
		return
	}

	// Initialize the game instance
	var out map[string][]messages.GameMessage
	if out, err = gameInit(g.Context, g, playerIDs); err != nil {
		g.Logger.Errorf("error in gameinit: %s", err.Error())
		g.Cancel()
		return
	}
	if err = g.sendMessagesToPlayers(out); err != nil {
		// TODO maybe
		g.Cancel()
		return
	}

	// simple game loop:
	ticker := time.NewTicker(time.Duration(tickIntervalMS) * time.Millisecond)
	msgs := []messages.GameMessage{}
	var complete bool
	for !complete {
		select {
		case <-ticker.C:

			// Run the gameTick

			var out map[string][]messages.GameMessage
			if complete, out, err = gameTick(g.Context, g, msgs); err != nil {
				g.Logger.Errorf("error in gametick: %s", err.Error())
				g.Cancel()
				return
			}

			// Send messages back to players
			if err = g.sendMessagesToPlayers(out); err != nil {
				// TODO maybe
				g.Cancel()
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

func (g *Game) AddPlayer(p player.GamePlayer) {
	g.PlayersMutex.Lock()
	if _, exists := g.Players[p.GetID()]; exists {
		//TODO this is a reconnection? do we need to do more? send reconnection message?
	}
	g.Players[p.GetID()] = p
	g.PlayersMutex.Unlock()

	// Listen for game messages from the player
	go g.listenToPlayer(p)

	g.GameMessages <- messages.NewPlayerJoinedMessage(p.GetID())

	g.Logger.WithField(
		"playerID", p.GetID(),
	).Info("player added to game")
}

func (g *Game) RemovePlayer(p player.GamePlayer) {
	g.PlayersMutex.Lock()
	delete(g.Players, p.GetID())
	g.PlayersMutex.Unlock()

	p.CloseConnection()

	g.GameMessages <- messages.NewPlayerLeftMessage(p.GetID())

	g.Logger.WithField(
		"playerID", p.GetID(),
	).Info("player removed from game")
}

func (g *Game) waitForPlayers(waitForPlayersTimeout int) (playerIDs []string, err error) {
	ctx, cancel := context.WithTimeout(g.Context, time.Duration(waitForPlayersTimeout)*time.Second)
	defer cancel()

	g.Logger.Debug("started waiting for players")

	players := map[string]bool{}
loop:
	for {
		select {
		case msg := <-g.GameMessages:
			if msg.Code == messages.PLAYER_JOINED {
				g.Logger.WithField("playerID", msg.Data.(string)).Debug("player joined")
				players[msg.Data.(string)] = true
			} else if msg.Code == messages.PLAYER_LEFT {
				g.Logger.WithField("playerID", msg.Data.(string)).Debug("player left")
				delete(players, msg.Data.(string))
			}
			if len(players) == g.NumPlayers {
				for p := range players {
					playerIDs = append(playerIDs, p)
				}
				break loop
			}
		case <-ctx.Done():
			err = errors.ErrGameTimedOutWaitingForPlayers
			g.Logger.WithField("error", err.Error()).Error("failed waiting for players")
			break loop
		}
	}

	if err == nil {
		fields := logrus.Fields{}
		for i, playerID := range playerIDs {
			fields[fmt.Sprintf("player%d_ID", i)] = playerID
		}
		g.Logger.WithFields(fields).Info("finished waiting for players")
	}
	return
}

func (g *Game) listenToPlayer(p player.GamePlayer) {
	// defer removing the player from the game
	defer func() {
		g.Logger.WithField(
			"playerID", p.GetID(),
		).Debug("stopped reading messages from player")
	}()

	g.Logger.WithField(
		"playerID", p.GetID(),
	).Debug("started reading messages from player")

	var err error
	var gamemsg messages.GameMessage
readLoop:
	for {
		select {
		case <-p.GetContext().Done():
			break readLoop
		case <-g.Context.Done():
			break readLoop
		default:
			if gamemsg, err = p.Read(); err != nil {
				g.Logger.WithFields(logrus.Fields{
					"playerID": p.GetID(),
					"error":    err.Error(),
				}).Error("failed reading message from player")
				break readLoop
			}
			g.GameMessages <- gamemsg
		}
	}
}

func (g *Game) sendMessagesToPlayers(out map[string][]messages.GameMessage) (err error) {
	var p player.GamePlayer
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
			if err = p.Write(msg); err != nil {
				g.Logger.WithFields(logrus.Fields{
					"playerID": p.GetID(),
					"error":    err.Error(),
				}).Error("failed writing message to player")
				return
			}
		}
	}
	return
}
