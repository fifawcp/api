package handlers

import (
	"context"
	"errors"
	"net/http"

	"github.com/fifawcp/api/internal/domain"
	"github.com/fifawcp/api/internal/httpx"
	"github.com/fifawcp/api/internal/infrastructure/logging"
)

const (
	// 404 Not Found
	codeUserNotFound         = "USER_NOT_FOUND"
	codeSessionNotFound      = "SESSION_NOT_FOUND"
	codeBoardNotFound        = "BOARD_NOT_FOUND"
	codeBoardMemberNotFound  = "BOARD_MEMBER_NOT_FOUND"
	codeMatchNotFound        = "MATCH_NOT_FOUND"
	codeOAuthAccountNotFound = "OAUTH_ACCOUNT_NOT_FOUND"
	codeMatchesNotFound      = "MATCHES_NOT_FOUND"
	codeCompetitionNotFound  = "COMPETITION_NOT_FOUND"

	// 409 Conflict
	codeUserAlreadyExists              = "USER_ALREADY_EXISTS"
	codeUsernameAlreadyExists          = "USERNAME_ALREADY_EXISTS"
	codeBoardMemberAlreadyInBoard      = "BOARD_MEMBER_ALREADY_IN_BOARD"
	codeBoardAlreadyExists             = "BOARD_ALREADY_EXISTS"
	codeBoardUserAlreadyInBoard        = "BOARD_USER_ALREADY_IN_BOARD"
	codeMaxBoardMembersExceeded        = "MAX_BOARD_MEMBERS_EXCEEDED"
	codeCompetitionPickemAlreadyExists = "COMPETITION_PICKEM_ALREADY_EXISTS"
	codeCompetitionAwardsAlreadyExists = "COMPETITION_AWARDS_ALREADY_EXISTS"
	codeCompetitionNameAlreadyExists   = "COMPETITION_NAME_ALREADY_EXISTS"
	codeDuplicatePickForMatch          = "DUPLICATE_PICK_FOR_MATCH"

	// 401 Unauthorized
	codeOTPInvalidOrExpired          = "OTP_INVALID_OR_EXPIRED"
	codeInvalidCredentials           = "INVALID_CREDENTIALS"
	codeRefreshTokenInvalidOrExpired = "REFRESH_TOKEN_INVALID_OR_EXPIRED"
	codeRefreshTokenNotFound         = "REFRESH_TOKEN_NOT_FOUND"
	codeBoardInvalidJoinCode         = "BOARD_INVALID_JOIN_CODE"

	// 429 Too Many Requests
	codeOTPTooManyAttempts = "OTP_TOO_MANY_ATTEMPTS"
	codeOTPCooldown        = "OTP_COOLDOWN"

	// 403 Forbidden
	codeForbidden                 = "FORBIDDEN"
	codeOAuthAccountNotVerified   = "OAUTH_ACCOUNT_NOT_VERIFIED"
	codeBoardIsGlobal             = "BOARD_IS_GLOBAL"
	codeCompetitionForbidden      = "COMPETITION_FORBIDDEN"
	codeBoardManageAdminForbidden = "BOARD_MANAGE_ADMIN_FORBIDDEN"
	codePredictionsHidden         = "PREDICTIONS_HIDDEN"

	// 400 Bad Request
	codeInvalidWinnerTeam          = "INVALID_WINNER_TEAM"
	codeInvalidThirdPlaceTeam      = "INVALID_THIRD_PLACE_TEAM"
	codeThirdPlaceNotInConflict    = "THIRD_PLACE_NOT_IN_CONFLICT"
	codeThirdPlaceInvalidSelection = "THIRD_PLACE_INVALID_SELECTION"
	codeOAuthStateNotFound         = "OAUTH_STATE_NOT_FOUND"
	codeInvalidGroupCode           = "INVALID_GROUP_CODE"
	codeInvalidStandingPosition    = "INVALID_STANDING_POSITION"
	codeInvalidStageCode           = "INVALID_STAGE_CODE"
	codeInvalidStatus              = "INVALID_STATUS"
	codeInvalidFifaCode            = "INVALID_FIFA_CODE"
	codeInvalidDateRange           = "INVALID_DATE_RANGE"
	codeInvalidQueryParam          = "INVALID_QUERY_PARAM"
	codePickemLocked               = "PICKEM_LOCKED"
	codeMatchPickLocked            = "MATCH_PICK_LOCKED"
	codeMatchTeamsNotAssigned      = "MATCH_TEAMS_NOT_ASSIGNED"
	codePenaltyForbidden           = "PENALTY_FORBIDDEN"
	codePenaltyRequired            = "PENALTY_REQUIRED"
	codePenaltyIncomplete          = "PENALTY_INCOMPLETE"
	codePenaltyTied                = "PENALTY_TIED"
	codeBestThirdsNotScoreable     = "BEST_THIRDS_NOT_SCOREABLE"
	codeInvalidGroupPicks          = "INVALID_GROUP_PICKS"
	codeInvalidBestThirdTeam       = "INVALID_BEST_THIRD_TEAM"
	codeInvalidBracketPick         = "INVALID_BRACKET_PICK"
	codeGroupPicksRequired         = "GROUP_PICKS_REQUIRED"
	codeTeamGroupMismatch          = "TEAM_GROUP_MISMATCH"
	codeCannotTransferToSelf       = "CANNOT_TRANSFER_OWNERSHIP_TO_SELF"
	codeAwardsLocked               = "AWARDS_LOCKED"
	codeInvalidAwardType           = "INVALID_AWARD_TYPE"
	codeAwardPlayerIneligible      = "AWARD_PLAYER_INELIGIBLE"
	codeAwardWinnersIncomplete     = "AWARD_WINNERS_INCOMPLETE"
	codePlayerNotFound             = "PLAYER_NOT_FOUND"

	// 502 Bad Gateway
	codeMissingIDToken = "MISSING_ID_TOKEN"

	// 500 Internal Server Error
	codeInternalServerError = "INTERNAL_SERVER_ERROR"
)

