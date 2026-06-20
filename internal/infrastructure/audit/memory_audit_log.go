package audit

import (
	"context"

	"go.uber.org/zap"

	"github.com/IltonSeixas/go-enterprise-boilerplate/internal/application/port"
	"github.com/IltonSeixas/go-enterprise-boilerplate/internal/domain/entity"
)

type MemoryAuditLog struct {
	logger *zap.Logger
}

func NewMemoryAuditLog(logger *zap.Logger) *MemoryAuditLog {
	return &MemoryAuditLog{logger: logger}
}

func (a *MemoryAuditLog) Record(_ context.Context, event entity.AuditEvent) {
	a.logger.Info("audit event recorded",
		zap.String("audit_event_id", event.ID().String()),
		zap.String("event_type", string(event.EventType())),
		zap.Any("actor_id", event.ActorID()),
		zap.Any("target_id", event.TargetID()),
		zap.String("detail", event.Detail()),
		zap.Time("occurred_at", event.OccurredAt()),
	)
}

var _ port.AuditPort = (*MemoryAuditLog)(nil)
