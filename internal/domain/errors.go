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
var ErrOTPInvalidOrExpired = errors.New("otp is invalid or expired, try again")
var ErrOTPTooManyAttempts = errors.New("too many attempts, try again later")

type OtpCooldownError struct {
	Cooldown time.Duration
}

func (e OtpCooldownError) Error() string {
	return "please wait " + e.Cooldown.String() + " before requesting a new code"
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

// Board Member
var ErrBoardMemberNotFound = errors.New("board member not found")
var ErrBoardMemberAlreadyInBoard = errors.New("user is already a member of this board")

// Group Standings
var ErrInvalidGroupCode = errors.New("invalid group code")

// Match
var ErrInvalidStageCode = errors.New("invalid stage code")
var ErrInvalidStatus = errors.New("invalid status")
var ErrInvalidFifaCode = errors.New("invalid fifa code")
var ErrInvalidDateRange = errors.New("from_date must be before or equal to to_date")

// TODO: documentar en el doc de matches que esto puede pasar si el match esta en TBD
var ErrInvalidWinnerTeam = errors.New("winner team must be either home or away team")

type MatchesNotFoundError struct {
	MatchIDs []int64
}

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
var ErrThirdPlaceNotInConflict = errors.New("third-place promotion is not in conflict, nothing to resolve")
var ErrThirdPlaceInvalidSelection = errors.New("third-place conflict resolution requires exactly 8 distinct teams from the candidate set")

// OAuth
var ErrOAuthStateNotFound = errors.New("oauth state not found")
var ErrMissingIDToken = errors.New("missing id_token")
var ErrOAuthAccountNotFound = errors.New("oauth account not found")
var ErrOAuthAccountNotVerified = errors.New("oauth account not verified")
