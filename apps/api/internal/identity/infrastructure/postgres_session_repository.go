package infrastructure

import (
	"context"
	"database/sql"
	"net"

	"github.com/google/uuid"
	"github.com/iqbaljlldn/nexus/apps/api/internal/identity/domain"
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
