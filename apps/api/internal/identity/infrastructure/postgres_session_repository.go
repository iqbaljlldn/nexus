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

func parseIP(ipStr string) pqtype.Inet {
	var inet pqtype.Inet
	ip := net.ParseIP(ipStr)
	if ip != nil {
		if v4 := ip.To4(); v4 != nil {
			inet.IPNet = net.IPNet{IP: v4, Mask: net.CIDRMask(32, 32)}
		} else {
			inet.IPNet = net.IPNet{IP: ip, Mask: net.CIDRMask(128, 128)}
		}
		inet.Valid = true
	}
	return inet
}

func (r *PostgresSessionRepository) Create(ctx context.Context, session *domain.Session) error {
	sessionID, err := uuid.Parse(session.ID)
	if err != nil {
		return err
	}

	userID, err := uuid.Parse(session.UserID)
	if err != nil {
		return err
	}

	params := CreateSessionParams{
		ID:               sessionID,
		UserID:           userID,
		RefreshTokenHash: session.RefreshTokenHash,
		UserAgent: sql.NullString{
			String: session.UserAgent,
			Valid:  session.UserAgent != "",
		},
		IpAddress: parseIP(session.IPAddress),
		ExpiresAt: session.ExpiresAt,
	}

	dbSession, err := New(r.db).CreateSession(ctx, params)
	if err != nil {
		return err
	}

	session.CreatedAt = dbSession.CreatedAt
	// Note: Status is handled by DB default (e.g. 'active')

	return nil
}

func (r *PostgresSessionRepository) RotateRefreshToken(ctx context.Context, oldSessionID, newSessionID, newTokenHash string) error {
	sessionUUID, err := uuid.Parse(oldSessionID)
	if err != nil {
		return err
	}

	newSessionUUID, err := uuid.Parse(newSessionID)
	if err != nil {
		return err
	}

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
	oldSession, err := q.RevokeSessionById(ctx, sessionUUID)
	if err != nil {
		return err
	}

	// Create the new session based on the old one
	params := CreateSessionParams{
		ID:               newSessionUUID,
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
