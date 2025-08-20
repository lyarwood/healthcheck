package mcp

import (
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"healthcheck/pkg/healthcheck"
)

// LLMJobAnalysis represents job analysis data optimized for LLM consumption
type LLMJobAnalysis struct {
	JobName     string                    `json:"job_name"`
	TimeRange   LLMTimeRange             `json:"time_range"`
	Statistics  LLMRunStatistics         `json:"statistics"`
	HealthStatus string                   `json:"health_status"`
	TopFailures []LLMFailurePattern      `json:"top_failures,omitempty"`
	Trends      LLMTrends                `json:"trends"`
	Categories  map[string]LLMCategory   `json:"failure_categories,omitempty"`
	Summary     string                   `json:"summary"`
}

type LLMTimeRange struct {
	Start    string `json:"start"`
	End      string `json:"end"`
	Duration string `json:"duration"`
	Period   string `json:"period"`
}

type LLMRunStatistics struct {
	TotalRuns     int     `json:"total_runs"`
	Successful    int     `json:"successful"`
	Failed        int     `json:"failed"`
	Unknown       int     `json:"unknown"`
	FailureRate   float64 `json:"failure_rate_percent"`
	TotalFailures int     `json:"total_test_failures"`
	UniqueTests   int     `json:"unique_failing_tests"`
}

type LLMFailurePattern struct {
	TestName         string   `json:"test_name"`
	FailureCount     int      `json:"failure_count"`
	Percentage       float64  `json:"percentage"`
	Category         string   `json:"category"`
	FirstSeen        string   `json:"first_seen,omitempty"`
	LastSeen         string   `json:"last_seen,omitempty"`
	SampleStackTrace string   `json:"sample_stack_trace,omitempty"`
	PotentialCauses  []string `json:"potential_causes,omitempty"`
}

type LLMTrends struct {
	IsImproving        bool   `json:"is_improving"`
	RegressionDetected bool   `json:"regression_detected"`
	Stability          string `json:"stability"`
	Recommendation     string `json:"recommendation"`
}

type LLMCategory struct {
	Count      int     `json:"count"`
	Percentage float64 `json:"percentage"`
	Examples   []string `json:"examples"`
}

type LLMJobFailures struct {
	JobName string           `json:"job_name"`
	Runs    []LLMJobRun      `json:"runs"`
	Summary LLMFailureSummary `json:"summary"`
}

type LLMJobRun struct {
	ID            string             `json:"run_id"`
	URL           string             `json:"url"`
	Status        string             `json:"status"`
	Timestamp     string             `json:"timestamp"`
	FailureCount  int                `json:"failure_count"`
	Failures      []LLMTestFailure   `json:"failures,omitempty"`
}

type LLMTestFailure struct {
	TestName     string `json:"test_name"`
	Category     string `json:"category"`
	StackTrace   string `json:"stack_trace,omitempty"`
	ErrorMessage string `json:"error_message,omitempty"`
	Duration     string `json:"duration,omitempty"`
}

type LLMFailureSummary struct {
	TotalRuns        int                    `json:"total_runs"`
	FailedRuns       int                    `json:"failed_runs"`
	CommonFailures   []LLMFailurePattern    `json:"common_failures"`
	FailuresByRun    map[string]int         `json:"failures_by_run"`
}

type LLMMergeAnalysis struct {
	Filter      LLMFilter                  `json:"filter"`
	Statistics  LLMMergeStatistics         `json:"statistics"`
	TopFailures []LLMFailurePattern        `json:"top_failures"`
	ByJob       map[string]LLMJobSummary   `json:"by_job"`
	Categories  map[string]LLMCategory     `json:"categories"`
	Summary     string                     `json:"summary"`
}

type LLMFilter struct {
	JobFilter  string `json:"job_filter"`
	TestFilter string `json:"test_filter"`
}

type LLMMergeStatistics struct {
	TotalFailures  int `json:"total_failures"`
	UniqueTests    int `json:"unique_tests"`
	AffectedJobs   int `json:"affected_jobs"`
	QuarantinedTests int `json:"quarantined_tests"`
}

type LLMJobSummary struct {
	FailureCount int      `json:"failure_count"`
	TestNames    []string `json:"test_names"`
}

type LLMPatternSearch struct {
	Pattern     string                 `json:"pattern"`
	SearchIn    string                 `json:"search_in"`
	Matches     []LLMPatternMatch      `json:"matches"`
	Statistics  LLMPatternStatistics   `json:"statistics"`
	Summary     string                 `json:"summary"`
}

