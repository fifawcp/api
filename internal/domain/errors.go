package domain

import (
	"errors"
	"fmt"
	"strings"
	"time"
)

// Generic
var ErrInvalidCredentials = errors.New("invalid credentials")
var ErrRegistrationFailed = errors.New("registration failed")
var ErrForbidden = errors.New("insufficient permissions")

// OTP
var ErrOTPInvalidOrExpired = errors.New("OTP is invalid or expired")
var ErrOTPTooManyAttempts = errors.New("too many OTP attempts")

type OtpCooldownError struct {
	Cooldown time.Duration
}

func (e OtpCooldownError) Error() string {
	return fmt.Sprintf("please wait %d seconds before requesting a new code", int(e.Cooldown.Seconds()))
}

func ErrOtpCooldown(cooldown time.Duration) error {
	return OtpCooldownError{Cooldown: cooldown}
}

// User
var ErrUserNotFound = errors.New("user not found")
var ErrUserAlreadyExists = errors.New("user already exists")
var ErrUsernameAlreadyExists = errors.New("username is already taken")

// Refresh Token
var ErrRefreshTokenNotFound = errors.New("refresh token not found")
var ErrRefreshTokenInvalidOrExpired = errors.New("refresh token is invalid or expired")

// Session
var ErrSessionNotFound = errors.New("session not found")
var ErrInvalidSessionExpiration = errors.New("invalid session expiration")
var ErrInvalidSessionLastUsed = errors.New("invalid session last used time")

// Board
var ErrBoardNotFound = errors.New("board not found")
var ErrMatchNotFound = errors.New("match not found")
var ErrBoardAlreadyExists = errors.New("board already exists")
var ErrBoardInvalidJoinCode = errors.New("invalid or expired board join code")
var ErrBoardUserAlreadyInBoard = errors.New("user is already in this board")
var ErrMaxBoardMembersExceeded = errors.New("maximum board members exceeded")
var ErrBoardOwnerCannotLeaveWithMembers = errors.New("board owner cannot leave while other members remain")
var ErrBoardIsPublic = errors.New("operation not allowed on public board")

// Board Member
var ErrBoardMemberNotFound = errors.New("board member not found")
var ErrBoardMemberAlreadyInBoard = errors.New("user is already a member of this board")

// Group Standings
var ErrInvalidGroupCode = errors.New("invalid group code")
var ErrInvalidStandingPosition = errors.New("invalid standing position: must be between 1 and 4")

// Match
var ErrInvalidStageCode = errors.New("invalid stage code")
var ErrInvalidStatus = errors.New("invalid status")
var ErrInvalidFifaCode = errors.New("invalid fifa code")
var ErrInvalidDateRange = errors.New("from_date must be before or equal to to_date")
var ErrInvalidQueryParam = errors.New("invalid query parameter")
var ErrInvalidWinnerTeam = errors.New("winner team must be either home or away team")

// Match result — penalty rules
var ErrPenaltyForbidden = errors.New("penalty score is not allowed: group-stage match, or knockout match decided in regular time")
var ErrPenaltyRequired = errors.New("penalty score is required: knockout match ended tied in regular time")
var ErrPenaltyIncomplete = errors.New("penalty score must include both home and away values")
var ErrPenaltyTied = errors.New("penalty score must be decisive: home and away values cannot be equal")

type MatchesNotFoundError struct {
	MatchIDs []int64
}

// TODO: this error shouldn't leak dynamic data to the client
// TODO: instead we can log and return something more generic like "one or more matches not found"
func (e MatchesNotFoundError) Error() string {
	idsStr := fmt.Sprintf("%v", e.MatchIDs)
	idsStr = strings.ReplaceAll(idsStr, " ", ", ")

	return "matches not found: " + idsStr
}

func ErrMatchesNotFound(matchIDs []int64) error {
	return MatchesNotFoundError{MatchIDs: matchIDs}
}

// Admin
var ErrInvalidThirdPlaceTeam = errors.New("team is not a valid third-place team")
var ErrThirdPlaceNotInConflict = errors.New("third-place promotion is not in conflict")
var ErrThirdPlaceInvalidSelection = errors.New("invalid third-place team selection")

// OAuth
var ErrOAuthStateNotFound = errors.New("oauth state not found")
var ErrMissingIDToken = errors.New("missing identity token")
var ErrOAuthAccountNotFound = errors.New("oauth account not found")
var ErrOAuthAccountNotVerified = errors.New("oauth account not verified")

// Pickem
var ErrPickemLocked = errors.New("pickem is locked: tournament has started")
var ErrMatchPickLocked = errors.New("match pick is locked: match has already started")
var ErrInvalidGroupPicks = errors.New("invalid group picks: each group must have exactly 4 distinct teams in positions 1-4")
var ErrInvalidBestThirdTeam = errors.New("each best-third team must be in position 3 within submitted group picks")
var ErrInvalidBracketPickTeam = errors.New("picked team is not a projected participant for this bracket match")
var ErrGroupPicksRequired = errors.New("group picks must be complete (12 groups x 4 teams + 8 best thirds) before bracket picks")
var ErrTeamGroupMismatch = errors.New("a submitted team does not belong to the declared group")
var ErrBestThirdsNotScoreable = errors.New("best-thirds scoring unavailable: not all 8 third-place teams have been placed in the round of 32")
