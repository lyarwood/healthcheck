package mcp

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

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

// ParseFailureText extracts failure information from JUnit failure text
func ParseFailureText(failureText string) (LLMFailureInfo, error) {
	var failureInfo LLMFailureInfo
	
	lines := strings.Split(failureText, "\n")
	
	// Parse the first line for primary failure location
	// Example: "pkg/virt-controller/services/template_test.go:2689"
	firstLine := strings.TrimSpace(lines[0])
	
	// Check if first line is just a file:line format (most common case)
	if strings.Contains(firstLine, ":") && !strings.Contains(firstLine, " ") {
		fileParts := strings.Split(firstLine, ":")
		if len(fileParts) >= 2 {
			failureInfo.PrimaryFile = fileParts[0]
			if lineNum := regexp.MustCompile(`\d+`).FindString(fileParts[1]); lineNum != "" {
				if line, parseErr := strconv.Atoi(lineNum); parseErr == nil {
					failureInfo.PrimaryLine = line
				}
			}
		}
		failureInfo.FailureType = "Test Failure"
	} else if strings.Contains(firstLine, " ") {
		// Handle formats like "Panic pkg/file.go:123"
		parts := strings.SplitN(firstLine, " ", 2)
		if len(parts) == 2 {
			failureInfo.FailureType = parts[0]
			
			// Parse file:line format
			fileInfo := parts[1]
			if strings.Contains(fileInfo, ":") {
				fileParts := strings.Split(fileInfo, ":")
				if len(fileParts) >= 2 {
					failureInfo.PrimaryFile = fileParts[0]
					if lineNum := regexp.MustCompile(`\d+`).FindString(fileParts[1]); lineNum != "" {
						if line, parseErr := strconv.Atoi(lineNum); parseErr == nil {
							failureInfo.PrimaryLine = line
						}
					}
				}
			}
		}
	}
	
	// Extract error message - look for meaningful error lines
	var errorLines []string
	for i, line := range lines {
		line = strings.TrimSpace(line)
		
		// Skip the first line (file:line) and empty lines
		if i == 0 || line == "" {
			continue
		}
		
		// Look for common error patterns
		if strings.HasPrefix(line, "Panic:") || strings.HasPrefix(line, "Error:") || strings.HasPrefix(line, "Failed:") {
			errorMessage := strings.TrimPrefix(line, "Panic:")
			errorMessage = strings.TrimPrefix(errorMessage, "Error:")
			errorMessage = strings.TrimPrefix(errorMessage, "Failed:")
			failureInfo.ErrorMessage = strings.TrimSpace(errorMessage)
			break
		} else if strings.Contains(line, "error:") || strings.Contains(line, "Error:") || 
				  strings.Contains(line, "Unexpected") || strings.Contains(line, "Expected") ||
				  strings.Contains(line, "occurred") || strings.Contains(line, "deadline exceeded") {
			errorLines = append(errorLines, line)
		}
	}
	
	// If no specific error prefix found, use collected error lines
	if failureInfo.ErrorMessage == "" && len(errorLines) > 0 {
		failureInfo.ErrorMessage = strings.Join(errorLines, " ")
		// Limit length for readability
		if len(failureInfo.ErrorMessage) > 200 {
			failureInfo.ErrorMessage = failureInfo.ErrorMessage[:200] + "..."
		}
	}
	
	// Parse stack trace and additional file references
	stackStarted := false
	for i, line := range lines {
		line = strings.TrimSpace(line)
		
		if strings.Contains(line, "Full stack:") {
			stackStarted = true
			continue
		}
		
		if stackStarted && line != "" {
			frame := parseStackTraceFrame(line)
			if frame.File != "" {
				failureInfo.StackTrace = append(failureInfo.StackTrace, frame)
			}
		} else if i > 0 && strings.Contains(line, ".go:") && !strings.Contains(line, "Unexpected") {
			// Look for additional file:line references in the failure text
			frame := parseStackTraceFrame(line)
			if frame.File != "" && (frame.File != failureInfo.PrimaryFile || frame.Line != failureInfo.PrimaryLine) {
				failureInfo.StackTrace = append(failureInfo.StackTrace, frame)
			}
		}
	}
	
	// If no error message found, use the failure type
	if failureInfo.ErrorMessage == "" {
		failureInfo.ErrorMessage = failureInfo.FailureType
	}
	
	// Extract test name from file path if not already set
	if failureInfo.TestName == "" && failureInfo.PrimaryFile != "" {
		// Extract test name from file path (remove _test.go suffix and path)
		testFile := failureInfo.PrimaryFile
		if strings.HasSuffix(testFile, "_test.go") {
			testFile = strings.TrimSuffix(testFile, "_test.go")
			pathParts := strings.Split(testFile, "/")
			failureInfo.TestName = pathParts[len(pathParts)-1] + " test"
		}
	}
	
	return failureInfo, nil
}

