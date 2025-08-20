package mcp

import (
	"fmt"
	"regexp"
	"strings"

	"healthcheck/pkg/healthcheck"
)

// buildProcessorConfig creates a ProcessorConfig from MCP parameters
func buildProcessorConfig(jobFilter, testFilter string, includeQuarantine bool) (healthcheck.ProcessorConfig, error) {
	// Handle job aliases
	if alias, exists := healthcheck.JobRegexAliases[jobFilter]; exists {
		jobFilter = alias
	}

	// Compile regex patterns
	jobRegex, err := regexp.Compile(jobFilter)
	if err != nil {
		return healthcheck.ProcessorConfig{}, fmt.Errorf("invalid job filter regex: %w", err)
	}

	testRegex, err := regexp.Compile(testFilter)
	if err != nil {
		return healthcheck.ProcessorConfig{}, fmt.Errorf("invalid test filter regex: %w", err)
	}

	return healthcheck.ProcessorConfig{
		JobRegex:        jobRegex,
		TestRegex:       testRegex,
		CountFailures:   true,
		CheckQuarantine: includeQuarantine,
	}, nil
}

// searchPatternsInResults searches for patterns in test results
func searchPatternsInResults(results *healthcheck.Results, pattern, jobFilter, searchIn string) []LLMPatternMatch {
	var matches []LLMPatternMatch

	// Compile patterns
	patternRegex, err := regexp.Compile("(?i)" + pattern)
	if err != nil {
		return matches
	}

	jobRegex, err := regexp.Compile(jobFilter)
	if err != nil {
		return matches
	}

	// Search through all jobs
	for _, job := range results.Data.SIGRetests.FailedJobLeaderBoard {
		if !jobRegex.MatchString(job.JobName) {
			continue
		}

		// For each failure URL, we need to fetch and check the test details
		// This is simplified - in practice you might want to cache this data
		for _, failureURL := range job.FailureURLs {
			// Extract test information from the job
			// For now, we'll search in the job name itself
			if searchIn == "test_names" || searchIn == "both" {
				if patternRegex.MatchString(job.JobName) {
					match := LLMPatternMatch{
						TestName: job.JobName,
						JobName:  job.JobName,
						URL:      failureURL,
						Context:  "job name match",
					}
					matches = append(matches, match)
				}
			}
		}
	}

	return matches
}

// Additional helper functions for MCP server operations

// validateJobName checks if a job name is valid
func validateJobName(jobName string) error {
	if jobName == "" {
		return fmt.Errorf("job name cannot be empty")
	}
	
	// Basic validation - job names should contain reasonable characters
	validJobName := regexp.MustCompile(`^[a-zA-Z0-9_-]+$`)
	if !validJobName.MatchString(jobName) {
		return fmt.Errorf("invalid job name format: %s", jobName)
	}
	
	return nil
}

// sanitizeTimePeriod ensures time period is in valid format
func sanitizeTimePeriod(period string) string {
	period = strings.ToLower(strings.TrimSpace(period))
	
	// Default fallbacks
	if period == "" {
		return "24h"
	}
	
	// Validate format with regex
	validPeriod := regexp.MustCompile(`^\d+[hdw]$`)
	if !validPeriod.MatchString(period) {
		return "24h" // fallback to safe default
	}
	
	return period
}

// limitResults limits the number of results to prevent overwhelming responses
func limitResults[T any](items []T, maxResults int) []T {
	if len(items) <= maxResults {
		return items
	}
	return items[:maxResults]
}

// extractTopFailures extracts the most significant failures for LLM analysis
func extractTopFailures(patterns []LLMFailurePattern, maxCount int) []LLMFailurePattern {
	// Sort by failure count (already sorted in formatters)
	if len(patterns) <= maxCount {
		return patterns
	}
	return patterns[:maxCount]
}

// categorizeFailuresByImpact groups failures by their potential impact
func categorizeFailuresByImpact(patterns []LLMFailurePattern) map[string][]LLMFailurePattern {
	categories := map[string][]LLMFailurePattern{
		"critical":  {},
		"high":      {},
		"medium":    {},
		"low":       {},
	}

	for _, pattern := range patterns {
		var impact string
		
		// Determine impact based on failure rate and frequency
		if pattern.Percentage > 50 || pattern.FailureCount > 10 {
			impact = "critical"
		} else if pattern.Percentage > 25 || pattern.FailureCount > 5 {
			impact = "high"
		} else if pattern.Percentage > 10 || pattern.FailureCount > 2 {
			impact = "medium"
		} else {
			impact = "low"
		}
		
		categories[impact] = append(categories[impact], pattern)
	}

	return categories
}

