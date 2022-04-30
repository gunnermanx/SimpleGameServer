package game_player

import (
	"context"
	"io"
	"net/http"

	sgs_errors "github.com/gunnermanx/simplegameserver/game/errors"
	messages "github.com/gunnermanx/simplegameserver/game/game_instance/messages"
	"github.com/pkg/errors"
	"nhooyr.io/websocket"
	"nhooyr.io/websocket/wsjson"
)

// SGSGamePlayer models a player connected to a game
// and contains the websocket connection used to communicate between the client and server
type SGSGamePlayer struct {
	ID     string
	WSConn *websocket.Conn

	RWCtx       context.Context
	RWCtxCancel context.CancelFunc
}

func NewSGSGamePlayer(id string, w http.ResponseWriter, r *http.Request) (p *SGSGamePlayer, err error) {
	p = &SGSGamePlayer{
		ID: id,
	}
	if p.WSConn, err = websocket.Accept(w, r, nil); err != nil {
		err = errors.Wrap(err, "failed creating player")
	}
	p.RWCtx, p.RWCtxCancel = context.WithCancel(context.Background())
	return
}

func (p *SGSGamePlayer) GetID() string {
	return p.ID
}

func (p *SGSGamePlayer) GetContext() context.Context {
	return p.RWCtx
}

func (p *SGSGamePlayer) Read() (gamemsg messages.GameMessage, err error) {
	if err = wsjson.Read(p.RWCtx, p.WSConn, &gamemsg); err != nil {
		if errors.Is(err, context.Canceled) {
			p.WSConn.Close(websocket.StatusInternalError, "context cancelled")
			err = sgs_errors.ErrContextCancelled
		} else if errors.Is(err, io.EOF) {
			p.WSConn.Close(websocket.StatusNormalClosure, "socket closed")
			err = sgs_errors.ErrPlayerConnectionClosed
		} else {
			p.WSConn.Close(websocket.StatusProtocolError, "bad game message")
			err = sgs_errors.ErrPlayerBadGameMessage
		}
	}
	return
}

func (p *SGSGamePlayer) Write(gamemsg messages.GameMessage) (err error) {
	if err = wsjson.Write(p.RWCtx, p.WSConn, &gamemsg); err != nil {
		// TODO
		p.WSConn.Close(websocket.StatusInternalError, "todo")
		err = sgs_errors.ErrPlayerBadGameMessage
	}
	return
}

func (p *SGSGamePlayer) CloseConnection() {
	p.RWCtxCancel()
	p.WSConn.Close(websocket.StatusNormalClosure, "player connection closed")
}

func (p *SGSGamePlayer) CloseConnectionWithError(err error) {
	p.RWCtxCancel()
	p.WSConn.Close(websocket.StatusAbnormalClosure, err.Error())
}
