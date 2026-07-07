package entity

import (
	"time"

	"github.com/google/uuid"
)

type AuditEventType string

const (
	AuditEventUserRegistered            AuditEventType = "user_registered"
	AuditEventLoginSucceeded            AuditEventType = "login_succeeded"
	AuditEventLoginFailed               AuditEventType = "login_failed"
	AuditEventPasswordChanged           AuditEventType = "password_changed"
	AuditEventRoleChanged               AuditEventType = "role_changed"
	AuditEventTokenRefreshed            AuditEventType = "token_refreshed"
	AuditEventRefreshTokenReuseDetected AuditEventType = "refresh_token_reuse_detected"
)

type AuditEvent struct {
	id         uuid.UUID
	eventType  AuditEventType
	actorID    uuid.NullUUID
	targetID   uuid.NullUUID
	detail     string
	occurredAt time.Time
}

func NewAuditEvent(
	eventType AuditEventType,
	actorID, targetID uuid.NullUUID,
	detail string,
) AuditEvent {
	return AuditEvent{
		id:         uuid.New(),
		eventType:  eventType,
		actorID:    actorID,
		targetID:   targetID,
		detail:     detail,
		occurredAt: time.Now().UTC(),
	}
}

func ReconstituteAuditEvent(
	id uuid.UUID,
	eventType AuditEventType,
	actorID, targetID uuid.NullUUID,
	detail string,
	occurredAt time.Time,
) AuditEvent {
	return AuditEvent{
		id:         id,
		eventType:  eventType,
		actorID:    actorID,
		targetID:   targetID,
		detail:     detail,
		occurredAt: occurredAt,
	}
}

func (e AuditEvent) ID() uuid.UUID             { return e.id }
func (e AuditEvent) EventType() AuditEventType { return e.eventType }
func (e AuditEvent) ActorID() uuid.NullUUID    { return e.actorID }
func (e AuditEvent) TargetID() uuid.NullUUID   { return e.targetID }
func (e AuditEvent) Detail() string            { return e.detail }
func (e AuditEvent) OccurredAt() time.Time     { return e.occurredAt }
