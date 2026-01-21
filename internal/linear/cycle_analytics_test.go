package linear

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestCalculateCycleMetrics_TeamWide tests cycle metrics calculation with history arrays
func TestCalculateCycleMetrics_TeamWide(t *testing.T) {
	cycle := &Cycle{
		ID:        "cycle-1",
		Name:      "Cycle 67",
		StartsAt:  "2026-01-06",
		EndsAt:    "2026-01-20",
		CompletedAt: nil,
		// History arrays: showing changes over time
		ScopeHistory:               []int{40, 42, 45, 48}, // Started at 40, grew to 48
		CompletedScopeHistory:      []int{10, 15, 25, 32}, // Completed grew to 32
		IssueCountHistory:          []int{18, 19, 20, 20},
		CompletedIssueCountHistory: []int{3, 5, 9, 14},
	}

	metrics := CalculateCycleMetrics(cycle, nil)

	assert.Equal(t, "cycle-1", metrics.CycleID)
	assert.Equal(t, 40, metrics.InitialScope)
	assert.Equal(t, 48, metrics.FinalScope)
	assert.Equal(t, 32, metrics.CompletedScope)
	assert.Equal(t, 8, metrics.ScopeCreep) // 48 - 40
	assert.Equal(t, 20.0, metrics.ScopeCreepPercent) // (8/40)*100
	assert.InDelta(t, 66.67, metrics.CompletionRate, 0.1) // (32/48)*100
	assert.Equal(t, 18, metrics.InitialIssueCount)
	assert.Equal(t, 14, metrics.CompletedIssues)
	assert.Equal(t, 14, metrics.Throughput)
}

// TestCalculateCycleMetrics_PerUser tests cycle metrics calculation with filtered issues
func TestCalculateCycleMetrics_PerUser(t *testing.T) {
	userIssues := []Issue{
		{
			ID:        "issue-1",
			Identifier: "CEN-100",
			Title:     "Feature A",
			Estimate:  ptr(5.0),
			State: struct {
				ID   string `json:"id"`
				Name string `json:"name"`
			}{"state-1", "Done"},
		},
		{
			ID:        "issue-2",
			Identifier: "CEN-101",
			Title:     "Feature B",
			Estimate:  ptr(3.0),
			State: struct {
				ID   string `json:"id"`
				Name string `json:"name"`
			}{"state-2", "In Progress"},
		},
		{
			ID:        "issue-3",
			Identifier: "CEN-102",
			Title:     "Feature C",
			Estimate:  ptr(2.0),
			State: struct {
				ID   string `json:"id"`
				Name string `json:"name"`
			}{"state-1", "Done"},
		},
	}

	cycle := &Cycle{
		ID:       "cycle-1",
		Name:     "Cycle 67",
		StartsAt: "2026-01-06",
		EndsAt:   "2026-01-20",
	}

	metrics := CalculateCycleMetrics(cycle, userIssues)

	assert.Equal(t, 10, metrics.InitialScope) // 5 + 3 + 2
	assert.Equal(t, 10, metrics.FinalScope)   // Estimates don't change
	assert.Equal(t, 7, metrics.CompletedScope) // 5 + 2 (only Done issues)
	assert.Equal(t, 3, metrics.InitialIssueCount)
	assert.Equal(t, 2, metrics.CompletedIssues) // 2 Done issues
	assert.InDelta(t, 70.0, metrics.CompletionRate, 0.1) // (7/10)*100
}

// TestCalculateCycleMetrics_EdgeCases tests handling of edge cases
func TestCalculateCycleMetrics_EdgeCases(t *testing.T) {
	t.Run("empty history arrays", func(t *testing.T) {
		cycle := &Cycle{
			ID:                         "cycle-1",
			Name:                       "Cycle 1",
			StartsAt:                   "2026-01-01",
			EndsAt:                     "2026-01-15",
			ScopeHistory:               []int{},
			CompletedScopeHistory:      []int{},
			IssueCountHistory:          []int{},
			CompletedIssueCountHistory: []int{},
		}

		metrics := CalculateCycleMetrics(cycle, nil)

		assert.Equal(t, 0, metrics.InitialScope)
		assert.Equal(t, 0, metrics.CompletedScope)
		assert.Equal(t, 0.0, metrics.CompletionRate)
	})

	t.Run("nil estimates", func(t *testing.T) {
		userIssues := []Issue{
			{
				ID:         "issue-1",
				Identifier: "CEN-100",
				Title:      "Feature A",
				Estimate:   nil, // No estimate
				State: struct {
					ID   string `json:"id"`
					Name string `json:"name"`
				}{"state-1", "Done"},
			},
		}

		cycle := &Cycle{
			ID:       "cycle-1",
			Name:     "Cycle 1",
			StartsAt: "2026-01-01",
			EndsAt:   "2026-01-15",
		}

		metrics := CalculateCycleMetrics(cycle, userIssues)

		assert.Equal(t, 0, metrics.InitialScope)
		assert.Equal(t, 1, metrics.CompletedIssues)
	})

	t.Run("zero final scope prevents division", func(t *testing.T) {
		cycle := &Cycle{
			ID:                         "cycle-1",
			Name:                       "Cycle 1",
			StartsAt:                   "2026-01-01",
			EndsAt:                     "2026-01-15",
			ScopeHistory:               []int{0},
			CompletedScopeHistory:      []int{0},
			IssueCountHistory:          []int{0},
			CompletedIssueCountHistory: []int{0},
		}

		metrics := CalculateCycleMetrics(cycle, nil)

		assert.Equal(t, 0.0, metrics.CompletionRate)
	})
}

