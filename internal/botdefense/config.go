package botdefense

import (
	"strings"
	"time"
)

// holds bot defense configuration
type Config struct {
	// whether bot defense is active
	Enabled bool

	// max requests per window before triggering
	RateLimit int

	// time window for rate limiting
	RateLimitWindow time.Duration

	// how long an IP stays trapped
	TrapTTL time.Duration

	// how long to slow-drip responses
	TarpitDuration time.Duration

	// delay between each byte sent during tarpitting
	TarpitChunkDelay time.Duration

	// paths that only bots would access
	HoneypotPaths []string

	// domains for reverse DNS verification
	VerifiedCrawlerDomains []string

	// paths that bypass bot defense (health checks, etc.)
	ExemptPaths []string
}

// returns sensible defaults
func DefaultConfig() *Config {
	return &Config{
		Enabled:          true,
		RateLimit:        200,
		RateLimitWindow:  time.Minute,
		TrapTTL:          24 * time.Hour,
		TarpitDuration:   60 * time.Second,
		TarpitChunkDelay: time.Second,
		HoneypotPaths: []string{
			// wordpress
			"/wp-admin",
			"/wp-login.php",
			"/wp-content",
			"/wp-includes",
			"/xmlrpc.php",

			// config/secrets
			"/.env",
			"/.git",
			"/.git/config",
			"/.gitignore",
			"/config.php",
			"/config.json",
			"/config.yml",
			"/secrets.json",
			"/.aws/credentials",

			// admin panels
			"/admin",
			"/admin.php",
			"/administrator",
			"/phpmyadmin",
			"/cpanel",

			// backups
			"/backup",
			"/backup.zip",
			"/backup.sql",
			"/db.sql",
			"/database.sql",

			// debug/internal
			"/debug",
			"/trace",
			"/server-status",
			"/server-info",
			"/.htaccess",
			"/.htpasswd",

			// api probing
			"/api/internal",
			"/api/admin",
			"/api/debug",
			"/api/v1/internal",

			// algojams-specific honeypots
			"/api/v1/strudels/export-all",
			"/api/v1/users/dump",
			"/api/v1/sessions/all",
		},
		VerifiedCrawlerDomains: []string{
			// google
			"googlebot.com",
			"google.com",

			// microsoft/bing
			"search.msn.com",
			"bing.com",

			// anthropic
			"anthropic.com",

			// openai
			"openai.com",

			// claude
			"anthropic.com",

			// apple
			"applebot.apple.com",

			// yandex
			"yandex.ru",
			"yandex.net",
			"yandex.com",

			// baidu
			"baidu.com",
			"baidu.jp",

			// duckduckgo
			"duckduckgo.com",

			// facebook
			"facebook.com",
			"fbsv.net",

			// twitter
			"twitter.com",
			"twitterbot.com",

			// linkedin
			"linkedin.com",

			// pinterest
			"pinterest.com",
		},
		ExemptPaths: []string{
			"/health",
			"/healthz",
			"/ready",
			"/metrics",
			"/api/v1/ws", // websocket connections are persistent, not burst requests
		},
	}
}

// checks if a path is a honeypot (prefix match)
func (c *Config) IsHoneypotPath(path string) bool {
	for _, hp := range c.HoneypotPaths {
		if path == hp || strings.HasPrefix(path, hp+"/") {
			return true
		}
	}

	return false
}

// checks if a path bypasses bot defense
func (c *Config) IsExemptPath(path string) bool {
	for _, ep := range c.ExemptPaths {
		if path == ep || strings.HasPrefix(path, ep+"/") {
			return true
		}
	}
	return false
}
