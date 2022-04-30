package game

import (
	"context"
	"net/http"
	"time"

	"github.com/gunnermanx/simplegameserver/common"
	game "github.com/gunnermanx/simplegameserver/game/game_instance"
	player "github.com/gunnermanx/simplegameserver/game/game_instance/player"
	"github.com/sirupsen/logrus"
)

const (
	REQUEST_TIMEOUT_S = 5
)

const (
	CONNECT_PATH     = "/connect"
	CREATE_GAME_PATH = "/game/create"
	JOIN_GAME_PATH   = "/game/join"
)

const (
	WS_STATUS_INVALID_PARAMETERS = 4000
)

const (
	DEFAULT_WAIT_FOR_PLAYERS_TIMEOUT_S = 60
)

func (sgs *SimpleGameServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(context.Background(), REQUEST_TIMEOUT_S*time.Second)
	defer cancel()

	var err error
	if ctx, err = sgs.authProvider.AuthenticateRequest(ctx, r); err != nil {
		common.WriteErrorResponse(w, http.StatusUnauthorized, err.Error())
		return
	}
	sgs.serveMux.ServeHTTP(w, r.WithContext(ctx))
}

// RegisterHandler is used by custom game servers to register new http handlers for the given pattern
func (sgs *SimpleGameServer) RegisterHandler(pattern string, handler func(http.ResponseWriter, *http.Request)) {
	sgs.serveMux.HandleFunc(pattern, handler)
}

func (sgs *SimpleGameServer) setupHandlers() {
	sgs.serveMux.HandleFunc(CONNECT_PATH, sgs.connectHandler)
	sgs.serveMux.HandleFunc(CREATE_GAME_PATH, sgs.createGameHandler)
	sgs.serveMux.HandleFunc(JOIN_GAME_PATH, sgs.joinGameHandler)
}

func (sgs *SimpleGameServer) connectHandler(w http.ResponseWriter, r *http.Request) {
	var err error
	var playerID string
	if playerID, err = sgs.authProvider.GetUIDFromRequest(r); err != nil {
		common.WriteErrorResponse(w, http.StatusBadRequest, err.Error())
		return
	}
	if sgs.connect(playerID); err != nil {
		common.WriteErrorResponse(w, http.StatusInternalServerError, err.Error())
		return
	}
	common.WriteResponse(w, http.StatusOK, common.ResponseData{})
}

func (sgs *SimpleGameServer) createGameHandler(w http.ResponseWriter, r *http.Request) {
	var err error
	var g *game.Game

	var statusCode int
	var req CreateGameRequest
	if statusCode, err = common.UnmarshalJSONRequestBody(w, r, &req); err != nil {
		common.WriteErrorResponse(w, statusCode, err.Error())
		return
	}
	if req.NumPlayers == 0 {
		common.WriteErrorResponse(w, http.StatusBadRequest, "numPlayers field is missing or 0")
		return
	}
	if req.WaitForPlayersTimeout == 0 {
		req.WaitForPlayersTimeout = DEFAULT_WAIT_FOR_PLAYERS_TIMEOUT_S
	}

	if g, err = sgs.createGame(req.NumPlayers, req.WaitForPlayersTimeout); err != nil {
		common.WriteErrorResponse(w, http.StatusInternalServerError, err.Error())
		return
	}

	common.WriteResponse(w, http.StatusCreated, common.ResponseData{
		"gameID": g.ID,
	})
}

func (sgs *SimpleGameServer) joinGameHandler(w http.ResponseWriter, r *http.Request) {
	var err error

	var playerID string
	if playerID, err = sgs.authProvider.GetUIDFromRequest(r); err != nil {
		common.WriteErrorResponse(w, http.StatusBadRequest, err.Error())
		return
	}

	gameID := r.URL.Query().Get("id")
	if gameID == "" {
		// TODO, may need to log warn or info?
		//wsconn.Close(WS_STATUS_INVALID_PARAMETERS, "missing or invalid id parameter")
		common.WriteErrorResponse(w, http.StatusBadRequest, err.Error())
		return
	}

	var player *player.SGSGamePlayer
	if player, err = sgs.createPlayer(playerID, w, r); err != nil {
		sgs.logger.WithFields(logrus.Fields{
			"playerID": playerID,
			"gameID":   gameID,
		})
		common.WriteErrorResponse(w, http.StatusInternalServerError, err.Error())
		return
	}

	if err = sgs.joinGame(gameID, player); err != nil {
		sgs.logger.WithFields(logrus.Fields{
			"playerID": playerID,
			"gameID":   gameID,
			"error":    err.Error(),
		}).Error("failed to join game")
		player.CloseConnectionWithError(err)
	}
}
