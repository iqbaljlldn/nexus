package infrastructure

import (
	"context"
	"database/sql"
	"errors"

	"github.com/iqbaljlldn/nexus/apps/api/internal/identity/domain"
	"github.com/jackc/pgx/v5/pgconn"
)

type PostgresUserRepository struct {
	q Querier
}

func NewPostgresUserRepository(q Querier) domain.UserRepository {
	return &PostgresUserRepository{q: q}
}

func (r *PostgresUserRepository) Create(ctx context.Context, user *domain.User) error {
	params := CreateUserParams{
		ID:           user.ID,
		Email:        user.Email.String(),
		Username:     user.Username.String(),
		DisplayName:  user.DisplayName,
		PasswordHash: user.PasswordHash.String(),
		CreatedAt:    user.CreatedAt,
		UpdatedAt:    user.UpdatedAt,
	}

	_, err := r.q.CreateUser(ctx, params)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			if pgErr.ConstraintName == "users_email_key" {
				return domain.ErrDuplicateEmail
			}
			if pgErr.ConstraintName == "users_username_key" {
				return domain.ErrDuplicateUsername
			}
		}
		return err
	}

	return nil
}

func (r *PostgresUserRepository) FindByEmail(ctx context.Context, email string) (*domain.User, error) {
	dbUser, err := r.q.FindUserByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrUserNotFound
		}
		return nil, err
	}

	domainEmail, _ := domain.NewEmail(dbUser.Email)
	domainUsername, _ := domain.NewUsername(dbUser.Username)
	passwordHash, _ := domain.NewPasswordHash(dbUser.PasswordHash)

	user := &domain.User{
		ID:           dbUser.ID,
		Email:        domainEmail,
		Username:     domainUsername,
		DisplayName:  dbUser.DisplayName,
		PasswordHash: passwordHash,
		IsSuspended:  dbUser.IsSuspended,
		IsBanned:     dbUser.IsBanned,
		CreatedAt:    dbUser.CreatedAt,
		UpdatedAt:    dbUser.UpdatedAt,
	}

	if dbUser.AvatarUrl.Valid {
		user.AvatarURL = &dbUser.AvatarUrl.String
	}

	if dbUser.DeletedAt.Valid {
		user.DeletedAt = &dbUser.DeletedAt.Time
	}

	return user, nil
}

func (r *PostgresUserRepository) FindByUsername(ctx context.Context, username string) (*domain.User, error) {
	dbUser, err := r.q.FindUserByUsername(ctx, username)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrUserNotFound
		}
		return nil, err
	}

	domainEmail, _ := domain.NewEmail(dbUser.Email)
	domainUsername, _ := domain.NewUsername(dbUser.Username)
	passwordHash, _ := domain.NewPasswordHash(dbUser.PasswordHash)

	user := &domain.User{
		ID:           dbUser.ID,
		Email:        domainEmail,
		Username:     domainUsername,
		DisplayName:  dbUser.DisplayName,
		PasswordHash: passwordHash,
		IsSuspended:  dbUser.IsSuspended,
		IsBanned:     dbUser.IsBanned,
		CreatedAt:    dbUser.CreatedAt,
		UpdatedAt:    dbUser.UpdatedAt,
	}

	if dbUser.AvatarUrl.Valid {
		user.AvatarURL = &dbUser.AvatarUrl.String
	}

	if dbUser.DeletedAt.Valid {
		user.DeletedAt = &dbUser.DeletedAt.Time
	}

	return user, nil
}