type LLMPatternMatch struct {
	TestName  string `json:"test_name"`
	JobName   string `json:"job_name"`
	URL       string `json:"url"`
	Context   string `json:"context,omitempty"`
}

type LLMPatternStatistics struct {
	TotalMatches  int                    `json:"total_matches"`
	UniqueTests   int                    `json:"unique_tests"`
	AffectedJobs  int                    `json:"affected_jobs"`
	ByJob         map[string]int         `json:"by_job"`
}

type LLMTimeComparison struct {
	JobName         string            `json:"job_name"`
	RecentPeriod    LLMPeriodAnalysis `json:"recent_period"`
	ComparisonPeriod LLMPeriodAnalysis `json:"comparison_period"`
	Changes         LLMChanges        `json:"changes"`
	Analysis        string            `json:"analysis"`
}

type LLMPeriodAnalysis struct {
	Period      string                  `json:"period"`
	TimeRange   LLMTimeRange           `json:"time_range"`
	Statistics  LLMRunStatistics       `json:"statistics"`
	TopFailures []LLMFailurePattern    `json:"top_failures"`
}

type LLMChanges struct {
	FailureRateChange    float64 `json:"failure_rate_change_percent"`
	NewFailures         []string `json:"new_failures"`
	ResolvedFailures    []string `json:"resolved_failures"`
	WorsedFailures      []string `json:"worsened_failures"`
	ImprovedFailures    []string `json:"improved_failures"`
	OverallTrend        string   `json:"overall_trend"`
}

type LLMFailureSourceContext struct {
	FailureInfo    LLMFailureInfo    `json:"failure_info"`
	RepositoryInfo LLMRepositoryInfo `json:"repository_info"`
	SourceContext  []LLMSourceFile   `json:"source_context"`
	Summary        string            `json:"summary"`
}

type LLMFailureInfo struct {
	TestName      string               `json:"test_name"`
	FailureType   string               `json:"failure_type"`
	ErrorMessage  string               `json:"error_message"`
	PrimaryFile   string               `json:"primary_file"`
	PrimaryLine   int                  `json:"primary_line"`
	StackTrace    []LLMStackTraceFrame `json:"stack_trace,omitempty"`
}

type LLMStackTraceFrame struct {
	Function string `json:"function"`
	File     string `json:"file"`
	Line     int    `json:"line"`
	Context  string `json:"context,omitempty"`
}

type LLMRepositoryInfo struct {
	Repository string `json:"repository"`
	Owner      string `json:"owner"`
	Commit     string `json:"commit"`
	Branch     string `json:"branch,omitempty"`
	PullRequest int   `json:"pull_request,omitempty"`
}

type LLMSourceFile struct {
	FilePath     string `json:"file_path"`
	LineNumber   int    `json:"line_number"`
	GitHubURL    string `json:"github_url"`
	RawURL       string `json:"raw_url"`
	Context      string `json:"context"`
	FileType     string `json:"file_type"`
}

// New data structures for extended MCP tools

type LLMTrendAnalysis struct {
	JobName            string                     `json:"job_name"`
	TrendPeriod        string                     `json:"trend_period"`
	TotalRuns          int                        `json:"total_runs"`
	FailedRuns         int                        `json:"failed_runs"`
	SuccessfulRuns     int                        `json:"successful_runs"`
	OverallFailureRate float64                    `json:"overall_failure_rate"`
	TrendDirection     string                     `json:"trend_direction"`
	Flakiness          LLMFlakinessAnalysis       `json:"flakiness_analysis"`
	FailurePatterns    []LLMTrendFailurePattern   `json:"failure_patterns"`
	Recommendations    []string                   `json:"recommendations"`
}

type LLMFlakinessAnalysis struct {
	FlakyTests      []string `json:"flaky_tests"`
	FlakinessScore  float64  `json:"flakiness_score"`
	PatternDetected bool     `json:"pattern_detected"`
}

type LLMTrendFailurePattern struct {
	TestName  string `json:"test_name"`
	Frequency int    `json:"frequency"`
	Trend     string `json:"trend"`
	Severity  string `json:"severity"`
}

type LLMCorrelationAnalysis struct {
	JobPattern          string                  `json:"job_pattern"`
	TimeWindow          string                  `json:"time_window"`
	CorrelatedFailures  []LLMCorrelatedFailure  `json:"correlated_failures"`
	EnvironmentAnalysis LLMEnvironmentAnalysis  `json:"environment_analysis"`
	SystemicIssues      []LLMSystemicIssue      `json:"systemic_issues"`
	Recommendations     []string                `json:"recommendations"`
}

