package audit_test

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest/observer"

	"github.com/IltonSeixas/go-enterprise-boilerplate/internal/domain/entity"
	"github.com/IltonSeixas/go-enterprise-boilerplate/internal/infrastructure/audit"
)

func TestMemoryAuditLog_RecordLogsEvent(t *testing.T) {
	core, logs := observer.New(zap.InfoLevel)
	a := audit.NewMemoryAuditLog(zap.New(core))

	event := entity.NewAuditEvent(
		entity.AuditEventLoginSucceeded,
		uuid.NullUUID{UUID: uuid.New(), Valid: true},
		uuid.NullUUID{},
		"login succeeded",
	)

	a.Record(context.Background(), event)

	require.Equal(t, 1, logs.Len())
	entry := logs.All()[0]
	assert.Equal(t, "audit event recorded", entry.Message)
	assert.Equal(t, string(entity.AuditEventLoginSucceeded), entry.ContextMap()["event_type"])
}