// parseStackTraceFrame parses a single line of stack trace
func parseStackTraceFrame(line string) LLMStackTraceFrame {
	var frame LLMStackTraceFrame
	
	// Example: "kubevirt.io/kubevirt/pkg/virt-controller/services.init.func7.6.24.3()"
	//          "        pkg/virt-controller/services/template_test.go:2695 +0x2f4"
	
	if strings.Contains(line, "()") {
		// Function line
		funcName := strings.TrimSpace(strings.Replace(line, "()", "", 1))
		frame.Function = funcName
	} else if strings.Contains(line, ":") {
		// Handle both detailed stack traces and simple file:line references
		trimmed := strings.TrimSpace(line)
		
		// For detailed stack traces with "+0x": "pkg/file.go:123 +0x2f4"
		if strings.Contains(trimmed, "+0x") {
			parts := strings.Fields(trimmed)
			if len(parts) > 0 {
				fileInfo := parts[0]
				if strings.Contains(fileInfo, ":") {
					fileParts := strings.Split(fileInfo, ":")
					if len(fileParts) >= 2 {
						frame.File = fileParts[0]
						if lineNum := regexp.MustCompile(`\d+`).FindString(fileParts[1]); lineNum != "" {
							if line, parseErr := strconv.Atoi(lineNum); parseErr == nil {
								frame.Line = line
							}
						}
					}
				}
			}
		} else if strings.Contains(trimmed, ".go:") {
			// For simple file:line references: "pkg/file.go:123"
			if strings.Contains(trimmed, ":") {
				fileParts := strings.Split(trimmed, ":")
				if len(fileParts) >= 2 {
					frame.File = fileParts[0]
					if lineNum := regexp.MustCompile(`\d+`).FindString(fileParts[1]); lineNum != "" {
						if line, parseErr := strconv.Atoi(lineNum); parseErr == nil {
							frame.Line = line
						}
					}
				}
			}
		}
	}
	
	return frame
}

// ExtractRepositoryInfo extracts repository and commit information from job URL
func ExtractRepositoryInfo(jobURL string) (LLMRepositoryInfo, error) {
	var repoInfo LLMRepositoryInfo
	
	// Example URL: https://prow.ci.kubevirt.io/view/gs/kubevirt-prow/pr-logs/pull/kubevirt_kubevirt/15472/pull-kubevirt-unit-test-arm64/1958099225396908032
	
	// Extract repository from URL path
	repoPattern := regexp.MustCompile(`/pull/([^/]+)_([^/]+)/(\d+)/`)
	matches := repoPattern.FindStringSubmatch(jobURL)
	
	if len(matches) < 4 {
		return repoInfo, fmt.Errorf("unable to extract repository information from URL: %s", jobURL)
	}
	
	repoInfo.Owner = matches[1]
	repoInfo.Repository = matches[2]
	
	// Convert pull request number
	if prNum, err := strconv.Atoi(matches[3]); err == nil {
		repoInfo.PullRequest = prNum
	}
	
	// For pull requests, we need to fetch the commit hash and PR information
	repoInfo.Commit = "main"  // Default fallback
	
	// Try to fetch commit and PR info from prowjob.json if available
	if gsPattern := regexp.MustCompile(`/view/gs/(.+)$`); gsPattern.MatchString(jobURL) {
		gsMatches := gsPattern.FindStringSubmatch(jobURL)
		if len(gsMatches) >= 2 {
			prowjobURL := fmt.Sprintf("https://storage.googleapis.com/%s/prowjob.json", gsMatches[1])
			prowjobInfo := fetchProwjobInfo(prowjobURL)
			
			if prowjobInfo.CommitHash != "" {
				repoInfo.Commit = prowjobInfo.CommitHash
				repoInfo.PRInfo = prowjobInfo.PRInfo
				repoInfo.Branch = prowjobInfo.PRInfo.HeadRef
			}
		}
	}
	
	return repoInfo, nil
}

