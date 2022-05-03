package matchmaking

import (
	"context"
	"net/http"
	"time"

	"github.com/gunnermanx/simplegameserver/common"
)

const (
	REQUEST_TIMEOUT_S = 5
)

const (
	FIND_MATCH_PATH = "/match/find"
)

func (sms *SimpleMatchmakingServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(context.Background(), REQUEST_TIMEOUT_S*time.Second)
	defer cancel()

	var err error
	if ctx, err = sms.authProvider.AuthenticateRequest(ctx, r); err != nil {
		common.WriteErrorResponse(w, http.StatusUnauthorized, err.Error())
		return
	}
	sms.serveMux.ServeHTTP(w, r.WithContext(ctx))
}

// RegisterHandler is used by custom game servers to register new http handlers for the given pattern
func (sms *SimpleMatchmakingServer) RegisterHandler(pattern string, handler func(http.ResponseWriter, *http.Request)) {
	sms.serveMux.HandleFunc(pattern, handler)
}

func (sms *SimpleMatchmakingServer) setupHandlers() {
	sms.serveMux.HandleFunc(FIND_MATCH_PATH, sms.findMatchHandler)
}

func (sms *SimpleMatchmakingServer) findMatchHandler(w http.ResponseWriter, r *http.Request) {
	var err error
	var playerID string
	if playerID, err = sms.authProvider.GetUIDFromRequest(r); err != nil {
		common.WriteErrorResponse(w, http.StatusBadRequest, err.Error())
		return
	}

	var player *MatchmakingPlayer
	if player, err = sms.GetPlayer(playerID); err != nil {

	}

	_ = player

	// if sgs.connect(playerID); err != nil {
	// 	common.WriteErrorResponse(w, http.StatusInternalServerError, err.Error())
	// 	return
	// }
	common.WriteResponse(w, http.StatusOK, common.ResponseData{})
}
