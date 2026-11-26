package httpx

import (
	"net/http"
	"strconv"
	"strings"
	"time"
)

// ISVUserAgent builds the decorated User-Agent Microsoft recommends for
// third-party apps ("ISV|Company|App/1.0"); SharePoint Online gives decorated
// traffic better treatment when throttling. Pass it as Options.UserAgent.
func ISVUserAgent(company, app, version string) string {
	clean := func(s string) string { return strings.NewReplacer("|", "-", "/", "-").Replace(s) }
	return "ISV|" + clean(company) + "|" + clean(app) + "/" + version
}

// RateLimit holds the RateLimit-* headers SharePoint Online sends to warn
// before throttling kicks in.
type RateLimit struct {
	Limit     int
	Remaining int
	// Reset is how long until the window resets.
	Reset time.Duration
}

// RateLimit parses RateLimit-Limit, RateLimit-Remaining and RateLimit-Reset.
// ok is false when the response carries none of them.
func (r *Response) RateLimit() (rl RateLimit, ok bool) {
	return parseRateLimit(r.Header)
}

func parseRateLimit(h http.Header) (rl RateLimit, ok bool) {
	num := func(k string) (int, bool) {
		n, err := strconv.Atoi(strings.TrimSpace(h.Get(k)))
		return n, err == nil
	}
	var a, b, c bool
	rl.Limit, a = num("RateLimit-Limit")
	rl.Remaining, b = num("RateLimit-Remaining")
	var secs int
	secs, c = num("RateLimit-Reset")
	rl.Reset = time.Duration(secs) * time.Second
	return rl, a || b || c
}