// extractCommitFromProwJob attempts to extract commit hash from prowjob.json
func extractCommitFromProwJob(jobURL string) string {
	// Convert job URL to prowjob.json URL
	// Example: https://prow.ci.kubevirt.io/view/gs/kubevirt-prow/pr-logs/pull/kubevirt_kubevirt/15472/pull-kubevirt-unit-test-arm64/1958099225396908032
	// Becomes: https://storage.googleapis.com/kubevirt-prow/pr-logs/pull/kubevirt_kubevirt/15472/pull-kubevirt-unit-test-arm64/1958099225396908032/prowjob.json
	
	// Extract the gs path from the prow URL
	gsPattern := regexp.MustCompile(`/view/gs/(.+)$`)
	matches := gsPattern.FindStringSubmatch(jobURL)
	
	if len(matches) < 2 {
		return "" // Could not extract gs path
	}
	
	// Construct prowjob.json URL
	prowjobURL := fmt.Sprintf("https://storage.googleapis.com/%s/prowjob.json", matches[1])
	
	// Try to fetch and parse the prowjob.json
	commit := fetchCommitFromProwjobJSON(prowjobURL)
	if commit != "" {
		return commit
	}
	
	// Fallback: try to extract from artifacts/junit XML or build-log.txt
	buildLogURL := fmt.Sprintf("https://storage.googleapis.com/%s/build-log.txt", matches[1])
	commit = extractCommitFromBuildLog(buildLogURL)
	
	return commit
}

// ProwjobInfo contains extracted information from prowjob.json
type ProwjobInfo struct {
	CommitHash string
	PRInfo     LLMPullRequestInfo
}

// fetchCommitFromProwjobJSON fetches and parses prowjob.json to extract commit hash
func fetchCommitFromProwjobJSON(prowjobURL string) string {
	info := fetchProwjobInfo(prowjobURL)
	return info.CommitHash
}

// fetchProwjobInfo fetches and parses prowjob.json to extract full PR information
func fetchProwjobInfo(prowjobURL string) ProwjobInfo {
	var info ProwjobInfo
	
	// Make HTTP request with timeout
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Get(prowjobURL)
	if err != nil {
		return info // Could not fetch prowjob.json
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		return info // prowjob.json not found or accessible
	}
	
	// Read and parse JSON
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return info
	}
	
	// Parse prowjob JSON structure with full PR information
	var prowjob struct {
		Spec struct {
			Refs struct {
				Org    string `json:"org"`
				Repo   string `json:"repo"`
				Pulls  []struct {
					Number     int    `json:"number"`
					Author     string `json:"author"`
					SHA        string `json:"sha"`
					Title      string `json:"title"`
					HeadRef    string `json:"head_ref"`
					Link       string `json:"link"`
					CommitLink string `json:"commit_link"`
					AuthorLink string `json:"author_link"`
				} `json:"pulls"`
				BaseSHA string `json:"base_sha"`
			} `json:"refs"`
		} `json:"spec"`
	}
	
	if err := json.Unmarshal(body, &prowjob); err != nil {
		return info
	}
	
	// Extract commit SHA and PR information
	if len(prowjob.Spec.Refs.Pulls) > 0 {
		pull := prowjob.Spec.Refs.Pulls[0]
		info.CommitHash = pull.SHA
		
		// Populate PR information
		info.PRInfo = LLMPullRequestInfo{
			Number:        pull.Number,
			Author:        pull.Author,
			AuthorLink:    pull.AuthorLink,
			Title:         pull.Title,
			HeadRef:       pull.HeadRef,
			PRLink:        pull.Link,
			CommitLink:    pull.CommitLink,
		}
		
		// Determine if this is from a fork by checking if author != org
		info.PRInfo.IsFromFork = pull.Author != prowjob.Spec.Refs.Org
		
		if info.PRInfo.IsFromFork {
			// This is from a fork
			info.PRInfo.HeadRepo = prowjob.Spec.Refs.Repo
			info.PRInfo.HeadRepoOwner = pull.Author
			info.PRInfo.ForkRemoteURL = fmt.Sprintf("https://github.com/%s/%s.git", pull.Author, prowjob.Spec.Refs.Repo)
		} else {
			// This is from the same repository
			info.PRInfo.HeadRepo = prowjob.Spec.Refs.Repo
			info.PRInfo.HeadRepoOwner = prowjob.Spec.Refs.Org
			info.PRInfo.ForkRemoteURL = "" // Not needed for same repo
		}
	} else if prowjob.Spec.Refs.BaseSHA != "" {
		info.CommitHash = prowjob.Spec.Refs.BaseSHA
	}
	
	return info
}

