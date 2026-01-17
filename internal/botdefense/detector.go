package botdefense

import (
	"net/http"
	"strings"
)

// known bot user-agent patterns (case-insensitive matching)
var botPatterns = []string{
	// generic bot indicators
	"bot",
	"crawler",
	"spider",
	"scraper",
	"fetch",
	"scan",
	// cli tools
	"curl",
	"wget",
	"httpie",
	"lynx",
	"links",
	// programming libraries
	"python-requests",
	"python-urllib",
	"go-http-client",
	"java",
	"ruby",
	"perl",
	"php",
	"node-fetch",
	"axios",
	"libwww",
	"apache-httpclient",
	"okhttp",
	// headless browsers (when exposed)
	"headless",
	"phantomjs",
	"selenium",
	"puppeteer",
	"playwright",
	// specific scrapers
	"scrapy",
	"nutch",
	"heritrix",
	"httrack",
	"mass-downloader",
}

// legitimate browser indicators
var browserIndicators = []string{
	"mozilla",
	"chrome",
	"safari",
	"firefox",
	"edge",
	"opera",
}

// contains detected bot indicators
type BotSignals struct {
	EmptyUserAgent    bool
	ShortUserAgent    bool
	BotPatternMatch   string
	MissingHeaders    []string
	SuspiciousHeaders []string
	Score             int
}

// analyzes a request for bot indicators
// returns signals and a score (higher = more likely bot)
func DetectBot(r *http.Request) *BotSignals {
	signals := &BotSignals{}
	userAgent := r.Header.Get("User-Agent")
	userAgentLower := strings.ToLower(userAgent)

	// check user-agent
	if userAgent == "" {
		signals.EmptyUserAgent = true
		signals.Score += 50
	} else if len(userAgent) < 20 {
		signals.ShortUserAgent = true
		signals.Score += 30
	}

	// check for bot patterns in user-agent
	for _, pattern := range botPatterns {
		if strings.Contains(userAgentLower, pattern) {
			signals.BotPatternMatch = pattern
			signals.Score += 40
			break
		}
	}

	// check for missing typical browser headers
	missingHeaders := []string{}

	if r.Header.Get("Accept-Language") == "" {
		missingHeaders = append(missingHeaders, "Accept-Language")
		signals.Score += 10
	}

	if r.Header.Get("Accept-Encoding") == "" {
		missingHeaders = append(missingHeaders, "Accept-Encoding")
		signals.Score += 10
	}

	if r.Header.Get("Accept") == "" {
		missingHeaders = append(missingHeaders, "Accept")
		signals.Score += 10
	}

	signals.MissingHeaders = missingHeaders

	// suspicious header combinations
	suspiciousHeaders := []string{}

	// connection: close is often used by scripts
	if r.Header.Get("Connection") == "close" && !hasBrowserIndicator(userAgentLower) {
		suspiciousHeaders = append(suspiciousHeaders, "Connection: close without browser UA")
		signals.Score += 15
	}

	// no referrer on non-entry pages might indicate direct access
	// (but this is too aggressive for API calls, so we skip it)

	signals.SuspiciousHeaders = suspiciousHeaders

	// reduce score if it looks like a real browser
	if hasBrowserIndicator(userAgentLower) && len(missingHeaders) == 0 {
		signals.Score -= 20
		if signals.Score < 0 {
			signals.Score = 0
		}
	}

	return signals
}

// checks if the user-agent contains browser indicators
func hasBrowserIndicator(userAgentLower string) bool {
	for _, indicator := range browserIndicators {
		if strings.Contains(userAgentLower, indicator) {
			return true
		}
	}
	return false
}

// checks if the request path looks like probing
func IsSuspiciousPath(path string) bool {
	pathLower := strings.ToLower(path)

	suspiciousPatterns := []string{
		".php",
		".asp",
		".aspx",
		".jsp",
		".cgi",
		"..%2f", // path traversal
		"../",
		"%00", // null byte
		"<script",
		"union+select",
		"' or '",
	}

	for _, pattern := range suspiciousPatterns {
		if strings.Contains(pathLower, pattern) {
			return true
		}
	}

	return false
}
