package middlewares

import (
	"context"

	"github.com/ncondes/fifawcp/internal/domain"
	"github.com/ncondes/fifawcp/internal/dtos"
)

type ContextKey string

const (
	requestInfoContextKey       ContextKey = "request_info"
	authenticatedUserContextKey ContextKey = "authenticated_user"
)

func GetRequestInfo(ctx context.Context) *dtos.RequestInfo {
	requestInfo, ok := ctx.Value(requestInfoContextKey).(*dtos.RequestInfo)

	if !ok {
		return nil
	}
	return requestInfo
}

func GetAuthenticatedUser(ctx context.Context) *domain.User {
	user, ok := ctx.Value(authenticatedUserContextKey).(*domain.User)

	if !ok {
		return nil
	}
	return user
}
