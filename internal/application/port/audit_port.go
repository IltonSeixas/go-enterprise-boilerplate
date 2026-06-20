package port

import (
	"context"

	"github.com/IltonSeixas/go-enterprise-boilerplate/internal/domain/entity"
)

// AuditPort implementations must never fail the use case they observe — on
// any underlying error, log and degrade gracefully rather than returning an error.
type AuditPort interface {
	Record(ctx context.Context, event entity.AuditEvent)
}
