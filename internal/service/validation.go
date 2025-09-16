package service

import (
	"regexp"
	"strings"

	"github.com/maxviazov/basketball-stats-service/internal/repository"
)

const (
	defaultLimit = 50
	maxLimit     = 100
)

var seasonRe = regexp.MustCompile(`^\d{4}-\d{2}$`)

func normalizePage(p repository.Page) repository.Page {
	limit := p.Limit
	offset := p.Offset
	if limit <= 0 {
		limit = defaultLimit
	} else if limit > maxLimit {
		limit = maxLimit
	}
	if offset < 0 {
		offset = 0
	}
	return repository.Page{Limit: limit, Offset: offset}
}

func normalizePosition(pos string) string {
	return strings.ToUpper(strings.TrimSpace(pos))
}

func isValidPosition(pos string) bool {
	switch normalizePosition(pos) {
	case "PG", "SG", "SF", "PF", "C":
		return true
	default:
		return false
	}
}

func normalizeStatus(status string) string { return strings.ToLower(strings.TrimSpace(status)) }

func isValidGameStatus(status string) bool {
	switch normalizeStatus(status) {
	case "scheduled", "in_progress", "finished":
		return true
	default:
		return false
	}
}

// IsValidSeason checks if the season string conforms to the YYYY-YY format.
func IsValidSeason(season string) bool {
	s := strings.TrimSpace(season)
	return seasonRe.MatchString(s)
}
