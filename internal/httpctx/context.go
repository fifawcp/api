package httpctx

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
	CompetitionIDContextKey     ContextKey = "competition_id"
	ReturnToContextKey          ContextKey = "return_to"
	OAuthStateContextKey        ContextKey = "oauth_state"
	OAuthCodeContextKey         ContextKey = "oauth_code"
	ResponseErrorContextKey     ContextKey = "response_error"
)

type ResponseError struct {
	Code    string
	Message string
}

func GetResponseError(ctx context.Context) *ResponseError {
	responseError, ok := ctx.Value(ResponseErrorContextKey).(*ResponseError)
	if !ok {
		return nil
	}

	return responseError
}

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

func GetBoardID(ctx context.Context) int64 {
	boardID, ok := ctx.Value(BoardIDContextKey).(int64)
	if !ok {
		return 0
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

func GetCompetitionID(ctx context.Context) int64 {
	competitionID, ok := ctx.Value(CompetitionIDContextKey).(int64)
	if !ok {
		return 0
	}
	return competitionID
}

func GetMatchID(ctx context.Context) int64 {
	matchID, ok := ctx.Value(MatchIDContextKey).(int64)
	if !ok {
		return 0
	}
	return matchID
}

func GetReturnTo(ctx context.Context) string {
	returnTo, ok := ctx.Value(ReturnToContextKey).(string)
	if !ok {
		return ""
	}
	return returnTo
}

func GetOAuthState(ctx context.Context) string {
	state, ok := ctx.Value(OAuthStateContextKey).(string)
	if !ok {
		return ""
	}
	return state
}

func GetOAuthCode(ctx context.Context) string {
	code, ok := ctx.Value(OAuthCodeContextKey).(string)
	if !ok {
		return ""
	}
	return code
}