type LLMCorrelatedFailure struct {
	TestName         string   `json:"test_name"`
	AffectedJobs     []string `json:"affected_jobs"`
	CorrelationScore float64  `json:"correlation_score"`
	Pattern          string   `json:"pattern"`
}

type LLMEnvironmentAnalysis struct {
	ArchitectureFailures map[string]int `json:"architecture_failures"`
	KubernetesVersions   map[string]int `json:"kubernetes_versions"`
	ResourceIssues       []string       `json:"resource_issues"`
}

type LLMSystemicIssue struct {
	IssueType    string   `json:"issue_type"`
	Description  string   `json:"description"`
	AffectedJobs []string `json:"affected_jobs"`
	Severity     string   `json:"severity"`
}

type LLMQuarantineAnalysis struct {
	Scope                   string                        `json:"scope"`
	TotalQuarantinedTests   int                           `json:"total_quarantined_tests"`
	QuarantineEffectiveness string                        `json:"quarantine_effectiveness"`
	ActiveQuarantines       []LLMQuarantineStatus         `json:"active_quarantines"`
	RecommendedActions      []LLMQuarantineRecommendation `json:"recommended_actions"`
}

type LLMQuarantineStatus struct {
	TestName           string  `json:"test_name"`
	Status             string  `json:"status"`
	EffectivenessScore float64 `json:"effectiveness_score"`
	RecommendedAction  string  `json:"recommended_action"`
}

type LLMQuarantineRecommendation struct {
	TestName    string `json:"test_name"`
	Action      string `json:"action"`
	Reasoning   string `json:"reasoning"`
	Priority    string `json:"priority"`
}

type LLMImpactAssessment struct {
	Context               string                     `json:"context"`
	OverallImpact         string                     `json:"overall_impact"`
	TriagePriority        string                     `json:"triage_priority"`
	ImpactCategories      map[string]LLMImpactCategory `json:"impact_categories"`
	CriticalFailures      []LLMCriticalFailure       `json:"critical_failures"`
	TriageRecommendations []string                   `json:"triage_recommendations"`
}

type LLMImpactCategory struct {
	Category        string  `json:"category"`
	ImpactLevel     string  `json:"impact_level"`
	FailureCount    int     `json:"failure_count"`
	BusinessImpact  string  `json:"business_impact"`
}

type LLMCriticalFailure struct {
	TestName        string `json:"test_name"`
	ImpactLevel     string `json:"impact_level"`
	Frequency       int    `json:"frequency"`
	BusinessImpact  string `json:"business_impact"`
	RecommendedAction string `json:"recommended_action"`
}

type LLMFailureReport struct {
	Scope            string           `json:"scope"`
	Format           string           `json:"format"`
	GeneratedAt      string           `json:"generated_at"`
	ExecutiveSummary string           `json:"executive_summary"`
	KeyMetrics       LLMReportMetrics `json:"key_metrics"`
	CriticalIssues   []LLMReportIssue `json:"critical_issues"`
	TrendAnalysis    LLMReportTrends  `json:"trend_analysis"`
	ActionItems      []string         `json:"action_items"`
}

type LLMReportMetrics struct {
	TotalJobs       int     `json:"total_jobs"`
	FailingJobs     int     `json:"failing_jobs"`
	OverallHealth   string  `json:"overall_health"`
	FailureRate     float64 `json:"failure_rate"`
	CriticalIssues  int     `json:"critical_issues"`
}

type LLMReportIssue struct {
	IssueType   string   `json:"issue_type"`
	Description string   `json:"description"`
	Severity    string   `json:"severity"`
	Impact      string   `json:"impact"`
	Actions     []string `json:"recommended_actions"`
}

type LLMReportTrends struct {
	Direction       string `json:"direction"`
	ChangePercent   float64 `json:"change_percent"`
	KeyChanges      []string `json:"key_changes"`
}

