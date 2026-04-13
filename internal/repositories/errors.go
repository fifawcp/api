package repositories

import (
	"database/sql"
	"fmt"

	"github.com/lib/pq"
	"github.com/ncondes/fifawcp/internal/domain"
)

type resourceType string

const (
	foreignKeyViolation      = "23503"
	uniqueViolation          = "23505"
	checkConstraintViolation = "23514"
)

const (
	resourceUser         resourceType = "user"
	resourceSession      resourceType = "session"
	resourceRefreshToken resourceType = "refresh_token"
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
	default:
		return err
	}
}

func translateForeignKeyViolation(pqErr *pq.Error) error {
	switch pqErr.Constraint {
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
	default:
		return buildErrorFromPQError(pqErr)
	}
}

func buildErrorFromPQError(pqErr *pq.Error) error {
	return fmt.Errorf("code: %s, constraint: %s, message: %s", pqErr.Code, pqErr.Constraint, pqErr.Message)
}
