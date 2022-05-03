package matchmaking

type MatchmakingPlayer struct {
	Rating int
}

func (sms *SimpleMatchmakingServer) GetPlayer(
	playerID string,
) (player *MatchmakingPlayer, err error) {
	var exists bool
	sms.playersMutex.Lock()
	if player, exists = sms.players[playerID]; exists {

		// get the player from the db

		player = &MatchmakingPlayer{}
	}
	sms.playersMutex.Unlock()

	return
}
