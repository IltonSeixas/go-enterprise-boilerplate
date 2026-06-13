package middleware

import (
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/time/rate"
)

// entryTTL controls how long an idle IP's limiter is retained before eviction.
const entryTTL = 10 * time.Minute

// cleanupInterval controls how often stale entries are swept from the map.
const cleanupInterval = time.Minute

type limiterEntry struct {
	limiter  *rate.Limiter
	lastSeen time.Time
}

type ipLimiter struct {
	mu       sync.Mutex
	limiters map[string]*limiterEntry
	r        rate.Limit
	b        int
}

func newIPLimiter(r rate.Limit, b int) *ipLimiter {
	l := &ipLimiter{
		limiters: make(map[string]*limiterEntry),
		r:        r,
		b:        b,
	}
	go l.cleanupLoop()
	return l
}

func (l *ipLimiter) get(ip string) *rate.Limiter {
	l.mu.Lock()
	defer l.mu.Unlock()
	entry, ok := l.limiters[ip]
	if !ok {
		entry = &limiterEntry{limiter: rate.NewLimiter(l.r, l.b)}
		l.limiters[ip] = entry
	}
	entry.lastSeen = time.Now()
	return entry.limiter
}

func (l *ipLimiter) cleanupLoop() {
	ticker := time.NewTicker(cleanupInterval)
	defer ticker.Stop()
	for range ticker.C {
		cutoff := time.Now().Add(-entryTTL)
		l.mu.Lock()
		for ip, entry := range l.limiters {
			if entry.lastSeen.Before(cutoff) {
				delete(l.limiters, ip)
			}
		}
		l.mu.Unlock()
	}
}

// RateLimit returns a middleware that allows r requests per second with burst b per IP.
func RateLimit(r rate.Limit, b int) gin.HandlerFunc {
	limiter := newIPLimiter(r, b)
	return func(c *gin.Context) {
		if !limiter.get(c.ClientIP()).Allow() {
			c.Header("Retry-After", time.Now().Add(time.Second).Format(time.RFC1123))
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{"error": "rate limit exceeded"})
			return
		}
		c.Next()
	}
}
