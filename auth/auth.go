package auth

import (
	"context"
	"errors"
	"net/http"
)

//go:generate mockgen -destination=../mocks/mock_authprovider.go -package=mocks github.com/gunnermanx/simplegameserver/auth AuthProvider

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
