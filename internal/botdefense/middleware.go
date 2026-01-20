package botdefense

import (
	"context"
	"strings"
	"time"

	"codeberg.org/algojams/server/internal/logger"
	"github.com/gin-gonic/gin"
)

const (
	// minimum score to consider a request as bot-like
	BotScoreThreshold = 40
)

// orchestrates all bot defense components
type Defense struct {
	config   *Config
	store    *Store
	verifier *CrawlerVerifier
}

// creates a new bot defense system
func New(config *Config, store *Store) *Defense {
	return &Defense{
		config:   config,
		store:    store,
		verifier: NewCrawlerVerifier(config.VerifiedCrawlerDomains),
	}
}

// returns a Gin middleware that implements bot defense
func (d *Defense) Middleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		if !d.config.Enabled {
			c.Next()
			return
		}

		ctx := c.Request.Context()
		ip := c.ClientIP()
		path := c.Request.URL.Path

		// exempt paths bypass all checks
		if d.config.IsExemptPath(path) {
			c.Next()
			return
		}

		// check if path is a honeypot
		if d.config.IsHoneypotPath(path) {
			d.handleHoneypot(ctx, c, ip, path)
			return
		}

		// check if IP is already trapped
		trapped, reason, err := d.store.IsTrapped(ctx, ip)
		if err != nil {
			logger.ErrorErr(err, "failed to check trapped status", "ip", ip)
		} else if trapped {
			d.handleTrapped(c, ip, reason)
			return
		}

		// check rate limit
		count, err := d.store.IncrementRate(ctx, ip)
		if err != nil {
			logger.ErrorErr(err, "failed to increment rate", "ip", ip)
		} else if count > int64(d.config.RateLimit) {
			d.handleRateLimited(c, ip)
			return
		}

		// check if verified crawler (Google, Bing, Anthropic, etc.)
		userAgent := c.Request.Header.Get("User-Agent")
		if isCrawler, _ := MightBeKnownCrawler(userAgent); isCrawler {
			if d.verifier.IsVerifiedCrawler(ctx, ip) {
				logger.Debug("verified crawler allowed", "ip", ip, "user_agent", userAgent)
				c.Next()
				return
			}

			logger.Warn("unverified crawler claim", "ip", ip, "user_agent", userAgent)
			if err := d.store.TrapIP(ctx, ip, ReasonBotPattern); err != nil {
				logger.ErrorErr(err, "failed to trap IP", "ip", ip)
			}
			d.handleTrapped(c, ip, ReasonBotPattern)
			return
		}

		// check suspicious path patterns
		if IsSuspiciousPath(path) {
			logger.Warn("suspicious path accessed", "ip", ip, "path", path)
			if err := d.store.TrapIP(ctx, ip, ReasonBotPattern); err != nil {
				logger.ErrorErr(err, "failed to trap IP", "ip", ip)
			}
			d.handleTrapped(c, ip, ReasonBotPattern)
			return
		}

		// apply bot detection heuristics
		signals := DetectBot(c.Request)
		if signals.Score >= BotScoreThreshold {
			logger.Warn("bot-like request detected",
				"ip", ip,
				"path", path,
				"score", signals.Score,
				"pattern", signals.BotPatternMatch,
				"missing_headers", signals.MissingHeaders,
				"user_agent", c.Request.Header.Get("User-Agent"),
			)
			if err := d.store.TrapIP(ctx, ip, ReasonBotPattern); err != nil {
				logger.ErrorErr(err, "failed to trap IP", "ip", ip)
			}
			d.handleTrapped(c, ip, ReasonBotPattern)
			return
		} else if strings.HasPrefix(path, "/api/v1/public") {
			// debug logging for public API requests
			logger.Debug("public API request passed bot check",
				"ip", ip,
				"path", path,
				"score", signals.Score,
				"user_agent", c.Request.Header.Get("User-Agent"),
			)
		}

		c.Next()
	}
}

func (d *Defense) handleHoneypot(ctx context.Context, c *gin.Context, ip, path string) {
	logger.Warn("honeypot triggered", "ip", ip, "path", path)

	if err := d.store.TrapIP(ctx, ip, ReasonHoneypot); err != nil {
		logger.ErrorErr(err, "failed to trap IP", "ip", ip)
	}

	if cryptoRandInt(2) == 0 {
		ServePoisonedJSON(c)
	} else {
		Tarpit(c, d.config.TarpitDuration, d.config.TarpitChunkDelay)
	}

	c.Abort()
}

func (d *Defense) handleTrapped(c *gin.Context, ip string, reason TrapReason) {
	logger.Debug("trapped IP request blocked", "ip", ip, "reason", reason)

	switch cryptoRandInt(3) {
	case 0:
		Tarpit(c, d.config.TarpitDuration, d.config.TarpitChunkDelay)
	case 1:
		TarpitJSON(c, d.config.TarpitDuration, d.config.TarpitChunkDelay)
	default:
		ServePoisonedJSON(c)
	}
	c.Abort()
}

func (d *Defense) handleRateLimited(c *gin.Context, ip string) {
	logger.Warn("rate limit exceeded", "ip", ip)

	c.Header("Retry-After", "60")
	c.AbortWithStatusJSON(429, gin.H{
		"error":   "rate_limit_exceeded",
		"message": "too many requests. please slow down.",
	})
}

// starts a background goroutine to clean the crawler cache
func (d *Defense) StartCacheCleaner(ctx context.Context, interval time.Duration) {
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				d.verifier.CleanCache()
			}
		}
	}()
}
