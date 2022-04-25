package auth

import (
	"context"
	"errors"
	"net/http"
)

var (
	ErrUnauthorized    = errors.New("unauthorized")
	ErrUnknownIdentity = errors.New("no identity")
)

type authedctxkey int

const (
	KeyUID authedctxkey = iota
)

type AuthProvider interface {
	AuthenticateRequest(context.Context, *http.Request) (context.Context, error)
	GetUIDFromRequest(*http.Request) (string, error)
}