// formatLaneSummaryForLLM converts lane summary to LLM-optimized format
func formatLaneSummaryForLLM(jobName string, summary *healthcheck.LaneSummary, includeDetails bool) LLMJobAnalysis {
	analysis := LLMJobAnalysis{
		JobName: jobName,
		TimeRange: LLMTimeRange{
			Start:    summary.FirstRunTime,
			End:      summary.LastRunTime,
			Duration: calculateHumanDuration(summary.FirstRunTime, summary.LastRunTime),
			Period:   inferPeriod(summary.FirstRunTime, summary.LastRunTime),
		},
		Statistics: LLMRunStatistics{
			TotalRuns:     summary.TotalRuns,
			Successful:    summary.SuccessfulRuns,
			Failed:        summary.FailedRuns,
			Unknown:       summary.TotalRuns - summary.SuccessfulRuns - summary.FailedRuns,
			FailureRate:   summary.FailureRate,
			TotalFailures: len(summary.AllFailures),
			UniqueTests:   len(summary.TestFailures),
		},
		HealthStatus: determineHealthStatus(summary.FailureRate),
		Trends: LLMTrends{
			IsImproving:        summary.FailureRate < 20,
			RegressionDetected: summary.FailureRate > 50,
			Stability:          determineStability(summary.FailureRate),
			Recommendation:     generateRecommendation(summary),
		},
		Summary: generateAnalysisSummary(jobName, summary),
	}

	if includeDetails {
		// Add top failures
		analysis.TopFailures = make([]LLMFailurePattern, 0, len(summary.TopFailures))
		for _, failure := range summary.TopFailures {
			pattern := LLMFailurePattern{
				TestName:        failure.TestName,
				FailureCount:    failure.Count,
				Percentage:      failure.Percentage,
				Category:        failure.Category,
				PotentialCauses: inferPotentialCauses(failure.TestName),
			}
			analysis.TopFailures = append(analysis.TopFailures, pattern)
		}

		// Add categories
		analysis.Categories = make(map[string]LLMCategory)
		categoryCount := make(map[string]int)
		categoryExamples := make(map[string][]string)
		
		for _, failure := range summary.TopFailures {
			categoryCount[failure.Category] += failure.Count
			if len(categoryExamples[failure.Category]) < 3 {
				categoryExamples[failure.Category] = append(categoryExamples[failure.Category], failure.TestName)
			}
		}

		for category, count := range categoryCount {
			percentage := float64(count) / float64(len(summary.AllFailures)) * 100
			analysis.Categories[category] = LLMCategory{
				Count:      count,
				Percentage: percentage,
				Examples:   categoryExamples[category],
			}
		}
	}

	return analysis
}

// formatJobFailuresForLLM converts job runs to LLM-optimized format
func formatJobFailuresForLLM(jobName string, runs []healthcheck.JobRun, includeStackTraces bool) LLMJobFailures {
	llmRuns := make([]LLMJobRun, 0, len(runs))
	totalFailures := 0
	failuresByRun := make(map[string]int)
	commonFailures := make(map[string]int)

	for _, run := range runs {
		failures := make([]LLMTestFailure, 0, len(run.Failures))
		for _, failure := range run.Failures {
			testFailure := LLMTestFailure{
				TestName:     failure.Name,
				Category:     categorizeTestName(failure.Name),
				ErrorMessage: extractErrorMessage(failure.Failure),
			}
			
			if includeStackTraces && failure.Failure != nil {
				testFailure.StackTrace = failure.Failure.Value
			}
			
			failures = append(failures, testFailure)
			commonFailures[failure.Name]++
		}

		llmRun := LLMJobRun{
			ID:           run.ID,
			URL:          run.URL,
			Status:       run.Status,
			Timestamp:    run.Timestamp,
			FailureCount: len(run.Failures),
			Failures:     failures,
		}
		
		llmRuns = append(llmRuns, llmRun)
		totalFailures += len(run.Failures)
		failuresByRun[run.ID] = len(run.Failures)
	}

	// Convert common failures to patterns
	patterns := make([]LLMFailurePattern, 0)
	for testName, count := range commonFailures {
		if count > 1 { // Only include tests that failed multiple times
			percentage := float64(count) / float64(totalFailures) * 100
			pattern := LLMFailurePattern{
				TestName:        testName,
				FailureCount:    count,
				Percentage:      percentage,
				Category:        categorizeTestName(testName),
				PotentialCauses: inferPotentialCauses(testName),
			}
			patterns = append(patterns, pattern)
		}
	}

	// Sort patterns by failure count
	sort.Slice(patterns, func(i, j int) bool {
		return patterns[i].FailureCount > patterns[j].FailureCount
	})

	failedRuns := 0
	for _, run := range runs {
		if run.Status == "FAILURE" {
			failedRuns++
		}
	}

	return LLMJobFailures{
		JobName: jobName,
		Runs:    llmRuns,
		Summary: LLMFailureSummary{
			TotalRuns:      len(runs),
			FailedRuns:     failedRuns,
			CommonFailures: patterns,
			FailuresByRun:  failuresByRun,
		},
	}
}