func handleServiceError(w http.ResponseWriter, r *http.Request, err error, logger logging.Logger) {
	if errors.Is(err, context.Canceled) {
		return
	}

	var cooldownErr domain.OtpCooldownError
	var matchesNotFoundErr domain.MatchesNotFoundError

	switch {

	// 404 Not Found
	case errors.Is(err, domain.ErrUserNotFound):
		httpx.NotFound(w, r, codeUserNotFound, domain.ErrUserNotFound.Error())
	case errors.Is(err, domain.ErrSessionNotFound):
		httpx.NotFound(w, r, codeSessionNotFound, domain.ErrSessionNotFound.Error())
	case errors.Is(err, domain.ErrBoardNotFound):
		httpx.NotFound(w, r, codeBoardNotFound, domain.ErrBoardNotFound.Error())
	case errors.Is(err, domain.ErrBoardMemberNotFound):
		httpx.NotFound(w, r, codeBoardMemberNotFound, domain.ErrBoardMemberNotFound.Error())
	case errors.Is(err, domain.ErrMatchNotFound):
		httpx.NotFound(w, r, codeMatchNotFound, domain.ErrMatchNotFound.Error())
	case errors.Is(err, domain.ErrOAuthAccountNotFound):
		httpx.NotFound(w, r, codeOAuthAccountNotFound, domain.ErrOAuthAccountNotFound.Error())
	case errors.As(err, &matchesNotFoundErr):
		httpx.NotFound(w, r, codeMatchesNotFound, matchesNotFoundErr.Error())

	// 404 Not Found (competitions)
	case errors.Is(err, domain.ErrCompetitionNotFound):
		httpx.NotFound(w, r, codeCompetitionNotFound, domain.ErrCompetitionNotFound.Error())

	// 409 Conflict
	case errors.Is(err, domain.ErrCompetitionPickemAlreadyExists):
		httpx.Conflict(w, r, codeCompetitionPickemAlreadyExists, domain.ErrCompetitionPickemAlreadyExists.Error())
	case errors.Is(err, domain.ErrCompetitionAwardsAlreadyExists):
		httpx.Conflict(w, r, codeCompetitionAwardsAlreadyExists, domain.ErrCompetitionAwardsAlreadyExists.Error())
	case errors.Is(err, domain.ErrUserAlreadyExists):
		httpx.Conflict(w, r, codeUserAlreadyExists, domain.ErrUserAlreadyExists.Error())
	case errors.Is(err, domain.ErrUsernameAlreadyExists):
		httpx.Conflict(w, r, codeUsernameAlreadyExists, domain.ErrUsernameAlreadyExists.Error())
	case errors.Is(err, domain.ErrBoardMemberAlreadyInBoard):
		httpx.Conflict(w, r, codeBoardMemberAlreadyInBoard, domain.ErrBoardMemberAlreadyInBoard.Error())
	case errors.Is(err, domain.ErrBoardAlreadyExists):
		httpx.Conflict(w, r, codeBoardAlreadyExists, domain.ErrBoardAlreadyExists.Error())
	case errors.Is(err, domain.ErrBoardUserAlreadyInBoard):
		httpx.Conflict(w, r, codeBoardUserAlreadyInBoard, domain.ErrBoardUserAlreadyInBoard.Error())
	case errors.Is(err, domain.ErrMaxBoardMembersExceeded):
		httpx.Conflict(w, r, codeMaxBoardMembersExceeded, domain.ErrMaxBoardMembersExceeded.Error())

	// 401 Unauthorized
	case errors.Is(err, domain.ErrOTPInvalidOrExpired):
		httpx.Unauthorized(w, r, codeOTPInvalidOrExpired, domain.ErrOTPInvalidOrExpired.Error())
	case errors.Is(err, domain.ErrInvalidCredentials):
		httpx.Unauthorized(w, r, codeInvalidCredentials, domain.ErrInvalidCredentials.Error())
	case errors.Is(err, domain.ErrRefreshTokenInvalidOrExpired):
		httpx.Unauthorized(w, r, codeRefreshTokenInvalidOrExpired, domain.ErrRefreshTokenInvalidOrExpired.Error())
	case errors.Is(err, domain.ErrRefreshTokenNotFound):
		httpx.Unauthorized(w, r, codeRefreshTokenNotFound, domain.ErrRefreshTokenNotFound.Error())
	case errors.Is(err, domain.ErrBoardInvalidJoinCode):
		httpx.Unauthorized(w, r, codeBoardInvalidJoinCode, domain.ErrBoardInvalidJoinCode.Error())

	// 429 Too Many Requests
	case errors.Is(err, domain.ErrOTPTooManyAttempts):
		httpx.TooManyRequests(w, r, codeOTPTooManyAttempts, domain.ErrOTPTooManyAttempts.Error())
	case errors.As(err, &cooldownErr):
		httpx.TooManyRequests(w, r, codeOTPCooldown, cooldownErr.Error())

	// 403 Forbidden
	case errors.Is(err, domain.ErrForbidden):
		httpx.Forbidden(w, r, codeForbidden, domain.ErrForbidden.Error())
	case errors.Is(err, domain.ErrOAuthAccountNotVerified):
		httpx.Forbidden(w, r, codeOAuthAccountNotVerified, domain.ErrOAuthAccountNotVerified.Error())
	case errors.Is(err, domain.ErrBoardIsGlobal):
		httpx.Forbidden(w, r, codeBoardIsGlobal, domain.ErrBoardIsGlobal.Error())
	case errors.Is(err, domain.ErrCompetitionForbidden):
		httpx.Forbidden(w, r, codeCompetitionForbidden, domain.ErrCompetitionForbidden.Error())
	case errors.Is(err, domain.ErrBoardManageAdminForbidden):
		httpx.Forbidden(w, r, codeBoardManageAdminForbidden, domain.ErrBoardManageAdminForbidden.Error())
	case errors.Is(err, domain.ErrPredictionsHidden):
		httpx.Forbidden(w, r, codePredictionsHidden, domain.ErrPredictionsHidden.Error())

	// 400 Bad Request
	case errors.Is(err, domain.ErrInvalidWinnerTeam):
		httpx.BadRequest(w, r, codeInvalidWinnerTeam, domain.ErrInvalidWinnerTeam.Error())
	case errors.Is(err, domain.ErrInvalidThirdPlaceTeam):
		httpx.BadRequest(w, r, codeInvalidThirdPlaceTeam, domain.ErrInvalidThirdPlaceTeam.Error())
	case errors.Is(err, domain.ErrThirdPlaceNotInConflict):
		httpx.BadRequest(w, r, codeThirdPlaceNotInConflict, domain.ErrThirdPlaceNotInConflict.Error())
	case errors.Is(err, domain.ErrThirdPlaceInvalidSelection):
		httpx.BadRequest(w, r, codeThirdPlaceInvalidSelection, domain.ErrThirdPlaceInvalidSelection.Error())
	case errors.Is(err, domain.ErrOAuthStateNotFound):
		httpx.BadRequest(w, r, codeOAuthStateNotFound, domain.ErrOAuthStateNotFound.Error())
	case errors.Is(err, domain.ErrInvalidGroupCode):
		httpx.BadRequest(w, r, codeInvalidGroupCode, domain.ErrInvalidGroupCode.Error())
	case errors.Is(err, domain.ErrInvalidStandingPosition):
		httpx.BadRequest(w, r, codeInvalidStandingPosition, domain.ErrInvalidStandingPosition.Error())
	case errors.Is(err, domain.ErrInvalidStageCode):
		httpx.BadRequest(w, r, codeInvalidStageCode, domain.ErrInvalidStageCode.Error())
	case errors.Is(err, domain.ErrInvalidStatus):
		httpx.BadRequest(w, r, codeInvalidStatus, domain.ErrInvalidStatus.Error())
	case errors.Is(err, domain.ErrInvalidFifaCode):
		httpx.BadRequest(w, r, codeInvalidFifaCode, domain.ErrInvalidFifaCode.Error())
	case errors.Is(err, domain.ErrInvalidDateRange):
		httpx.BadRequest(w, r, codeInvalidDateRange, domain.ErrInvalidDateRange.Error())
	case errors.Is(err, domain.ErrInvalidQueryParam):
		httpx.BadRequest(w, r, codeInvalidQueryParam, err.Error())
	case errors.Is(err, domain.ErrPickemLocked):
		httpx.BadRequest(w, r, codePickemLocked, domain.ErrPickemLocked.Error())
	case errors.Is(err, domain.ErrMatchPickLocked):
		httpx.BadRequest(w, r, codeMatchPickLocked, domain.ErrMatchPickLocked.Error())
	case errors.Is(err, domain.ErrMatchTeamsNotAssigned):
		httpx.BadRequest(w, r, codeMatchTeamsNotAssigned, domain.ErrMatchTeamsNotAssigned.Error())
	case errors.Is(err, domain.ErrPenaltyForbidden):
		httpx.BadRequest(w, r, codePenaltyForbidden, domain.ErrPenaltyForbidden.Error())
	case errors.Is(err, domain.ErrPenaltyRequired):
		httpx.BadRequest(w, r, codePenaltyRequired, domain.ErrPenaltyRequired.Error())
	case errors.Is(err, domain.ErrPenaltyIncomplete):
		httpx.BadRequest(w, r, codePenaltyIncomplete, domain.ErrPenaltyIncomplete.Error())
	case errors.Is(err, domain.ErrPenaltyTied):
		httpx.BadRequest(w, r, codePenaltyTied, domain.ErrPenaltyTied.Error())
	case errors.Is(err, domain.ErrInvalidGroupPicks):
		httpx.BadRequest(w, r, codeInvalidGroupPicks, domain.ErrInvalidGroupPicks.Error())
	case errors.Is(err, domain.ErrInvalidBestThirdTeam):
		httpx.BadRequest(w, r, codeInvalidBestThirdTeam, domain.ErrInvalidBestThirdTeam.Error())
	case errors.Is(err, domain.ErrInvalidBracketPickTeam):
		httpx.BadRequest(w, r, codeInvalidBracketPick, domain.ErrInvalidBracketPickTeam.Error())
	case errors.Is(err, domain.ErrGroupPicksRequired):
		httpx.BadRequest(w, r, codeGroupPicksRequired, domain.ErrGroupPicksRequired.Error())
	case errors.Is(err, domain.ErrTeamGroupMismatch):
		httpx.BadRequest(w, r, codeTeamGroupMismatch, domain.ErrTeamGroupMismatch.Error())
	case errors.Is(err, domain.ErrBestThirdsNotScoreable):
		httpx.BadRequest(w, r, codeBestThirdsNotScoreable, domain.ErrBestThirdsNotScoreable.Error())
	case errors.Is(err, domain.ErrAwardsLocked):
		httpx.BadRequest(w, r, codeAwardsLocked, domain.ErrAwardsLocked.Error())
	case errors.Is(err, domain.ErrInvalidAwardType):
		httpx.BadRequest(w, r, codeInvalidAwardType, domain.ErrInvalidAwardType.Error())
	case errors.Is(err, domain.ErrAwardPlayerIneligible):
		httpx.BadRequest(w, r, codeAwardPlayerIneligible, domain.ErrAwardPlayerIneligible.Error())
	case errors.Is(err, domain.ErrAwardWinnersIncomplete):
		httpx.BadRequest(w, r, codeAwardWinnersIncomplete, domain.ErrAwardWinnersIncomplete.Error())
	case errors.Is(err, domain.ErrPlayerNotFound):
		httpx.NotFound(w, r, codePlayerNotFound, domain.ErrPlayerNotFound.Error())
	case errors.Is(err, domain.ErrCannotTransferOwnershipToSelf):
		httpx.BadRequest(w, r, codeCannotTransferToSelf, domain.ErrCannotTransferOwnershipToSelf.Error())
	case errors.Is(err, domain.ErrCompetitionNameAlreadyExists):
		httpx.Conflict(w, r, codeCompetitionNameAlreadyExists, domain.ErrCompetitionNameAlreadyExists.Error())
	case errors.Is(err, domain.ErrDuplicatePickForMatch):
		httpx.Conflict(w, r, codeDuplicatePickForMatch, domain.ErrDuplicatePickForMatch.Error())

	// 502 Bad Gateway
	case errors.Is(err, domain.ErrMissingIDToken):
		httpx.BadGateway(w, r, codeMissingIDToken, domain.ErrMissingIDToken.Error())

	// 500 Internal Server Error
	default:
		fields := []any{logging.Error, err.Error()}
		if r != nil {
			fields = append(fields, logging.Method, r.Method, logging.Path, r.URL.Path)
		}

		logger.Error("internal server error", fields...)
		httpx.InternalServerError(w, r, codeInternalServerError, "internal server error")
	}
}
