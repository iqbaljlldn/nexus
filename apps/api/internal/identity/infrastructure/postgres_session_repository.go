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
	q Querier
}

func NewPostgresSessionRepository(q Querier) domain.SessionRepository {
	return &PostgresSessionRepository{q: q}
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

	dbSession, err := r.q.CreateSession(ctx, params)
	if err != nil {
		return err
	}

	session.ID = dbSession.ID.String()
	session.CreatedAt = dbSession.CreatedAt
	// Note: Status is handled by DB default (e.g. 'active')

	return nil
}
