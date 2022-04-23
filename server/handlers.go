package server

import (
	"context"
	"net/http"
	"time"

	"nhooyr.io/websocket"
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

func (sgs *SimpleGameServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(context.Background(), REQUEST_TIMEOUT_S*time.Second)
	defer cancel()

	var err error
	if ctx, err = sgs.authProvider.AuthenticateRequest(ctx, r); err != nil {
		WriteErrorResponse(w, http.StatusUnauthorized, err.Error())
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
		WriteErrorResponse(w, http.StatusBadRequest, err.Error())
		return
	}
	if sgs.connect(playerID); err != nil {
		WriteErrorResponse(w, http.StatusInternalServerError, err.Error())
		return
	}
	WriteResponse(w, http.StatusOK, ResponseData{})
}

func (sgs *SimpleGameServer) createGameHandler(w http.ResponseWriter, r *http.Request) {
	var err error
	var game *Game

	var statusCode int
	var req CreateGameRequest
	if statusCode, err = UnmarshalJSONRequestBody(w, r, &req); err != nil {
		WriteErrorResponse(w, statusCode, err.Error())
		return
	}
	sgs.logger.Infof("req: %v", req)
	if req.NumPlayers == 0 {
		WriteErrorResponse(w, http.StatusBadRequest, "numPlayers field is missing or 0")
		return
	}

	if game, err = sgs.createGame(req.NumPlayers); err != nil {
		WriteErrorResponse(w, http.StatusInternalServerError, err.Error())
		return
	}

	WriteResponse(w, http.StatusCreated, ResponseData{
		"gameID": game.ID,
	})
}

func (sgs *SimpleGameServer) joinGameHandler(w http.ResponseWriter, r *http.Request) {
	var err error

	var wsconn *websocket.Conn
	if wsconn, err = websocket.Accept(w, r, nil); err != nil {
		sgs.logger.Infof("failed to accept connection %w", err)
		return
	}

	// TODO get playerID from context later on
	var playerID string
	if playerID, err = sgs.authProvider.GetUIDFromRequest(r); err != nil {
		WriteErrorResponse(w, http.StatusBadRequest, err.Error())
		return
	}
	gameID := r.URL.Query().Get("id")
	if gameID == "" {
		// TODO, may need to log warn or info?
		wsconn.Close(WS_STATUS_INVALID_PARAMETERS, "missing or invalid id parameter")
		return
	}

	sgs.joinGame(gameID, playerID, wsconn)
}
