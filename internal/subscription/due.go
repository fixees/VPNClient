package subscription

import (
	"strings"
	"time"

	"myinternetvpn/client/internal/profiles"
)

// Due reports whether a profile subscription should be refreshed.
func Due(p profiles.Profile, globalIntervalMin int, now time.Time) bool {
	if strings.TrimSpace(p.SubscriptionURL) == "" {
		return false
	}
	if p.LastSyncAt == "" {
		return true
	}
	last, err := time.Parse(time.RFC3339, p.LastSyncAt)
	if err != nil {
		return true
	}
	var wait time.Duration
	if p.UpdateIntervalHours > 0 {
		wait = time.Duration(p.UpdateIntervalHours) * time.Hour
	} else if globalIntervalMin > 0 {
		wait = time.Duration(globalIntervalMin) * time.Minute
	} else {
		wait = 6 * time.Hour
	}
	return !last.Add(wait).After(now)
}
