package repository_test

import (
	"context"
	"testing"
	"time"

	"github.com/maxviazov/basketball-stats-service/internal/model"
	pg "github.com/maxviazov/basketball-stats-service/internal/repository/postgres"
	"github.com/stretchr/testify/require"
)

// TestAdvancedStatsPostgres contains integration tests for advanced aggregation queries.
func TestAdvancedStatsPostgres(t *testing.T) {
	skipIfNeeded(t)
	truncateAll(t)

	// 1. Setup: Create repositories and seed data
	teamRepo := pg.NewTeamRepository(pool)
	playerRepo := pg.NewPlayerRepository(pool)
	gameRepo := pg.NewGameRepository(pool)
	statsRepo := pg.NewStatsRepository(pool)

	ctx := context.Background()

	// Seed teams
	t1, err := teamRepo.Create(ctx, model.Team{Name: "Lakers"})
	require.NoError(t, err)
	t2, err := teamRepo.Create(ctx, model.Team{Name: "Clippers"})
	require.NoError(t, err)

	// Seed players
	p1, err := playerRepo.Create(ctx, model.Player{TeamID: t1.ID, FirstName: "LeBron", LastName: "James", Position: "SF"})
	require.NoError(t, err)

	// Seed games in different seasons
	g1, err := gameRepo.Create(ctx, model.Game{Season: "2023-24", Date: time.Now(), HomeTeamID: t1.ID, AwayTeamID: t2.ID, Status: "finished"})
	require.NoError(t, err)
	g2, err := gameRepo.Create(ctx, model.Game{Season: "2023-24", Date: time.Now(), HomeTeamID: t2.ID, AwayTeamID: t1.ID, Status: "finished"})
	require.NoError(t, err)
	g3, err := gameRepo.Create(ctx, model.Game{Season: "2024-25", Date: time.Now(), HomeTeamID: t1.ID, AwayTeamID: t2.ID, Status: "finished"})
	require.NoError(t, err)

	// Seed player stats for those games
	_, err = statsRepo.UpsertStatLine(ctx, model.PlayerStatLine{PlayerID: p1.ID, GameID: g1.ID, Points: 25, Rebounds: 8, Assists: 7})
	require.NoError(t, err)
	_, err = statsRepo.UpsertStatLine(ctx, model.PlayerStatLine{PlayerID: p1.ID, GameID: g2.ID, Points: 30, Rebounds: 10, Assists: 5})
	require.NoError(t, err)
	_, err = statsRepo.UpsertStatLine(ctx, model.PlayerStatLine{PlayerID: p1.ID, GameID: g3.ID, Points: 22, Rebounds: 6, Assists: 9})
	require.NoError(t, err)

	// 2. Run Player Aggregated Stats Tests
	t.Run("PlayerAggregatedStats", func(t *testing.T) {
		t.Run("Career Stats", func(t *testing.T) {
			stats, err := playerRepo.GetPlayerAggregatedStats(ctx, p1.ID, nil)
			require.NoError(t, err)
			require.Equal(t, 3, stats.GamesPlayed)
			require.Equal(t, 77, stats.TotalPoints)   // 25 + 30 + 22
			require.Equal(t, 24, stats.TotalRebounds) // 8 + 10 + 6
			require.Equal(t, 21, stats.TotalAssists)  // 7 + 5 + 9
			require.InEpsilon(t, 25.66, stats.AvgPoints, 0.01)
		})

		t.Run("Seasonal Stats", func(t *testing.T) {
			season := "2023-24"
			stats, err := playerRepo.GetPlayerAggregatedStats(ctx, p1.ID, &season)
			require.NoError(t, err)
			require.Equal(t, 2, stats.GamesPlayed)
			require.Equal(t, 55, stats.TotalPoints) // 25 + 30
			require.Equal(t, 18, stats.TotalRebounds)
			require.Equal(t, 12, stats.TotalAssists)
			require.InEpsilon(t, 27.5, stats.AvgPoints, 0.01)
		})
	})

	// 3. Run Team Aggregated Stats Tests
	t.Run("TeamAggregatedStats", func(t *testing.T) {
		// To test team stats, we need scores for both teams.
		p2, err := playerRepo.Create(ctx, model.Player{TeamID: t2.ID, FirstName: "Kawhi", LastName: "Leonard", Position: "SF"})
		require.NoError(t, err)
		// Game 1: Lakers (p1: 25) vs Clippers (p2: 20) -> Lakers win
		_, err = statsRepo.UpsertStatLine(ctx, model.PlayerStatLine{PlayerID: p2.ID, GameID: g1.ID, Points: 20})
		require.NoError(t, err)
		// Game 2: Clippers (p2: 35) vs Lakers (p1: 30) -> Clippers win
		_, err = statsRepo.UpsertStatLine(ctx, model.PlayerStatLine{PlayerID: p2.ID, GameID: g2.ID, Points: 35})
		require.NoError(t, err)
		// Game 3: Lakers (p1: 22) vs Clippers (p2: 18) -> Lakers win
		_, err = statsRepo.UpsertStatLine(ctx, model.PlayerStatLine{PlayerID: p2.ID, GameID: g3.ID, Points: 18})
		require.NoError(t, err)

		t.Run("Career Stats", func(t *testing.T) {
			stats, err := teamRepo.GetTeamAggregatedStats(ctx, t1.ID, nil) // Lakers
			require.NoError(t, err)
			require.Equal(t, 2, stats.Wins)                // g1, g3
			require.Equal(t, 1, stats.Losses)              // g2
			require.Equal(t, 77, stats.TotalPointsScored)  // 25 + 30 + 22
			require.Equal(t, 73, stats.TotalPointsAllowed) // 20 + 35 + 18
		})

		t.Run("Seasonal Stats", func(t *testing.T) {
			season := "2023-24"
			stats, err := teamRepo.GetTeamAggregatedStats(ctx, t1.ID, &season) // Lakers
			require.NoError(t, err)
			require.Equal(t, 1, stats.Wins)
			require.Equal(t, 1, stats.Losses)
			require.Equal(t, 55, stats.TotalPointsScored)  // 25 + 30
			require.Equal(t, 55, stats.TotalPointsAllowed) // 20 + 35
		})
	})
}
