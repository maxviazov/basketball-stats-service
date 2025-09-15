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
