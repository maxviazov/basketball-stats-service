package service

import (
	"strings"

	"github.com/maxviazov/basketball-stats-service/internal/repository"
)

func normalizePage(p repository.Page) repository.Page {
	limit := p.Limit
	offset := p.Offset
	if limit <= 0 {
		limit = 50
	}
	if offset < 0 {
		offset = 0
	}
	return repository.Page{Limit: limit, Offset: offset}
}

func isValidPosition(pos string) bool {
	s := strings.ToUpper(strings.TrimSpace(pos))
	switch s {
	case "PG", "SG", "SF", "PF", "C":
		return true
	default:
		return false
	}
}

func isValidGameStatus(status string) bool {
	s := strings.ToLower(strings.TrimSpace(status))
	switch s {
	case "scheduled", "in_progress", "finished":
		return true
	default:
		return false
	}
}
