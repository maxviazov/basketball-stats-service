// Package model contains domain entities and DTOs used across layers.
// I keep it lean and focused on data shapes without behavior.
package model

import "time"

// Team represents a basketball team.
type Team struct {
	ID        int64     `json:"id"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// Player represents an athlete belonging to a team.
type Player struct {
	ID        int64     `json:"id"`
	TeamID    int64     `json:"team_id"`
	FirstName string    `json:"first_name"`
	LastName  string    `json:"last_name"`
	Position  string    `json:"position"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// Game represents a scheduled or finished match.
type Game struct {
	ID         int64     `json:"id"`
	Season     string    `json:"season"`
	Date       time.Time `json:"date"`
	HomeTeamID int64     `json:"home_team_id"`
	AwayTeamID int64     `json:"away_team_id"`
	Status     string    `json:"status"` // scheduled, in_progress, finished
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

// PlayerStatLine represents per-game stats for a player.
type PlayerStatLine struct {
	ID            int64     `json:"id"`
	PlayerID      int64     `json:"player_id"`
	GameID        int64     `json:"game_id"`
	Points        int       `json:"points"`
	Rebounds      int       `json:"rebounds"`
	Assists       int       `json:"assists"`
	Steals        int       `json:"steals"`
	Blocks        int       `json:"blocks"`
	Fouls         int       `json:"fouls"`
	Turnovers     int       `json:"turnovers"`
	MinutesPlayed float32   `json:"minutes_played"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

// PlayerAggregatedStats holds calculated statistics for a player, such as career totals or seasonal averages.
// This model is designed for read-only query results and is not persisted directly.
type PlayerAggregatedStats struct {
	GamesPlayed   int     `json:"games_played"`
	TotalPoints   int     `json:"total_points"`
	TotalRebounds int     `json:"total_rebounds"`
	TotalAssists  int     `json:"total_assists"`
	TotalSteals   int     `json:"total_steals"`
	TotalBlocks   int     `json:"total_blocks"`
	AvgPoints     float32 `json:"avg_points"`
	AvgRebounds   float32 `json:"avg_rebounds"`
	AvgAssists    float32 `json:"avg_assists"`
}

// TeamAggregatedStats provides a summary of a team's performance, including win-loss record and point differentials.
// It's a read-only model derived from game results.
type TeamAggregatedStats struct {
	Wins               int     `json:"wins"`
	Losses             int     `json:"losses"`
	TotalPointsScored  int     `json:"total_points_scored"`
	TotalPointsAllowed int     `json:"total_points_allowed"`
	AvgPointsScored    float32 `json:"avg_points_scored"`
	AvgPointsAllowed   float32 `json:"avg_points_allowed"`
}
