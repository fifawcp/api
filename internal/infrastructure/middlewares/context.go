package middlewares

import (
	"context"

	"github.com/ncondes/fifawcp/internal/domain"
	"github.com/ncondes/fifawcp/internal/dtos"
)

type ContextKey string

const (
	RequestInfoContextKey       ContextKey = "request_info"
	AuthenticatedUserContextKey ContextKey = "authenticated_user"
)

func GetRequestInfo(ctx context.Context) *dtos.RequestInfo {
	requestInfo, ok := ctx.Value(RequestInfoContextKey).(*dtos.RequestInfo)

	if !ok {
		return nil
	}
	return requestInfo
}

func GetAuthenticatedUser(ctx context.Context) *domain.User {
	user, ok := ctx.Value(AuthenticatedUserContextKey).(*domain.User)

	if !ok {
		return nil
	}
	return user
}
