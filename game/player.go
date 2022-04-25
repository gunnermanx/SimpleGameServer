package server

import "nhooyr.io/websocket"

// Player models a player connected to a game
// and contains the websocket connection used to communicate between the client and server
type Player struct {
	ID     string
	WSConn *websocket.Conn
}

