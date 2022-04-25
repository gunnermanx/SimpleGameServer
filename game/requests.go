package game

type CreateGameRequest struct {
	NumPlayers            int `json:"numPlayers"`
	WaitForPlayersTimeout int `json:"waitForPlayersTimeout"`
}
