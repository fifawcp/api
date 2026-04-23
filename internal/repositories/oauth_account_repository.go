package repositories

import (
	"context"
	"database/sql"

	"github.com/fifawcp/api/internal/domain"
	"github.com/fifawcp/api/internal/infrastructure/config"
)

type OAuthAccountRepository struct {
	db  *sql.DB
	cfg *config.Config
}

func NewOAuthAccountRepository(
	db *sql.DB,
	cfg *config.Config,
) *OAuthAccountRepository {
	return &OAuthAccountRepository{
		db:  db,
		cfg: cfg,
	}
}

func (r *OAuthAccountRepository) CreateOAuthAccount(
	ctx context.Context,
	oauthAccount *domain.OAuthAccount,
) error {
	ctx, cancel := context.WithTimeout(ctx, r.cfg.DB.QueryTimeout)
	defer cancel()

	query := `INSERT INTO oauth_accounts (
		provider,
		provider_sub,
		user_id
	) VALUES ($1, $2, $3)
	RETURNING id, created_at`

	err := r.db.QueryRowContext(
		ctx,
		query,
		oauthAccount.Provider,
		oauthAccount.ProviderSub,
		oauthAccount.UserID,
	).Scan(
		&oauthAccount.ID,
		&oauthAccount.CreatedAt,
	)
	if err != nil {
		return handleDBError(err, resourceOAuthAccount)
	}

	return nil
}

func (r *OAuthAccountRepository) CreateUserWithOAuthAccount(
	ctx context.Context,
	user *domain.User,
	account *domain.OAuthAccount,
) error {
	ctx, cancel := context.WithTimeout(ctx, r.cfg.DB.QueryTimeout)
	defer cancel()

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	userQuery := `INSERT INTO users (
		first_name,
		last_name,
		username,
		email
	) VALUES ($1, $2, $3, $4)
	RETURNING id, created_at, updated_at`

	err = tx.QueryRowContext(ctx, userQuery,
		user.FirstName,
		user.LastName,
		user.Username,
		user.Email,
	).Scan(&user.ID, &user.CreatedAt, &user.UpdatedAt)
	if err != nil {
		return handleDBError(err, resourceUser)
	}

	accountQuery := `INSERT INTO oauth_accounts (
		provider,
		provider_sub,
		user_id
	) VALUES ($1, $2, $3)
	RETURNING id, created_at`

	err = tx.QueryRowContext(ctx, accountQuery,
		account.Provider,
		account.ProviderSub,
		user.ID,
	).Scan(&account.ID, &account.CreatedAt)
	if err != nil {
		return handleDBError(err, resourceOAuthAccount)
	}

	account.UserID = user.ID

	return tx.Commit()
}

func (r *OAuthAccountRepository) GetByProviderSub(
	ctx context.Context,
	provider string,
	providerSub string,
) (*domain.OAuthAccount, error) {
	ctx, cancel := context.WithTimeout(ctx, r.cfg.DB.QueryTimeout)
	defer cancel()

	query := `SELECT
		id,
		provider,
		provider_sub,
		user_id,
		created_at
	FROM oauth_accounts
	WHERE provider = $1 AND provider_sub = $2`

	var oauthAccount domain.OAuthAccount
	err := r.db.QueryRowContext(ctx, query, provider, providerSub).Scan(
		&oauthAccount.ID,
		&oauthAccount.Provider,
		&oauthAccount.ProviderSub,
		&oauthAccount.UserID,
		&oauthAccount.CreatedAt,
	)
	if err != nil {
		return nil, handleDBError(err, resourceOAuthAccount)
	}

	return &oauthAccount, nil
}