// formatMergeFailuresForLLM converts merge analysis to LLM-optimized format
func formatMergeFailuresForLLM(result *healthcheck.ProcessorResult, jobFilter, testFilter string) LLMMergeAnalysis {
	totalFailures := 0
	affectedJobs := make(map[string]bool)
	categories := make(map[string]int)
	categoryExamples := make(map[string][]string)
	
	patterns := make([]LLMFailurePattern, 0)
	byJob := make(map[string]LLMJobSummary)

	for testName, testcases := range result.FailedTests {
		count := len(testcases)
		totalFailures += count
		category := categorizeTestName(testName)
		categories[category] += count
		
		if len(categoryExamples[category]) < 3 {
			categoryExamples[category] = append(categoryExamples[category], testName)
		}

		// Extract job names from URLs
		jobNames := make(map[string]bool)
		for _, tc := range testcases {
			jobName := extractJobNameFromURL(tc.URL)
			if jobName != "" {
				jobNames[jobName] = true
				affectedJobs[jobName] = true
			}
		}

		pattern := LLMFailurePattern{
			TestName:        testName,
			FailureCount:    count,
			Percentage:      float64(count) / float64(totalFailures) * 100,
			Category:        category,
			PotentialCauses: inferPotentialCauses(testName),
		}
		patterns = append(patterns, pattern)

		// Update by-job summary
		for jobName := range jobNames {
			summary := byJob[jobName]
			summary.FailureCount += count
			summary.TestNames = append(summary.TestNames, testName)
			byJob[jobName] = summary
		}
	}

	// Sort patterns by failure count
	sort.Slice(patterns, func(i, j int) bool {
		return patterns[i].FailureCount > patterns[j].FailureCount
	})

	// Convert categories
	llmCategories := make(map[string]LLMCategory)
	for category, count := range categories {
		percentage := float64(count) / float64(totalFailures) * 100
		llmCategories[category] = LLMCategory{
			Count:      count,
			Percentage: percentage,
			Examples:   categoryExamples[category],
		}
	}

	return LLMMergeAnalysis{
		Filter: LLMFilter{
			JobFilter:  jobFilter,
			TestFilter: testFilter,
		},
		Statistics: LLMMergeStatistics{
			TotalFailures: totalFailures,
			UniqueTests:   len(result.FailedTests),
			AffectedJobs:  len(affectedJobs),
		},
		TopFailures: patterns,
		ByJob:       byJob,
		Categories:  llmCategories,
		Summary:     generateMergeSummary(totalFailures, len(result.FailedTests), len(affectedJobs)),
	}
}

// formatPatternSearchForLLM converts pattern search results to LLM-optimized format
func formatPatternSearchForLLM(pattern string, matches []LLMPatternMatch, searchIn string) LLMPatternSearch {
	statistics := LLMPatternStatistics{
		TotalMatches: len(matches),
		ByJob:        make(map[string]int),
	}

	uniqueTests := make(map[string]bool)
	affectedJobs := make(map[string]bool)

	for _, match := range matches {
		uniqueTests[match.TestName] = true
		affectedJobs[match.JobName] = true
		statistics.ByJob[match.JobName]++
	}

	statistics.UniqueTests = len(uniqueTests)
	statistics.AffectedJobs = len(affectedJobs)

	return LLMPatternSearch{
		Pattern:    pattern,
		SearchIn:   searchIn,
		Matches:    matches,
		Statistics: statistics,
		Summary:    generatePatternSearchSummary(pattern, statistics),
	}
}

