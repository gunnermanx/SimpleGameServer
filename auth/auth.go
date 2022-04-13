package auth

import (
	"context"
	"net/http"
)

type AuthProvider interface {
	AuthenticateRequest(context.Context, *http.Request) (context.Context, error)
}
