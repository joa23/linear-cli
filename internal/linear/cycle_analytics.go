package linear

import (
	"fmt"
	"math"
	"sort"
)

// CycleMetrics represents calculated metrics for a single cycle
type CycleMetrics struct {
	CycleID            string
	CycleName          string
	StartDate          string
	EndDate            string
	CompletedAt        *string

	// Scope metrics
	InitialScope       int     // scopeHistory[0] or sum of initial issue estimates
	FinalScope         int     // scopeHistory[last] or sum of final issue estimates
	CompletedScope     int     // completedScopeHistory[last] or sum of completed issue estimates
	ScopeCreep         int     // finalScope - initialScope
	ScopeCreepPercent  float64 // (scopeCreep / initialScope) * 100

	// Completion metrics
	CompletionRate float64 // (completedScope / finalScope) * 100

	// Issue metrics
	InitialIssueCount int // issueCountHistory[0] or count of initial issues
	CompletedIssues   int // completedIssueCountHistory[last] or count of completed issues
	Throughput        int // completedIssues
}

// CycleAnalysis represents aggregated metrics across multiple cycles
type CycleAnalysis struct {
	CycleCount int
	Metrics    []CycleMetrics

	// Summary statistics
	AvgVelocity         float64 // average completedScope
	AvgCompletionRate   float64
	AvgScopeCreep       float64
	AvgScopeCreepPercent float64
	AvgThroughput       float64

	// Statistical measures
	StdDevVelocity float64
	MedianVelocity float64
	P80Velocity    float64 // 80th percentile (conservative)
	P20Velocity    float64 // 20th percentile (optimistic)
}

// CapacityRecommendation represents suggested cycle scope based on historical data
type CapacityRecommendation struct {
	ConservativeScope  int     // P80 velocity
	TargetScope        int     // Median velocity
	OptimisticScope    int     // P20 velocity
	ConservativeIssues int     // Based on avg points/issue
	TargetIssues       int
	OptimisticIssues   int
	Rationale          string  // Explanation of calculation
}

// CalculateCycleMetrics calculates metrics for a single cycle
// If userIssues is nil/empty, uses cycle history arrays (team-wide analysis)
// If userIssues is provided, calculates from filtered issues (per-user analysis)
func CalculateCycleMetrics(cycle *Cycle, userIssues []Issue) *CycleMetrics {
	metrics := &CycleMetrics{
		CycleID:     cycle.ID,
		CycleName:   cycle.Name,
		StartDate:   cycle.StartsAt,
		EndDate:     cycle.EndsAt,
		CompletedAt: cycle.CompletedAt,
	}

	// If userIssues provided, calculate from filtered issues (per-user)
	if len(userIssues) > 0 {
		return calculateUserMetrics(metrics, userIssues)
	}

	// Otherwise use cycle history arrays (team-wide)
	return calculateTeamMetrics(metrics, cycle)
}

// calculateTeamMetrics uses cycle history arrays for team-wide analysis
func calculateTeamMetrics(metrics *CycleMetrics, cycle *Cycle) *CycleMetrics {
	// Extract initial scope (start of cycle)
	if len(cycle.ScopeHistory) > 0 {
		metrics.InitialScope = cycle.ScopeHistory[0]
	}

	// Extract final scope (end of cycle)
	if len(cycle.ScopeHistory) > 0 {
		metrics.FinalScope = cycle.ScopeHistory[len(cycle.ScopeHistory)-1]
	}

	// Extract completed scope
	if len(cycle.CompletedScopeHistory) > 0 {
		metrics.CompletedScope = cycle.CompletedScopeHistory[len(cycle.CompletedScopeHistory)-1]
	}

	// Calculate scope creep
	metrics.ScopeCreep = metrics.FinalScope - metrics.InitialScope
	if metrics.InitialScope > 0 {
		metrics.ScopeCreepPercent = (float64(metrics.ScopeCreep) / float64(metrics.InitialScope)) * 100
	}

	// Calculate completion rate
	if metrics.FinalScope > 0 {
		metrics.CompletionRate = (float64(metrics.CompletedScope) / float64(metrics.FinalScope)) * 100
	}

	// Extract issue counts
	if len(cycle.IssueCountHistory) > 0 {
		metrics.InitialIssueCount = cycle.IssueCountHistory[0]
	}
	if len(cycle.CompletedIssueCountHistory) > 0 {
		metrics.CompletedIssues = cycle.CompletedIssueCountHistory[len(cycle.CompletedIssueCountHistory)-1]
	}
	metrics.Throughput = metrics.CompletedIssues

	return metrics
}

// calculateUserMetrics calculates metrics from filtered user issues
func calculateUserMetrics(metrics *CycleMetrics, userIssues []Issue) *CycleMetrics {
	completedCount := 0
	completedScope := 0

	for _, issue := range userIssues {
		// Add estimate to initial scope
		if issue.Estimate != nil {
			metrics.InitialScope += int(*issue.Estimate)
			metrics.FinalScope += int(*issue.Estimate)
		}

		metrics.InitialIssueCount++

		// Check if issue is completed (in a completed state)
		if isCompletedState(issue.State.Name) {
			completedCount++
			if issue.Estimate != nil {
				completedScope += int(*issue.Estimate)
			}
		}
	}

	metrics.CompletedIssues = completedCount
	metrics.CompletedScope = completedScope
	metrics.Throughput = completedCount

	// Calculate scope creep (for per-user, typically minimal as estimates don't change much)
	metrics.ScopeCreep = 0
	metrics.ScopeCreepPercent = 0

	// Calculate completion rate
	if metrics.FinalScope > 0 {
		metrics.CompletionRate = (float64(metrics.CompletedScope) / float64(metrics.FinalScope)) * 100
	}

	return metrics
}

