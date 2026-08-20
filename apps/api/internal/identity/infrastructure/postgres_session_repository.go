package infrastructure

import (
	"context"
	"database/sql"
	"net"
	"time"

	"github.com/google/uuid"
	"github.com/iqbaljlldn/nexus/apps/api/internal/identity/domain"
	"github.com/iqbaljlldn/nexus/pkg/contextutil"
	"github.com/sqlc-dev/pqtype"
)

type PostgresSessionRepository struct {
	db *sql.DB
}

func NewPostgresSessionRepository(db *sql.DB) domain.SessionRepository {
	return &PostgresSessionRepository{db: db}
}

func (r *PostgresSessionRepository) Create(ctx context.Context, session *domain.Session) error {
	userID, err := uuid.Parse(session.UserID)
	if err != nil {
		return err
	}

	ip := net.ParseIP(session.IPAddress)
	var inet pqtype.Inet
	if ip != nil {
		// Just passing the IP, assuming it's a single host (/32 or /128)
		if v4 := ip.To4(); v4 != nil {
			inet.IPNet = net.IPNet{IP: v4, Mask: net.CIDRMask(32, 32)}
		} else {
			inet.IPNet = net.IPNet{IP: ip, Mask: net.CIDRMask(128, 128)}
		}
		inet.Valid = true
	}

	params := CreateSessionParams{
		UserID:           userID,
		RefreshTokenHash: session.RefreshTokenHash,
		UserAgent: sql.NullString{
			String: session.UserAgent,
			Valid:  session.UserAgent != "",
		},
		IpAddress: inet,
		ExpiresAt: session.ExpiresAt,
	}

	dbSession, err := New(r.db).CreateSession(ctx, params)
	if err != nil {
		return err
	}

	session.ID = dbSession.ID.String()
	session.CreatedAt = dbSession.CreatedAt
	// Note: Status is handled by DB default (e.g. 'active')

	return nil
}

func (r *PostgresSessionRepository) RotateRefreshToken(ctx context.Context, oldTokenHash, newTokenHash string) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() {
		if rollbackErr := tx.Rollback(); rollbackErr != nil {
			err = rollbackErr
		}
	}()

	q := New(tx)

	// 1. Revoke the old session
	oldSession, err := q.RevokeSession(ctx, oldTokenHash)
	if err != nil {
		return err
	}

	// Create the new session based on the old one
	params := CreateSessionParams{
		UserID:           oldSession.UserID,
		RefreshTokenHash: newTokenHash,
		UserAgent:        oldSession.UserAgent,
		IpAddress:        oldSession.IpAddress,
		ExpiresAt:        oldSession.ExpiresAt, // Keep original expiration or extend it? Usually refresh tokens keep same absolute expiration or extend. Let's just extend by 24h as a standard.
	}

	_, err = q.CreateSession(ctx, params)
	if err != nil {
		return err
	}

	return tx.Commit()
}

func (r *PostgresSessionRepository) FindByRefreshToken(ctx context.Context, refreshToken string) (*domain.Session, error) {
	dbSession, err := New(r.db).FindSessionByTokenHash(ctx, refreshToken)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, domain.ErrInvalidToken
		}
		return nil, err
	}

	if dbSession.ExpiresAt.Before(time.Now()) {
		return nil, domain.ErrInvalidToken
	}

	if !dbSession.UserAgent.Valid {
		return nil, domain.ErrInvalidToken
	}

	if !dbSession.IpAddress.Valid {
		return nil, domain.ErrInvalidToken
	}

	return &domain.Session{
		ID:               dbSession.ID.String(),
		UserID:           dbSession.UserID.String(),
		RefreshTokenHash: dbSession.RefreshTokenHash,
		UserAgent:        dbSession.UserAgent.String,
		IPAddress:        dbSession.IpAddress.IPNet.String(),
		ExpiresAt:        dbSession.ExpiresAt,
		CreatedAt:        dbSession.CreatedAt,
	}, nil
}

func (r *PostgresSessionRepository) RevokeSession(ctx context.Context, refreshToken string) error {
	_, err := New(r.db).RevokeSession(ctx, refreshToken)
	if err != nil {
		if err == sql.ErrNoRows {
			return domain.ErrInvalidToken
		}
		return err
	}

	return nil
}

func (r *PostgresSessionRepository) RevokeAllSessions(ctx context.Context) error {
	userID, err := contextutil.UserID(ctx)
	if err != nil {
		return err
	}
	if err := New(r.db).RevokeAllSessionsByUserId(ctx, userID); err != nil {
		if err == sql.ErrNoRows {
			return nil
		}
		return err
	}

	return nil
}

func (r *PostgresSessionRepository) GetActiveSessions(ctx context.Context) ([]domain.Session, error) {
	userID, err := contextutil.UserID(ctx)
	if err != nil {
		return nil, err
	}
	dbSession, err := New(r.db).FindActiveSessionsByUserId(ctx, userID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	sessions := make([]domain.Session, len(dbSession))
	for i, session := range dbSession {
		sessions[i] = domain.Session{
			ID:               session.ID.String(),
			UserID:           session.UserID.String(),
			RefreshTokenHash: session.RefreshTokenHash,
			UserAgent:        session.UserAgent.String,
			IPAddress:        session.IpAddress.IPNet.String(),
			ExpiresAt:        session.ExpiresAt,
			CreatedAt:        session.CreatedAt,
		}
	}

	return sessions, nil
}

func (r *PostgresSessionRepository) RevokeSessionById(ctx context.Context, id uuid.UUID) (*domain.Session, error) {
	dbSession, err := New(r.db).RevokeSessionById(ctx, id)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, domain.ErrInvalidToken
		}
		return nil, err
	}

	if !dbSession.UserAgent.Valid || !dbSession.IpAddress.Valid || dbSession.ExpiresAt.Before(time.Now()) {
		return nil, domain.ErrInvalidToken
	}

	return &domain.Session{
		ID:               dbSession.ID.String(),
		UserID:           dbSession.UserID.String(),
		RefreshTokenHash: dbSession.RefreshTokenHash,
		UserAgent:        dbSession.UserAgent.String,
		IPAddress:        dbSession.IpAddress.IPNet.String(),
		ExpiresAt:        dbSession.ExpiresAt,
		CreatedAt:        dbSession.CreatedAt,
	}, nil
}

func (r *PostgresSessionRepository) FindById(ctx context.Context, id uuid.UUID) (*domain.Session, error) {
	dbSession, err := New(r.db).FindSessionById(ctx, id)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, domain.ErrInvalidToken
		}
		return nil, err
	}

	return &domain.Session{
		ID:               dbSession.ID.String(),
		UserID:           dbSession.UserID.String(),
		RefreshTokenHash: dbSession.RefreshTokenHash,
		UserAgent:        dbSession.UserAgent.String,
		IPAddress:        dbSession.IpAddress.IPNet.String(),
		ExpiresAt:        dbSession.ExpiresAt,
		CreatedAt:        dbSession.CreatedAt,
	}, nil
}