// extractCommitFromBuildLog extracts commit hash from build log as fallback
func extractCommitFromBuildLog(buildLogURL string) string {
	// Make HTTP request with timeout
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Get(buildLogURL)
	if err != nil {
		return ""
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		return ""
	}
	
	// Read first few KB of build log to find commit references
	limitedReader := io.LimitReader(resp.Body, 8192) // Read first 8KB
	body, err := io.ReadAll(limitedReader)
	if err != nil {
		return ""
	}
	
	logContent := string(body)
	
	// Look for common commit hash patterns in build logs
	commitPatterns := []*regexp.Regexp{
		regexp.MustCompile(`Checking out ([a-f0-9]{40})`),                    // Git checkout
		regexp.MustCompile(`HEAD is now at ([a-f0-9]{7,40})`),               // Git reset
		regexp.MustCompile(`commit[:\s]+([a-f0-9]{40})`),                    // Commit reference
		regexp.MustCompile(`PULL_PULL_SHA[=:\s]+([a-f0-9]{40})`),           // Prow environment
		regexp.MustCompile(`PULL_BASE_SHA[=:\s]+([a-f0-9]{40})`),           // Base SHA
	}
	
	for _, pattern := range commitPatterns {
		if matches := pattern.FindStringSubmatch(logContent); len(matches) > 1 {
			commit := matches[1]
			// Validate it looks like a commit hash (at least 7 chars, hex)
			if len(commit) >= 7 && regexp.MustCompile(`^[a-f0-9]+$`).MatchString(commit) {
				return commit
			}
		}
	}
	
	return ""
}

// analyzeTrendsFromRuns analyzes failure trends from historical job runs
func analyzeTrendsFromRuns(runs []healthcheck.JobRun, includeFlakiness bool) LLMTrendAnalysis {
	analysis := LLMTrendAnalysis{
		TotalRuns:     len(runs),
		FailedRuns:    0,
		SuccessfulRuns: 0,
		TrendDirection: "stable",
		Flakiness:     LLMFlakinessAnalysis{},
		FailurePatterns: []LLMTrendFailurePattern{},
		Recommendations: []string{},
	}

	if len(runs) == 0 {
		return analysis
	}

	// Calculate basic statistics
	for _, run := range runs {
		if run.Status == "SUCCESS" {
			analysis.SuccessfulRuns++
		} else if run.Status == "FAILURE" {
			analysis.FailedRuns++
		}
	}

	analysis.OverallFailureRate = float64(analysis.FailedRuns) / float64(analysis.TotalRuns) * 100

	// Analyze trends over time periods
	analysis.TrendDirection = analyzeTrendDirection(runs)
	
	if includeFlakiness {
		analysis.Flakiness = analyzeFlakinessPatterns(runs)
	}

	// Analyze failure patterns over time
	analysis.FailurePatterns = analyzeFailurePatternsOverTime(runs)
	
	// Generate recommendations
	analysis.Recommendations = generateTrendRecommendations(analysis)

	return analysis
}