// isCompletedState checks if an issue state indicates completion
func isCompletedState(state string) bool {
	// Common completed states in Linear
	completedStates := map[string]bool{
		"Done":      true,
		"Completed": true,
		"Closed":    true,
		"Finished":  true,
		"Resolved":  true,
	}
	return completedStates[state]
}

// AnalyzeMultipleCycles aggregates metrics across multiple cycles
func AnalyzeMultipleCycles(cycles []*Cycle, userIssues map[string][]Issue) *CycleAnalysis {
	analysis := &CycleAnalysis{
		CycleCount: len(cycles),
	}

	if len(cycles) == 0 {
		return analysis
	}

	// Calculate metrics for each cycle
	var velocities []float64
	var completionRates []float64
	var scopeCreeps []float64
	var throughputs []float64

	for _, cycle := range cycles {
		// Get user issues for this cycle if filtering
		var issues []Issue
		if userIssues != nil {
			issues = userIssues[cycle.ID]
		}

		metrics := CalculateCycleMetrics(cycle, issues)
		analysis.Metrics = append(analysis.Metrics, *metrics)

		// Collect data for aggregation
		velocities = append(velocities, float64(metrics.CompletedScope))
		completionRates = append(completionRates, metrics.CompletionRate)
		scopeCreeps = append(scopeCreeps, float64(metrics.ScopeCreep))
		throughputs = append(throughputs, float64(metrics.Throughput))
	}

	// Calculate averages
	analysis.AvgVelocity = average(velocities)
	analysis.AvgCompletionRate = average(completionRates)
	analysis.AvgScopeCreep = average(scopeCreeps)
	analysis.AvgThroughput = average(throughputs)

	// Calculate scope creep percentage
	var scopeCreepPercents []float64
	for _, m := range analysis.Metrics {
		scopeCreepPercents = append(scopeCreepPercents, m.ScopeCreepPercent)
	}
	analysis.AvgScopeCreepPercent = average(scopeCreepPercents)

	// Calculate standard deviation
	analysis.StdDevVelocity = stdDev(velocities, analysis.AvgVelocity)

	// Calculate percentiles
	sort.Float64s(velocities)
	analysis.MedianVelocity = percentile(velocities, 50)
	analysis.P80Velocity = percentile(velocities, 80)
	analysis.P20Velocity = percentile(velocities, 20)

	return analysis
}

// GenerateCapacityRecommendation creates scope recommendations based on analysis
func GenerateCapacityRecommendation(analysis *CycleAnalysis, isPerUser bool) *CapacityRecommendation {
	rec := &CapacityRecommendation{
		ConservativeScope: int(analysis.P80Velocity),
		TargetScope:       int(analysis.MedianVelocity),
		OptimisticScope:   int(analysis.P20Velocity),
	}

	// Calculate recommended issue counts based on average points per issue
	avgPointsPerIssue := 1.0
	if analysis.AvgThroughput > 0 {
		avgPointsPerIssue = analysis.AvgVelocity / analysis.AvgThroughput
	}

	rec.ConservativeIssues = int(float64(rec.ConservativeScope) / avgPointsPerIssue)
	rec.TargetIssues = int(float64(rec.TargetScope) / avgPointsPerIssue)
	rec.OptimisticIssues = int(float64(rec.OptimisticScope) / avgPointsPerIssue)

	// Generate rationale
	if isPerUser {
		rec.Rationale = fmt.Sprintf("- Completes avg %.1f points/cycle\n- Very high completion rate (%.1f%%) indicates reliable delivery\n- Minimal scope creep suggests good initial estimation\n- Recommend staying within %d-%d point range for sustainable pace",
			analysis.AvgVelocity,
			analysis.AvgCompletionRate,
			rec.ConservativeScope,
			rec.OptimisticScope)
	} else {
		rec.Rationale = fmt.Sprintf("- Team completes avg %.1f points/cycle\n- Accounting for %.1f%% scope creep, start with %d points\n- Add buffer: recommend %d-%d point range",
			analysis.AvgVelocity,
			analysis.AvgScopeCreepPercent,
			int(analysis.AvgVelocity*(1-analysis.AvgScopeCreepPercent/100)),
			rec.ConservativeScope,
			rec.OptimisticScope)
	}

	return rec
}

// Helper functions

func average(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}
	sum := 0.0
	for _, v := range values {
		sum += v
	}
	return sum / float64(len(values))
}

func stdDev(values []float64, mean float64) float64 {
	if len(values) < 2 {
		return 0
	}
	sumSquares := 0.0
	for _, v := range values {
		diff := v - mean
		sumSquares += diff * diff
	}
	variance := sumSquares / float64(len(values))
	return math.Sqrt(variance)
}

func percentile(sortedValues []float64, p float64) float64 {
	if len(sortedValues) == 0 {
		return 0
	}
	if len(sortedValues) == 1 {
		return sortedValues[0]
	}

	// Use linear interpolation method
	index := (p / 100) * float64(len(sortedValues)-1)
	lower := int(index)
	upper := lower + 1

	if upper >= len(sortedValues) {
		return sortedValues[len(sortedValues)-1]
	}

	weight := index - float64(lower)
	return sortedValues[lower]*(1-weight) + sortedValues[upper]*weight
}
