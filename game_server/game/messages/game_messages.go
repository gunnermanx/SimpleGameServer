package game_messages

const (
	PLAYER_JOINED = 10
	PLAYER_LEFT   = 11
)

type GameMessage struct {
	Code int         `json:"code"`
	Data interface{} `json:"data"`
}

func NewPlayerJoinedMessage(playerID string) (g GameMessage) {
	return GameMessage{
		Code: PLAYER_JOINED,
		Data: playerID,
	}
}

func NewPlayerLeftMessage(playerID string) (g GameMessage) {
	return GameMessage{
		Code: PLAYER_LEFT,
		Data: playerID,
	}
}
