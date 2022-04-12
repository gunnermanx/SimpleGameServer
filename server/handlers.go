package server

import (
	"context"
	"net/http"
	"time"
)

const (
	REQUEST_TIMEOUT_S = 5
)

const (
	CREATE_GAME_PATH = "/creategame"
)

func (mm *SimpleGameServer) SetupHandlers() {
	mm.serveMux.HandleFunc(CREATE_GAME_PATH, mm.CreateGameHandler)
}

func (mm *SimpleGameServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(context.Background(), REQUEST_TIMEOUT_S*time.Second)
	defer cancel()

	mm.serveMux.ServeHTTP(w, r.WithContext(ctx))
}

func (mm *SimpleGameServer) CreateGameHandler(w http.ResponseWriter, r *http.Request) {
	tmp := []byte("testtest")
	w.Write(tmp)
}
