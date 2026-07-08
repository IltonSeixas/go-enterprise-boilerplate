package middleware

import (
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/time/rate"
)

// entryTTL controls how long an idle key's limiter is retained before eviction.
const entryTTL = 10 * time.Minute

// cleanupInterval controls how often stale entries are swept from the map.
const cleanupInterval = time.Minute

type limiterEntry struct {
	limiter  *rate.Limiter
	lastSeen time.Time
}

// keyedLimiter is a per-key token bucket limiter — the key is caller-supplied
// (client IP for the public auth endpoints, authenticated user id for
// sensitive account actions), so this single implementation backs both
// RateLimit and UserRateLimit below.
type keyedLimiter struct {
	mu       sync.Mutex
	limiters map[string]*limiterEntry
	r        rate.Limit
	b        int
}

func newKeyedLimiter(r rate.Limit, b int) *keyedLimiter {
	l := &keyedLimiter{
		limiters: make(map[string]*limiterEntry),
		r:        r,
		b:        b,
	}
	go l.cleanupLoop()
	return l
}

func (l *keyedLimiter) get(key string) *rate.Limiter {
	l.mu.Lock()
	defer l.mu.Unlock()
	entry, ok := l.limiters[key]
	if !ok {
		entry = &limiterEntry{limiter: rate.NewLimiter(l.r, l.b)}
		l.limiters[key] = entry
	}
	entry.lastSeen = time.Now()
	return entry.limiter
}

func (l *keyedLimiter) cleanupLoop() {
	ticker := time.NewTicker(cleanupInterval)
	defer ticker.Stop()
	for range ticker.C {
		cutoff := time.Now().Add(-entryTTL)
		l.mu.Lock()
		for key, entry := range l.limiters {
			if entry.lastSeen.Before(cutoff) {
				delete(l.limiters, key)
			}
		}
		l.mu.Unlock()
	}
}

func tooManyRequests(c *gin.Context) {
	c.Header("Retry-After", time.Now().Add(time.Second).Format(time.RFC1123))
	c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{"error": "rate limit exceeded"})
}

// RateLimit returns a middleware that allows r requests per second with burst b per IP.
func RateLimit(r rate.Limit, b int) gin.HandlerFunc {
	limiter := newKeyedLimiter(r, b)
	return func(c *gin.Context) {
		if !limiter.get(c.ClientIP()).Allow() {
			tooManyRequests(c)
			return
		}
		c.Next()
	}
}

// UserRateLimit returns a middleware that allows r requests per second with
// burst b per authenticated user id, for sensitive account actions
// (password change, role change) where a valid access token already rules
// out anonymous brute force but a holder of a (possibly stolen) token could
// otherwise hammer the endpoint without limit — either to brute-force the
// current password before the token expires, or to force repeated expensive
// password-hash verification as a cost-amplification/DoS vector.
//
// Must run after RequireAuth: it reads the authenticated claims set by that
// middleware and falls through to it (returning 401, not 429) if no claims
// are present, since it has no per-user key to limit on in that case.
func UserRateLimit(r rate.Limit, b int) gin.HandlerFunc {
	limiter := newKeyedLimiter(r, b)
	return func(c *gin.Context) {
		claims, ok := GetClaims(c)
		if !ok {
			c.Next()
			return
		}
		if !limiter.get(claims.UserID.String()).Allow() {
			tooManyRequests(c)
			return
		}
		c.Next()
	}
}