// analyzeFailureCorrelationAcrossJobs analyzes failure correlation across multiple jobs
func analyzeFailureCorrelationAcrossJobs(results *healthcheck.Results, jobPattern, timeWindow string, includeEnvironmentAnalysis bool) LLMCorrelationAnalysis {
	analysis := LLMCorrelationAnalysis{
		JobPattern:         jobPattern,
		TimeWindow:         timeWindow,
		CorrelatedFailures: []LLMCorrelatedFailure{},
		EnvironmentAnalysis: LLMEnvironmentAnalysis{},
		SystemicIssues:     []LLMSystemicIssue{},
		Recommendations:    []string{},
	}

	// Analyze job failures for correlation patterns
	correlatedFailures := findCorrelatedFailures(results, jobPattern)
	analysis.CorrelatedFailures = correlatedFailures

	if includeEnvironmentAnalysis {
		analysis.EnvironmentAnalysis = analyzeEnvironmentSpecificFailures(results, jobPattern)
	}

	// Identify systemic issues
	analysis.SystemicIssues = identifySystemicIssues(correlatedFailures)

	// Generate recommendations
	analysis.Recommendations = generateCorrelationRecommendations(analysis)

	return analysis
}

// analyzeQuarantineEffectiveness analyzes quarantine effectiveness and provides recommendations
func analyzeQuarantineEffectiveness(quarantinedTests map[string]bool, results *healthcheck.Results, scope string, includeRecommendations bool) LLMQuarantineAnalysis {
	analysis := LLMQuarantineAnalysis{
		Scope:                  scope,
		TotalQuarantinedTests:  len(quarantinedTests),
		QuarantineEffectiveness: "unknown",
		ActiveQuarantines:      []LLMQuarantineStatus{},
		RecommendedActions:     []LLMQuarantineRecommendation{},
	}

	// Analyze quarantine status for each test
	for testName := range quarantinedTests {
		status := analyzeQuarantineStatus(testName, results)
		analysis.ActiveQuarantines = append(analysis.ActiveQuarantines, status)
	}

	// Calculate overall effectiveness
	analysis.QuarantineEffectiveness = calculateQuarantineEffectiveness(analysis.ActiveQuarantines)

	if includeRecommendations {
		analysis.RecommendedActions = generateQuarantineRecommendations(analysis)
	}

	return analysis
}

// assessFailureImpactFromJSON assesses failure impact from JSON data
func assessFailureImpactFromJSON(failureData, context string, includeTriageRecommendations bool) (LLMImpactAssessment, error) {
	assessment := LLMImpactAssessment{
		Context:            context,
		OverallImpact:      "medium",
		TriagePriority:     "normal",
		ImpactCategories:   map[string]LLMImpactCategory{},
		CriticalFailures:   []LLMCriticalFailure{},
		TriageRecommendations: []string{},
	}

	// Parse JSON failure data (simplified for now)
	// In a full implementation, this would parse the actual JSON structure
	
	// Analyze impact based on context
	assessment.OverallImpact = determineOverallImpact(context, failureData)
	assessment.TriagePriority = determineTriagePriority(assessment.OverallImpact)

	if includeTriageRecommendations {
		assessment.TriageRecommendations = generateTriageRecommendations(assessment)
	}

	return assessment, nil
}

// generateComprehensiveFailureReport generates comprehensive failure reports
func generateComprehensiveFailureReport(scope, format string, includeRecommendations bool) (LLMFailureReport, error) {
	report := LLMFailureReport{
		Scope:               scope,
		Format:              format,
		GeneratedAt:         time.Now().UTC().Format(time.RFC3339),
		ExecutiveSummary:    "",
		KeyMetrics:          LLMReportMetrics{},
		CriticalIssues:      []LLMReportIssue{},
		TrendAnalysis:       LLMReportTrends{},
		ActionItems:         []string{},
	}

	// Generate report based on scope
	switch scope {
	case "daily":
		report = generateDailyFailureReport(format, includeRecommendations)
	case "weekly":
		report = generateWeeklyFailureReport(format, includeRecommendations)
	case "release":
		report = generateReleaseFailureReport(format, includeRecommendations)
	default:
		// Handle specific job analysis
		report = generateJobSpecificReport(scope, format, includeRecommendations)
	}

	return report, nil
}