// TestAnalyzeMultipleCycles tests aggregating metrics across cycles
func TestAnalyzeMultipleCycles(t *testing.T) {
	cycles := []*Cycle{
		{
			ID:                         "cycle-1",
			Name:                       "Cycle 65",
			ScopeHistory:               []int{30, 32, 35},
			CompletedScopeHistory:      []int{10, 20, 28},
			IssueCountHistory:          []int{15, 16, 16},
			CompletedIssueCountHistory: []int{5, 10, 14},
		},
		{
			ID:                         "cycle-2",
			Name:                       "Cycle 66",
			ScopeHistory:               []int{35, 38, 40},
			CompletedScopeHistory:      []int{15, 25, 28},
			IssueCountHistory:          []int{16, 17, 17},
			CompletedIssueCountHistory: []int{6, 11, 13},
		},
		{
			ID:                         "cycle-3",
			Name:                       "Cycle 67",
			ScopeHistory:               []int{40, 45, 48},
			CompletedScopeHistory:      []int{10, 20, 32},
			IssueCountHistory:          []int{18, 19, 20},
			CompletedIssueCountHistory: []int{3, 8, 14},
		},
	}

	analysis := AnalyzeMultipleCycles(cycles, nil)

	assert.Equal(t, 3, analysis.CycleCount)
	assert.Equal(t, 3, len(analysis.Metrics))

	// Check aggregated velocity (completed scope)
	expectedAvgVelocity := (28 + 28 + 32) / 3.0 // (28 + 28 + 32) / 3 = 29.33
	assert.InDelta(t, expectedAvgVelocity, analysis.AvgVelocity, 0.1)

	// Check median velocity
	assert.InDelta(t, 28.0, analysis.MedianVelocity, 0.1)

	// Check percentiles
	assert.True(t, analysis.P80Velocity >= analysis.MedianVelocity)
	assert.True(t, analysis.P20Velocity <= analysis.MedianVelocity)
}

// TestAnalyzeMultipleCycles_EmptyList tests handling empty cycle list
func TestAnalyzeMultipleCycles_EmptyList(t *testing.T) {
	analysis := AnalyzeMultipleCycles([]*Cycle{}, nil)

	assert.Equal(t, 0, analysis.CycleCount)
	assert.Equal(t, 0, len(analysis.Metrics))
	assert.Equal(t, 0.0, analysis.AvgVelocity)
}

// TestGenerateCapacityRecommendation tests recommendation generation
func TestGenerateCapacityRecommendation(t *testing.T) {
	analysis := &CycleAnalysis{
		AvgVelocity:          34.0,
		AvgCompletionRate:    78.5,
		AvgScopeCreep:        4.2,
		AvgScopeCreepPercent: 12.3,
		AvgThroughput:        15.0,
		MedianVelocity:       35.0,
		P80Velocity:          32.0, // Conservative
		P20Velocity:          38.0, // Optimistic
	}

	rec := GenerateCapacityRecommendation(analysis, false)

	assert.Equal(t, 32, rec.ConservativeScope)
	assert.Equal(t, 35, rec.TargetScope)
	assert.Equal(t, 38, rec.OptimisticScope)

	// Should have reasonable issue counts
	assert.Greater(t, rec.ConservativeIssues, 0)
	assert.Greater(t, rec.TargetIssues, rec.ConservativeIssues)
	assert.Greater(t, rec.OptimisticIssues, rec.TargetIssues)

	// Check rationale is generated
	assert.NotEmpty(t, rec.Rationale)
}

// TestGenerateCapacityRecommendation_PerUser tests recommendation for individual users
func TestGenerateCapacityRecommendation_PerUser(t *testing.T) {
	analysis := &CycleAnalysis{
		AvgVelocity:       8.5,
		AvgCompletionRate: 92.3,
		AvgThroughput:     4.2,
	}

	rec := GenerateCapacityRecommendation(analysis, true)

	// Rationale should mention individual performance
	assert.Contains(t, rec.Rationale, "reliable delivery")
	assert.NotEmpty(t, rec.Rationale)
}

// TestPercentile tests percentile calculations
func TestPercentile(t *testing.T) {
	values := []float64{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}

	// Test various percentiles
	assert.Equal(t, 1.0, percentile(values, 0))
	assert.InDelta(t, 5.5, percentile(values, 50), 0.1) // Median
	assert.Equal(t, 10.0, percentile(values, 100))
}

// TestPercentile_EdgeCases tests percentile with edge cases
func TestPercentile_EdgeCases(t *testing.T) {
	t.Run("empty array", func(t *testing.T) {
		assert.Equal(t, 0.0, percentile([]float64{}, 50))
	})

	t.Run("single element", func(t *testing.T) {
		assert.Equal(t, 42.0, percentile([]float64{42}, 50))
	})

	t.Run("two elements", func(t *testing.T) {
		result := percentile([]float64{1, 2}, 50)
		assert.InDelta(t, 1.5, result, 0.1)
	})
}

// TestAverageAndStdDev tests statistical helper functions
func TestAverageAndStdDev(t *testing.T) {
	values := []float64{10, 20, 30, 40, 50}

	avg := average(values)
	assert.Equal(t, 30.0, avg)

	stddev := stdDev(values, avg)
	assert.InDelta(t, 14.14, stddev, 0.1) // sqrt(200) ~14.14 for this data
}

// Helper function
func ptr(f float64) *float64 {
	return &f
}