// generateInsights creates actionable insights from failure data
func generateInsights(analysis LLMJobAnalysis) []string {
	var insights []string
	
	// Health-based insights
	switch analysis.HealthStatus {
	case "critical":
		insights = append(insights, "URGENT: Job requires immediate attention - failure rate exceeds 80%")
	case "unhealthy":
		insights = append(insights, "WARNING: Job shows significant instability - investigate top failures")
	case "unstable":
		insights = append(insights, "NOTICE: Job stability is concerning - monitor trends closely")
	}
	
	// Pattern-based insights
	if len(analysis.TopFailures) > 0 {
		topFailure := analysis.TopFailures[0]
		if topFailure.Percentage > 50 {
			insights = append(insights, fmt.Sprintf("Single test dominates failures: %s (%.1f%%)", 
				topFailure.TestName, topFailure.Percentage))
		}
	}
	
	// Category-based insights
	for category, categoryData := range analysis.Categories {
		if categoryData.Percentage > 40 {
			insights = append(insights, fmt.Sprintf("%s tests are primary failure source (%.1f%%)", 
				strings.Title(category), categoryData.Percentage))
		}
	}
	
	// Trend-based insights
	if analysis.Trends.RegressionDetected {
		insights = append(insights, "Regression detected - compare with previous stable period")
	}
	
	if len(insights) == 0 {
		insights = append(insights, "No critical patterns detected - continue monitoring")
	}
	
	return insights
}

// formatInsightsForLLM formats insights in a way that's useful for LLM analysis
func formatInsightsForLLM(insights []string) string {
	if len(insights) == 0 {
		return "No significant insights generated."
	}
	
	result := "Key Insights:\n"
	for i, insight := range insights {
		result += fmt.Sprintf("%d. %s\n", i+1, insight)
	}
	
	return result
}

// detectAnomalies identifies unusual patterns in failure data
func detectAnomalies(patterns []LLMFailurePattern) []string {
	var anomalies []string
	
	if len(patterns) == 0 {
		return anomalies
	}
	
	// Check for single dominant failure
	if patterns[0].Percentage > 70 {
		anomalies = append(anomalies, fmt.Sprintf("Single test accounts for %.1f%% of all failures: %s", 
			patterns[0].Percentage, patterns[0].TestName))
	}
	
	// Check for many diverse failures
	if len(patterns) > 20 && patterns[0].Percentage < 20 {
		anomalies = append(anomalies, "High diversity in failures - no clear dominant pattern")
	}
	
	// Check for category concentration
	categoryCount := make(map[string]int)
	for _, pattern := range patterns {
		categoryCount[pattern.Category]++
	}
	
	for category, count := range categoryCount {
		if count > len(patterns)/2 {
			anomalies = append(anomalies, fmt.Sprintf("Failures concentrated in %s category (%d tests)", 
				category, count))
		}
	}
	
	return anomalies
}

// suggestNextSteps provides actionable recommendations based on analysis
func suggestNextSteps(analysis LLMJobAnalysis) []string {
	var steps []string
	
	switch analysis.HealthStatus {
	case "critical", "unhealthy":
		steps = append(steps, "1. Investigate top failing test immediately")
		steps = append(steps, "2. Check for recent code changes that might have caused regression")
		steps = append(steps, "3. Consider temporarily quarantining the most problematic tests")
		
	case "unstable":
		steps = append(steps, "1. Analyze failure patterns to identify root causes")
		steps = append(steps, "2. Review recent changes to infrastructure or test environment")
		
	default:
		steps = append(steps, "1. Continue monitoring for trend changes")
		steps = append(steps, "2. Address any intermittent failures to improve stability")
	}
	
	// Add category-specific suggestions
	for category := range analysis.Categories {
		switch category {
		case "migration":
			steps = append(steps, "• Review migration timeouts and resource allocation")
		case "network":
			steps = append(steps, "• Check network connectivity and DNS resolution")
		case "storage":
			steps = append(steps, "• Verify storage backend performance and capacity")
		}
	}
	
	return steps
}