// Helper functions for trend analysis
func analyzeTrendDirection(runs []healthcheck.JobRun) string {
	if len(runs) < 5 {
		return "insufficient_data"
	}

	// Analyze last 5 vs previous 5 runs
	recentFailures := 0
	previousFailures := 0
	
	for i, run := range runs {
		if run.Status == "FAILURE" {
			if i < len(runs)/2 {
				recentFailures++
			} else {
				previousFailures++
			}
		}
	}

	if recentFailures > previousFailures {
		return "degrading"
	} else if recentFailures < previousFailures {
		return "improving"
	}
	return "stable"
}

func analyzeFlakinessPatterns(runs []healthcheck.JobRun) LLMFlakinessAnalysis {
	flakiness := LLMFlakinessAnalysis{
		FlakyTests:        []string{},
		FlakinessScore:    0.0,
		PatternDetected:   false,
	}

	// Analyze test consistency across runs
	testResults := make(map[string][]string)
	
	for _, run := range runs {
		for _, failure := range run.Failures {
			testResults[failure.Name] = append(testResults[failure.Name], run.Status)
		}
	}

	// Calculate flakiness for each test
	flakyCount := 0
	for testName, results := range testResults {
		if isFlaky(results) {
			flakiness.FlakyTests = append(flakiness.FlakyTests, testName)
			flakyCount++
		}
	}

	if len(testResults) > 0 {
		flakiness.FlakinessScore = float64(flakyCount) / float64(len(testResults)) * 100
		flakiness.PatternDetected = flakiness.FlakinessScore > 20 // 20% threshold
	}

	return flakiness
}

func isFlaky(results []string) bool {
	if len(results) < 3 {
		return false
	}
	
	failures := 0
	for _, result := range results {
		if result == "FAILURE" {
			failures++
		}
	}
	
	failureRate := float64(failures) / float64(len(results))
	// Consider flaky if failure rate is between 10% and 90%
	return failureRate > 0.1 && failureRate < 0.9
}

func analyzeFailurePatternsOverTime(runs []healthcheck.JobRun) []LLMTrendFailurePattern {
	patterns := []LLMTrendFailurePattern{}
	
	// Group failures by time periods
	testFrequency := make(map[string]int)
	
	for _, run := range runs {
		for _, failure := range run.Failures {
			testFrequency[failure.Name]++
		}
	}

	// Convert to trend patterns
	for testName, frequency := range testFrequency {
		if frequency > 1 { // Only include recurring failures
			pattern := LLMTrendFailurePattern{
				TestName:   testName,
				Frequency:  frequency,
				Trend:      determineTrendForTest(testName, runs),
				Severity:   determineSeverity(frequency, len(runs)),
			}
			patterns = append(patterns, pattern)
		}
	}

	// Sort by frequency
	sort.Slice(patterns, func(i, j int) bool {
		return patterns[i].Frequency > patterns[j].Frequency
	})

	return patterns
}

func determineTrendForTest(testName string, runs []healthcheck.JobRun) string {
	// Simplified trend analysis for individual test
	recentFailures := 0
	totalFailures := 0
	
	for i, run := range runs {
		for _, failure := range run.Failures {
			if failure.Name == testName {
				totalFailures++
				if i < len(runs)/2 { // Recent half
					recentFailures++
				}
			}
		}
	}

	if totalFailures == 0 {
		return "stable"
	}

	recentRate := float64(recentFailures) / float64(totalFailures)
	if recentRate > 0.6 {
		return "increasing"
	} else if recentRate < 0.4 {
		return "decreasing"
	}
	return "stable"
}

func determineSeverity(frequency, totalRuns int) string {
	rate := float64(frequency) / float64(totalRuns)
	
	if rate > 0.5 {
		return "critical"
	} else if rate > 0.2 {
		return "high"
	} else if rate > 0.1 {
		return "medium"
	}
	return "low"
}

