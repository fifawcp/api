package middlewares

import (
	"context"

	"github.com/fifawcp/api/internal/domain"
	"github.com/fifawcp/api/internal/dtos"
)

type ContextKey string

const (
	RequestInfoContextKey       ContextKey = "request_info"
	AuthenticatedUserContextKey ContextKey = "authenticated_user"
	BoardIDContextKey           ContextKey = "board_id"
	BoardMemberRoleContextKey   ContextKey = "board_member_role"
	UserIDContextKey            ContextKey = "user_id"
	MatchIDContextKey           ContextKey = "match_id"
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

func GetBoardID(ctx context.Context) string {
	boardID, ok := ctx.Value(BoardIDContextKey).(string)
	if !ok {
		return ""
	}
	return boardID
}

func GetBoardMemberRole(ctx context.Context) domain.BoardMemberRole {
	boardMemberRole, ok := ctx.Value(BoardMemberRoleContextKey).(domain.BoardMemberRole)
	if !ok {
		return ""
	}
	return boardMemberRole
}

func GetUserID(ctx context.Context) string {
	userID, ok := ctx.Value(UserIDContextKey).(string)
	if !ok {
		return ""
	}
	return userID
}

func GetMatchID(ctx context.Context) int64 {
	matchID, ok := ctx.Value(MatchIDContextKey).(int64)
	if !ok {
		return 0
	}
	return matchID
}