// formatTimeComparisonForLLM converts time comparison to LLM-optimized format
func formatTimeComparisonForLLM(jobName string, recent, comparison *healthcheck.LaneSummary, recentPeriod, comparisonPeriod string) LLMTimeComparison {
	recentAnalysis := LLMPeriodAnalysis{
		Period: recentPeriod,
		TimeRange: LLMTimeRange{
			Start:    recent.FirstRunTime,
			End:      recent.LastRunTime,
			Duration: calculateHumanDuration(recent.FirstRunTime, recent.LastRunTime),
		},
		Statistics: LLMRunStatistics{
			TotalRuns:     recent.TotalRuns,
			Successful:    recent.SuccessfulRuns,
			Failed:        recent.FailedRuns,
			FailureRate:   recent.FailureRate,
			TotalFailures: len(recent.AllFailures),
			UniqueTests:   len(recent.TestFailures),
		},
	}

	comparisonAnalysis := LLMPeriodAnalysis{
		Period: comparisonPeriod,
		TimeRange: LLMTimeRange{
			Start:    comparison.FirstRunTime,
			End:      comparison.LastRunTime,
			Duration: calculateHumanDuration(comparison.FirstRunTime, comparison.LastRunTime),
		},
		Statistics: LLMRunStatistics{
			TotalRuns:     comparison.TotalRuns,
			Successful:    comparison.SuccessfulRuns,
			Failed:        comparison.FailedRuns,
			FailureRate:   comparison.FailureRate,
			TotalFailures: len(comparison.AllFailures),
			UniqueTests:   len(comparison.TestFailures),
		},
	}

	// Calculate changes
	failureRateChange := recent.FailureRate - comparison.FailureRate
	
	// Find new and resolved failures
	recentTests := make(map[string]bool)
	comparisonTests := make(map[string]bool)
	
	for testName := range recent.TestFailures {
		recentTests[testName] = true
	}
	for testName := range comparison.TestFailures {
		comparisonTests[testName] = true
	}

	var newFailures, resolvedFailures []string
	for testName := range recentTests {
		if !comparisonTests[testName] {
			newFailures = append(newFailures, testName)
		}
	}
	for testName := range comparisonTests {
		if !recentTests[testName] {
			resolvedFailures = append(resolvedFailures, testName)
		}
	}

	overallTrend := "stable"
	if failureRateChange > 10 {
		overallTrend = "worsening"
	} else if failureRateChange < -10 {
		overallTrend = "improving"
	}

	changes := LLMChanges{
		FailureRateChange: failureRateChange,
		NewFailures:      newFailures,
		ResolvedFailures: resolvedFailures,
		OverallTrend:     overallTrend,
	}

	return LLMTimeComparison{
		JobName:          jobName,
		RecentPeriod:     recentAnalysis,
		ComparisonPeriod: comparisonAnalysis,
		Changes:          changes,
		Analysis:         generateTimeComparisonAnalysis(jobName, failureRateChange, len(newFailures), len(resolvedFailures)),
	}
}

// Helper functions

func calculateHumanDuration(start, end string) string {
	if start == "" || end == "" {
		return "unknown"
	}

	startTime, err1 := time.Parse(time.RFC3339, start)
	endTime, err2 := time.Parse(time.RFC3339, end)
	
	if err1 != nil || err2 != nil {
		return "unknown"
	}

	duration := endTime.Sub(startTime)
	
	if duration < time.Hour {
		return strconv.Itoa(int(duration.Minutes())) + " minutes"
	} else if duration < 24*time.Hour {
		return strconv.FormatFloat(duration.Hours(), 'f', 1, 64) + " hours"
	} else {
		days := duration.Hours() / 24
		return strconv.FormatFloat(days, 'f', 1, 64) + " days"
	}
}

func inferPeriod(start, end string) string {
	duration := calculateHumanDuration(start, end)
	if strings.Contains(duration, "minutes") {
		return "< 1 hour"
	} else if strings.Contains(duration, "hours") {
		return "< 1 day"
	} else if strings.Contains(duration, "days") {
		return "> 1 day"
	}
	return "unknown"
}

func determineHealthStatus(failureRate float64) string {
	if failureRate > 80 {
		return "critical"
	} else if failureRate > 50 {
		return "unhealthy"
	} else if failureRate > 20 {
		return "unstable"
	} else if failureRate > 5 {
		return "acceptable"
	}
	return "healthy"
}

func determineStability(failureRate float64) string {
	if failureRate > 50 {
		return "unstable"
	} else if failureRate > 20 {
		return "moderate"
	}
	return "stable"
}

func categorizeTestName(testName string) string {
	testLower := strings.ToLower(testName)
	
	if strings.Contains(testLower, "network") || strings.Contains(testLower, "bridge") || 
	   strings.Contains(testLower, "masquerade") || strings.Contains(testLower, "sriov") {
		return "network"
	}
	if strings.Contains(testLower, "storage") || strings.Contains(testLower, "volume") || 
	   strings.Contains(testLower, "disk") || strings.Contains(testLower, "pvc") {
		return "storage"
	}
	if strings.Contains(testLower, "migration") || strings.Contains(testLower, "migrate") {
		return "migration"
	}
	if strings.Contains(testLower, "compute") || strings.Contains(testLower, "cpu") || 
	   strings.Contains(testLower, "memory") || strings.Contains(testLower, "lifecycle") {
		return "compute"
	}
	if strings.Contains(testLower, "operator") {
		return "operator"
	}
	
	return "general"
}

