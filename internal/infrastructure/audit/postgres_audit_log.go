package audit

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"

	"github.com/IltonSeixas/go-enterprise-boilerplate/internal/application/port"
	"github.com/IltonSeixas/go-enterprise-boilerplate/internal/domain/entity"
)

type PostgresAuditLog struct {
	pool   *pgxpool.Pool
	logger *zap.Logger
}

func NewPostgresAuditLog(pool *pgxpool.Pool, logger *zap.Logger) *PostgresAuditLog {
	return &PostgresAuditLog{pool: pool, logger: logger}
}

func (a *PostgresAuditLog) Record(ctx context.Context, event entity.AuditEvent) {
	_, err := a.pool.Exec(ctx, `
		INSERT INTO audit_log (id, event_type, actor_id, target_id, detail, occurred_at)
		VALUES ($1, $2, $3, $4, $5, $6)`,
		event.ID(), string(event.EventType()), event.ActorID(), event.TargetID(),
		event.Detail(), event.OccurredAt(),
	)
	if err != nil {
		a.logger.Error("failed to persist audit event",
			zap.Error(err),
			zap.String("audit_event_id", event.ID().String()),
		)
	}
}

var _ port.AuditPort = (*PostgresAuditLog)(nil)