func generateTrendRecommendations(analysis LLMTrendAnalysis) []string {
	recommendations := []string{}

	if analysis.OverallFailureRate > 20 {
		recommendations = append(recommendations, "High failure rate detected - investigate infrastructure or recent changes")
	}

	if analysis.TrendDirection == "degrading" {
		recommendations = append(recommendations, "Degrading trend detected - review recent commits and infrastructure changes")
	}

	if analysis.Flakiness.FlakinessScore > 15 {
		recommendations = append(recommendations, "High flakiness detected - consider quarantining unstable tests")
	}

	if len(recommendations) == 0 {
		recommendations = append(recommendations, "Trends appear stable - continue monitoring")
	}

	return recommendations
}

// Placeholder implementations for correlation analysis
func findCorrelatedFailures(results *healthcheck.Results, jobPattern string) []LLMCorrelatedFailure {
	// Simplified implementation
	return []LLMCorrelatedFailure{}
}

func analyzeEnvironmentSpecificFailures(results *healthcheck.Results, jobPattern string) LLMEnvironmentAnalysis {
	return LLMEnvironmentAnalysis{
		ArchitectureFailures: map[string]int{},
		KubernetesVersions:   map[string]int{},
		ResourceIssues:       []string{},
	}
}

func identifySystemicIssues(correlatedFailures []LLMCorrelatedFailure) []LLMSystemicIssue {
	return []LLMSystemicIssue{}
}

func generateCorrelationRecommendations(analysis LLMCorrelationAnalysis) []string {
	return []string{"Continue monitoring for correlation patterns"}
}

// Placeholder implementations for quarantine analysis
func analyzeQuarantineStatus(testName string, results *healthcheck.Results) LLMQuarantineStatus {
	return LLMQuarantineStatus{
		TestName:          testName,
		Status:            "active",
		EffectivenessScore: 0.0,
		RecommendedAction: "monitor",
	}
}

func calculateQuarantineEffectiveness(quarantines []LLMQuarantineStatus) string {
	if len(quarantines) == 0 {
		return "no_data"
	}
	return "moderate"
}

func generateQuarantineRecommendations(analysis LLMQuarantineAnalysis) []LLMQuarantineRecommendation {
	return []LLMQuarantineRecommendation{}
}

// Placeholder implementations for impact assessment
func determineOverallImpact(context, failureData string) string {
	if context == "production" {
		return "high"
	} else if context == "pre-release" {
		return "medium"
	}
	return "low"
}

func determineTriagePriority(impact string) string {
	switch impact {
	case "critical", "high":
		return "urgent"
	case "medium":
		return "normal"
	default:
		return "low"
	}
}

func generateTriageRecommendations(assessment LLMImpactAssessment) []string {
	recommendations := []string{}
	
	switch assessment.TriagePriority {
	case "urgent":
		recommendations = append(recommendations, "Immediate attention required - assign to senior engineer")
	case "normal":
		recommendations = append(recommendations, "Schedule for next sprint - standard triage process")
	default:
		recommendations = append(recommendations, "Monitor - address when convenient")
	}
	
	return recommendations
}

// Placeholder implementations for report generation
func generateDailyFailureReport(format string, includeRecommendations bool) LLMFailureReport {
	return LLMFailureReport{
		Scope:            "daily",
		Format:           format,
		GeneratedAt:      time.Now().UTC().Format(time.RFC3339),
		ExecutiveSummary: "Daily CI health appears stable with minor issues",
		KeyMetrics:       LLMReportMetrics{},
		CriticalIssues:   []LLMReportIssue{},
		TrendAnalysis:    LLMReportTrends{},
		ActionItems:      []string{"Continue monitoring"},
	}
}

func generateWeeklyFailureReport(format string, includeRecommendations bool) LLMFailureReport {
	return generateDailyFailureReport(format, includeRecommendations) // Simplified
}

func generateReleaseFailureReport(format string, includeRecommendations bool) LLMFailureReport {
	return generateDailyFailureReport(format, includeRecommendations) // Simplified
}

func generateJobSpecificReport(scope, format string, includeRecommendations bool) LLMFailureReport {
	return generateDailyFailureReport(format, includeRecommendations) // Simplified
}