func inferPotentialCauses(testName string) []string {
	testLower := strings.ToLower(testName)
	var causes []string
	
	if strings.Contains(testLower, "timeout") {
		causes = append(causes, "timeout", "resource_contention", "slow_operations")
	}
	if strings.Contains(testLower, "migration") {
		causes = append(causes, "migration_timeout", "network_issues", "resource_shortage")
	}
	if strings.Contains(testLower, "connection") || strings.Contains(testLower, "network") {
		causes = append(causes, "network_connectivity", "dns_issues", "firewall_rules")
	}
	if strings.Contains(testLower, "memory") || strings.Contains(testLower, "oom") {
		causes = append(causes, "memory_pressure", "resource_limits", "memory_leak")
	}
	if strings.Contains(testLower, "disk") || strings.Contains(testLower, "storage") {
		causes = append(causes, "disk_space", "io_performance", "storage_backend_issues")
	}
	
	if len(causes) == 0 {
		causes = append(causes, "unknown", "investigate_logs")
	}
	
	return causes
}

func extractErrorMessage(failure *healthcheck.Failure) string {
	if failure == nil {
		return ""
	}
	
	lines := strings.Split(failure.Value, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" && !strings.HasPrefix(line, "goroutine") && !strings.HasPrefix(line, "	") {
			return line
		}
	}
	
	return failure.Value
}

func extractJobNameFromURL(url string) string {
	// Extract job name from Prow URL
	re := regexp.MustCompile(`/([^/]+)/\d+/?$`)
	matches := re.FindStringSubmatch(url)
	if len(matches) > 1 {
		return matches[1]
	}
	return ""
}

func generateAnalysisSummary(jobName string, summary *healthcheck.LaneSummary) string {
	status := determineHealthStatus(summary.FailureRate)
	
	return fmt.Sprintf("Job %s shows %s health with %.1f%% failure rate over %d runs. %d unique tests failed with %d total failures.",
		jobName, status, summary.FailureRate, summary.TotalRuns, len(summary.TestFailures), len(summary.AllFailures))
}

func generateRecommendation(summary *healthcheck.LaneSummary) string {
	if summary.FailureRate > 80 {
		return "Immediate investigation required - systemic issues detected"
	} else if summary.FailureRate > 50 {
		return "High priority investigation - significant instability"
	} else if summary.FailureRate > 20 {
		return "Monitor trends and investigate common failures"
	}
	return "Continue monitoring - acceptable failure rate"
}

func generateMergeSummary(totalFailures, uniqueTests, affectedJobs int) string {
	return fmt.Sprintf("Found %d total failures across %d unique tests affecting %d jobs",
		totalFailures, uniqueTests, affectedJobs)
}

func generatePatternSearchSummary(pattern string, stats LLMPatternStatistics) string {
	return fmt.Sprintf("Pattern '%s' found %d matches across %d unique tests in %d jobs",
		pattern, stats.TotalMatches, stats.UniqueTests, stats.AffectedJobs)
}

func generateTimeComparisonAnalysis(jobName string, failureRateChange float64, newFailures, resolvedFailures int) string {
	trend := "stable"
	if failureRateChange > 10 {
		trend = "worsening significantly"
	} else if failureRateChange > 0 {
		trend = "slightly worsening"
	} else if failureRateChange < -10 {
		trend = "improving significantly"
	} else if failureRateChange < 0 {
		trend = "slightly improving"
	}

	return fmt.Sprintf("Job %s is %s with %.1f%% failure rate change. %d new failures, %d resolved failures.",
		jobName, trend, failureRateChange, newFailures, resolvedFailures)
}

