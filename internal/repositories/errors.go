package repositories

import (
	"database/sql"
	"fmt"

	"github.com/fifawcp/api/internal/domain"
	"github.com/lib/pq"
)

type resourceType string

const (
	foreignKeyViolation       = "23503"
	uniqueViolation           = "23505"
	checkConstraintViolation  = "23514"
	invalidTextRepresentation = "22P02"
)

const (
	resourceUser             resourceType = "user"
	resourceSession          resourceType = "session"
	resourceRefreshToken     resourceType = "refresh_token"
	resourceBoard            resourceType = "board"
	resourceBoardMember      resourceType = "board_member"
	resourceGroupStanding    resourceType = "group_standing"
	resourceMatch            resourceType = "match"
	resourceMatchAPIFixture  resourceType = "match_api_fixture"
	resourceOAuthAccount     resourceType = "oauth_account"
	resourcePickem           resourceType = "pickem"
	resourceMatchScorePick   resourceType = "match_score_pick"
	resourceScoreEvent       resourceType = "score_event"
	resourceTeam             resourceType = "team"
	resourceMatchFairPlay    resourceType = "match_fair_play"
	resourceCompetition      resourceType = "competition"
	resourceCompetitionScore resourceType = "competition_score"
	resourcePlayer           resourceType = "player"
	resourceAwardPick        resourceType = "award_pick"
)

func handleDBError(
	err error,
	resource resourceType,
) error {
	// Handle SQL errors
	if err == sql.ErrNoRows {
		return translateSQLNoRowsError(err, resource)
	}

	// Handle postgres specific errors
	pqErr, ok := err.(*pq.Error)
	if !ok {
		return err
	}

	switch pqErr.Code {
	case foreignKeyViolation:
		return translateForeignKeyViolation(pqErr)
	case uniqueViolation:
		return translateUniqueViolation(pqErr)
	case checkConstraintViolation:
		return translateCheckConstraintViolation(pqErr)
	default:
		return buildErrorFromPQError(pqErr)
	}
}

func translateSQLNoRowsError(err error, resource resourceType) error {
	switch resource {
	case resourceUser:
		return domain.ErrUserNotFound
	case resourceSession:
		return domain.ErrSessionNotFound
	case resourceRefreshToken:
		return domain.ErrRefreshTokenNotFound
	case resourceBoard:
		return domain.ErrBoardNotFound
	case resourceBoardMember:
		return domain.ErrBoardMemberNotFound
	case resourceOAuthAccount:
		return domain.ErrOAuthAccountNotFound
	case resourceMatch:
		return domain.ErrMatchNotFound
	case resourceMatchAPIFixture:
		return domain.ErrMatchAPIFixtureNotFound
	case resourceCompetition:
		return domain.ErrCompetitionNotFound
	case resourcePlayer:
		return domain.ErrPlayerNotFound
	default:
		return err
	}
}

func translateForeignKeyViolation(pqErr *pq.Error) error {
	switch pqErr.Constraint {
	case
		"board_members_board_id_fkey",
		"board_members_user_id_fkey":
		return domain.ErrUserNotFound
	case "competitions_match_id_fkey":
		return domain.ErrMatchNotFound
	default:
		return buildErrorFromPQError(pqErr)
	}
}

func translateUniqueViolation(pqErr *pq.Error) error {
	switch pqErr.Constraint {
	case "users_username_key":
		return domain.ErrUsernameAlreadyExists
	case "users_email_key":
		return domain.ErrUserAlreadyExists
	case "boards_join_code_key":
		return domain.ErrBoardAlreadyExists
	case "board_members_pkey":
		return domain.ErrBoardMemberAlreadyInBoard
	case "idx_competitions_one_pickem_per_board":
		return domain.ErrCompetitionPickemAlreadyExists
	case "idx_competitions_one_pool_per_match":
		return domain.ErrDuplicatePoolForMatch
	case "competitions_board_id_name_key":
		return domain.ErrCompetitionNameAlreadyExists
	default:
		return buildErrorFromPQError(pqErr)
	}
}

func translateCheckConstraintViolation(pqErr *pq.Error) error {
	switch pqErr.Constraint {
	case "check_expires_at_after_created":
		return domain.ErrInvalidSessionExpiration
	case "check_last_used_before_expires":
		return domain.ErrInvalidSessionLastUsed
	case "check_winner_is_home_or_away":
		return domain.ErrInvalidWinnerTeam
	default:
		return buildErrorFromPQError(pqErr)
	}
}

func buildErrorFromPQError(pqErr *pq.Error) error {
	return fmt.Errorf("code: %s, constraint: %s, message: %s", pqErr.Code, pqErr.Constraint, pqErr.Message)
}
