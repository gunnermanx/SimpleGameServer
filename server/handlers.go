package server

import (
	"context"
	"encoding/json"
	"net/http"
	"time"
)

const (
	REQUEST_TIMEOUT_S = 5
)

const (
	CREATE_GAME_PATH = "/game/create"
	JOIN_GAME_PATH   = "/game/join"
)

func (sgs *SimpleGameServer) SetupHandlers() {
	sgs.serveMux.HandleFunc(CREATE_GAME_PATH, sgs.CreateGameHandler)
	sgs.serveMux.HandleFunc(JOIN_GAME_PATH, sgs.JoinGameHandler)
}

func (sgs *SimpleGameServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	var err error

	ctx, cancel := context.WithTimeout(context.Background(), REQUEST_TIMEOUT_S*time.Second)
	defer cancel()

	if ctx, err = sgs.authProvider.AuthenticateRequest(ctx, r); err != nil {
		response := make(map[string]string)
		response["reason"] = err.Error()
		WriteResponse(w, http.StatusUnauthorized, response)
		return
	}

	sgs.serveMux.ServeHTTP(w, r.WithContext(ctx))
}

func (sgs *SimpleGameServer) CreateGameHandler(w http.ResponseWriter, r *http.Request) {
	ctx := context.Background()
	var err error
	var game *Game
	response := make(map[string]string)

	if game, err = sgs.CreateGame(ctx); err != nil {
		response["error"] = err.Error()
		WriteResponse(w, http.StatusInternalServerError, response)
	}

	response["gameID"] = game.ID
	WriteResponse(w, http.StatusCreated, response)
}

func (sgs *SimpleGameServer) JoinGameHandler(w http.ResponseWriter, r *http.Request) {

}

func WriteResponse(w http.ResponseWriter, statusCode int, response map[string]string) {
	w.WriteHeader(statusCode)
	w.Header().Set("Content-Type", "application/json")

	var responseBytes []byte
	var err error
	if responseBytes, err = json.Marshal(response); err != nil {
		w.Write([]byte("failed to write response"))
	}

	w.Write(responseBytes)
}