// FormatFailureSourceContextForLLM converts failure source context to LLM-optimized format
func FormatFailureSourceContextForLLM(failureInfo LLMFailureInfo, repoInfo LLMRepositoryInfo, includeStackTrace bool) LLMFailureSourceContext {
	sourceFiles := make([]LLMSourceFile, 0)

	// Add primary failure file
	if failureInfo.PrimaryFile != "" && failureInfo.PrimaryLine > 0 {
		sourceFile := createSourceFileEntry(failureInfo.PrimaryFile, failureInfo.PrimaryLine, repoInfo, "primary failure location")
		sourceFiles = append(sourceFiles, sourceFile)
	}

	// Add stack trace files if requested
	if includeStackTrace {
		for _, frame := range failureInfo.StackTrace {
			if frame.File != "" && frame.Line > 0 && !isSystemFile(frame.File) {
				sourceFile := createSourceFileEntry(frame.File, frame.Line, repoInfo, fmt.Sprintf("stack trace: %s", frame.Function))
				sourceFiles = append(sourceFiles, sourceFile)
			}
		}
	}

	// Generate summary
	summary := generateFailureSourceSummary(failureInfo, repoInfo, len(sourceFiles))

	return LLMFailureSourceContext{
		FailureInfo:    failureInfo,
		RepositoryInfo: repoInfo,
		SourceContext:  sourceFiles,
		Summary:        summary,
	}
}

// createSourceFileEntry creates a source file entry with GitHub URLs
func createSourceFileEntry(filePath string, lineNumber int, repoInfo LLMRepositoryInfo, context string) LLMSourceFile {
	// Clean file path (remove leading paths like bazel-out, etc.)
	cleanPath := cleanFilePath(filePath)
	
	// Generate GitHub URLs
	githubURL := fmt.Sprintf("https://github.com/%s/%s/blob/%s/%s#L%d", 
		repoInfo.Owner, repoInfo.Repository, repoInfo.Commit, cleanPath, lineNumber)
	rawURL := fmt.Sprintf("https://raw.githubusercontent.com/%s/%s/%s/%s", 
		repoInfo.Owner, repoInfo.Repository, repoInfo.Commit, cleanPath)

	// Determine file type
	fileType := determineFileType(cleanPath)

	return LLMSourceFile{
		FilePath:     cleanPath,
		LineNumber:   lineNumber,
		GitHubURL:    githubURL,
		RawURL:       rawURL,
		Context:      context,
		FileType:     fileType,
	}
}

// cleanFilePath removes bazel-out prefixes and other build artifacts from file paths
func cleanFilePath(filePath string) string {
	// Remove bazel-out prefix
	if strings.Contains(filePath, "bazel-out/") {
		parts := strings.Split(filePath, "bazel-out/")
		if len(parts) > 1 {
			// Look for the next meaningful part after bazel-out
			remaining := parts[1]
			// Skip the architecture and build type directories
			pathParts := strings.Split(remaining, "/")
			if len(pathParts) > 2 {
				// Rejoin from the third part onwards, usually starting with "bin" or similar
				if pathParts[2] == "bin" && len(pathParts) > 3 {
					return strings.Join(pathParts[3:], "/")
				}
			}
		}
	}
	
	// Handle other known prefixes
	prefixes := []string{
		"external/io_bazel_rules_go/stdlib_/src/",
		"./",
	}
	
	for _, prefix := range prefixes {
		if strings.HasPrefix(filePath, prefix) {
			return strings.TrimPrefix(filePath, prefix)
		}
	}
	
	return filePath
}

// isSystemFile checks if a file is part of system/runtime code that shouldn't be included
func isSystemFile(filePath string) bool {
	systemPrefixes := []string{
		"runtime/",
		"external/io_bazel_rules_go/stdlib_/src/runtime/",
		"/usr/",
		"/go/",
	}
	
	for _, prefix := range systemPrefixes {
		if strings.HasPrefix(filePath, prefix) {
			return true
		}
	}
	
	return false
}

// determineFileType determines the programming language/file type from extension
func determineFileType(filePath string) string {
	if strings.HasSuffix(filePath, ".go") {
		return "go"
	}
	if strings.HasSuffix(filePath, ".yaml") || strings.HasSuffix(filePath, ".yml") {
		return "yaml"
	}
	if strings.HasSuffix(filePath, ".json") {
		return "json"
	}
	if strings.HasSuffix(filePath, ".sh") {
		return "shell"
	}
	if strings.HasSuffix(filePath, ".py") {
		return "python"
	}
	return "text"
}

// generateFailureSourceSummary creates a human-readable summary
func generateFailureSourceSummary(failureInfo LLMFailureInfo, repoInfo LLMRepositoryInfo, sourceCount int) string {
	return fmt.Sprintf("Test '%s' failed with '%s' in %s/%s. Primary failure at %s:%d. %d source context files available for LLM analysis.",
		failureInfo.TestName, failureInfo.FailureType, repoInfo.Owner, repoInfo.Repository,
		failureInfo.PrimaryFile, failureInfo.PrimaryLine, sourceCount)
}