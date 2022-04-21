package auth

import (
	"context"
	"net/http"
)

type authedctxkey int

const (
	KeyUID authedctxkey = iota
)

type AuthProvider interface {
	AuthenticateRequest(context.Context, *http.Request) (context.Context, error)
	GetUIDFromRequest(*http.Request) (string, error)
}
