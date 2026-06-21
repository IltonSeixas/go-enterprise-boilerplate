package handler

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
)

type Pinger interface {
	Ping(ctx context.Context) error
}

type RedisPinger struct {
	Client interface {
		Ping(ctx context.Context) *redis.StatusCmd
	}
}

func (p RedisPinger) Ping(ctx context.Context) error {
	return p.Client.Ping(ctx).Err()
}

type HealthHandler struct {
	redis    Pinger
	database Pinger
}

func NewHealthHandler(redis Pinger, database Pinger) *HealthHandler {
	return &HealthHandler{redis: redis, database: database}
}

func (h *HealthHandler) Health(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

func (h *HealthHandler) Ready(c *gin.Context) {
	checks := gin.H{}
	ready := true

	if err := h.redis.Ping(c.Request.Context()); err != nil {
		checks["redis"] = "error"
		ready = false
	} else {
		checks["redis"] = "ok"
	}

	if h.database != nil {
		if err := h.database.Ping(c.Request.Context()); err != nil {
			checks["postgres"] = "error"
			ready = false
		} else {
			checks["postgres"] = "ok"
		}
	}

	status := http.StatusOK
	statusText := "ready"
	if !ready {
		status = http.StatusServiceUnavailable
		statusText = "not_ready"
	}

	c.JSON(status, gin.H{"status": statusText, "checks": checks})
}
