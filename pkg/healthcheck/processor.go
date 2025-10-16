package healthcheck

import (
	"fmt"
	"regexp"
	"sort"
	"strings"
	"time"
)

var JobRegexAliases = map[string]string{
	"main":        "sig-[a-zA-Z0-9_-]+$",
	"1.6":         "release-1.6$",
	"1.5":         "release-1.5$",
	"1.4":         "release-1.4$",
	"compute":     "sig-compute$|sig-compute-serial$|sig-compute-migrations$|sig-operator$|.*arm64.*",
	"compute-1.6": "sig-compute-1.6$|sig-compute-serial-1.6$|sig-compute-migrations1-.6$|sig-operator1.6$|.*arm64.*-1.6$",
	"network":     "sig-network$",
	"storage":     "sig-storage$",
}

type ProcessorConfig struct {
	JobRegex             *regexp.Regexp
	TestRegex            *regexp.Regexp
	DisplayOnlyURLs      bool
	DisplayOnlyTestNames bool
	DisplayFailures      bool
	CountFailures        bool
	GroupByLaneRun       bool
	CheckQuarantine      bool
	TimePeriod           time.Duration
	SuppressOutput       bool // Suppress all immediate output for JSON formatting
	Summary              bool
}

type ProcessorResult struct {
	FailedTests     map[string][]Testcase
	LaneRunFailures map[string][]Testcase
}

func ExtractLaneRunUUID(failureURL string) string {
	parts := strings.Split(failureURL, "/")
	if len(parts) > 0 {
		return parts[len(parts)-1]
	}
	return ""
}

func ProcessFailures(results *Results, config ProcessorConfig) (*ProcessorResult, error) {
	result := &ProcessorResult{
		FailedTests:     make(map[string][]Testcase),
		LaneRunFailures: make(map[string][]Testcase),
	}

	// Fetch quarantined tests if checking is enabled
	var quarantinedTests map[string]bool
	if config.CheckQuarantine {
		var err error
		quarantinedTests, err = FetchQuarantinedTests()
		if err != nil {
			// Don't fail the entire operation if quarantine check fails
			fmt.Printf("Warning: Failed to fetch quarantined tests: %v\n", err)
			quarantinedTests = make(map[string]bool)
		}
	}

	for _, job := range results.Data.SIGRetests.FailedJobLeaderBoard {
		if !config.JobRegex.MatchString(job.JobName) {
			continue
		}

		for _, failureURL := range job.FailureURLs {
			if err := processJobFailure(job, failureURL, config, result, quarantinedTests); err != nil {
				return nil, err
			}
		}
	}

	return result, nil
}

// fetchJobTypeFromURL fetches the job type from prowjob.json for a given failure URL
func fetchJobTypeFromURL(failureURL string) string {
	// Convert prow URL to prowjob.json URL
	prowjobURL := strings.Replace(failureURL, "prow.ci.kubevirt.io//view/gs", "storage.googleapis.com", 1)
	if !strings.HasSuffix(prowjobURL, "/") {
		prowjobURL += "/"
	}
	prowjobURL += "prowjob.json"

	// Fetch job info
	info, err := fetchProwJobInfo(prowjobURL)
	if err != nil {
		// If we can't fetch job type, return empty string
		return ""
	}

	return info.JobType
}

func processJobFailure(job Job, failureURL string, config ProcessorConfig,
	result *ProcessorResult, quarantinedTests map[string]bool) error {

	// Check time filter if specified
	if config.TimePeriod > 0 {
		timestamp := extractTimestampFromURL(failureURL)
		if timestamp != "" && !IsWithinTimePeriod(timestamp, config.TimePeriod) {
			return nil // Skip this failure as it's outside the time period
		}
	}

	// Fetch job type from prowjob.json
	jobType := fetchJobTypeFromURL(failureURL)

	testsuite, err := fetchTestSuite(failureURL)
	if err != nil {
		return err
	}

	if testsuite == nil {
		return handleMissingTestsuite(job, failureURL, jobType, config, result, quarantinedTests)
	}

	return processTestcases(testsuite, failureURL, jobType, config, result, quarantinedTests)
}

func handleMissingTestsuite(job Job, failureURL string, jobType string, config ProcessorConfig,
	result *ProcessorResult, _ map[string]bool) error {
	if config.DisplayOnlyURLs && !config.SuppressOutput {
		fmt.Println(failureURL)
		return nil
	}
	if config.DisplayOnlyTestNames && !config.SuppressOutput {
		fmt.Printf("%s (no junit file to parse)\n", job.JobName)
		return nil
	}
	if config.GroupByLaneRun {
		laneRunUUID := ExtractLaneRunUUID(failureURL)
		if laneRunUUID != "" {
			placeholder := Testcase{Name: fmt.Sprintf("%s (no junit file to parse)", job.JobName), URL: failureURL, JobType: jobType}
			result.LaneRunFailures[laneRunUUID] = append(result.LaneRunFailures[laneRunUUID], placeholder)
		}
		return nil
	}
	if !config.SuppressOutput {
		fmt.Printf("%s (no junit file to parse)\n", job.JobName)
		fmt.Printf("%s\n\n", failureURL)
	}

	// Always add placeholder testcase for missing junit files
	placeholder := Testcase{Name: fmt.Sprintf("%s (no junit file to parse)", job.JobName), URL: failureURL, JobType: jobType}
	result.FailedTests[placeholder.Name] = append(result.FailedTests[placeholder.Name], placeholder)

	return nil
}

func processTestcases(testsuite *Testsuite, failureURL string, jobType string, config ProcessorConfig,
	result *ProcessorResult, quarantinedTests map[string]bool) error {
	for _, testcase := range testsuite.Testcase {
		if testcase.Failure == nil || !config.TestRegex.MatchString(testcase.Name) {
			continue
		}

		if config.DisplayOnlyURLs && !config.SuppressOutput {
			fmt.Println(failureURL)
			continue
		}
		if config.DisplayOnlyTestNames && !config.SuppressOutput {
			fmt.Println(testcase.Name)
			continue
		}

		testcase.URL = failureURL
		testcase.JobType = jobType

		// Check if test is quarantined
		if config.CheckQuarantine && quarantinedTests != nil {
			testcase.IsQuarantined = isTestQuarantined(testcase.Name, quarantinedTests)
		}

		processTestcase(testcase, config, result)
	}
	return nil
}

func processTestcase(testcase Testcase, config ProcessorConfig, result *ProcessorResult) {
	if config.GroupByLaneRun {
		laneRunUUID := ExtractLaneRunUUID(testcase.URL)
		if laneRunUUID != "" {
			result.LaneRunFailures[laneRunUUID] = append(result.LaneRunFailures[laneRunUUID], testcase)
		}
		return
	}
	if config.CountFailures {
		result.FailedTests[testcase.Name] = append(result.FailedTests[testcase.Name], testcase)
		return
	}

	// Default output for non-count, non-grouped mode
	if !config.SuppressOutput {
		fmt.Println(testcase.Name)
		if config.DisplayFailures {
			fmt.Printf("%s\n\n", testcase.Failure)
		}
		fmt.Printf("%s\n\n", testcase.URL)
	}
	
	// Always add to result data for JSON output or other processing
	result.FailedTests[testcase.Name] = append(result.FailedTests[testcase.Name], testcase)
}

// isTestQuarantined checks if a test name matches any quarantined test
func isTestQuarantined(testName string, quarantinedTests map[string]bool) bool {
	// Direct match
	if quarantinedTests[testName] {
		return true
	}

	// Check for partial matches - test names in junit files often contain extra context
	for quarantinedName := range quarantinedTests {
		if strings.Contains(testName, quarantinedName) {
			return true
		}
	}

	return false
}

// AnalyzeLaneRuns processes job runs and creates a summary
func AnalyzeLaneRuns(runs []JobRun) (*LaneSummary, error) {
	summary := &LaneSummary{
		TotalRuns:          len(runs),
		TestFailures:       make(map[string]int),
		JobTypeStats:       make(map[string]int),
		JobTypeFailureRate: make(map[string]float64),
		Runs:               runs,
		AllFailures:        []Testcase{},
	}

	// Track failures per job type for calculating failure rates
	jobTypeFailures := make(map[string]int)

	// Analyze each job run
	for i := range runs {
		run := &runs[i]

		// Fetch artifacts and analyze failures
		if err := fetchJobArtifacts(run); err != nil {
			// Don't fail completely if one job fails to fetch
			continue
		}

		// Count job types
		if run.JobType != "" {
			summary.JobTypeStats[run.JobType]++
		}

		// Track if this run failed
		isFailed := false

		// Count status
		switch run.Status {
		case "SUCCESS":
			summary.SuccessfulRuns++
		case "FAILURE":
			summary.FailedRuns++
			isFailed = true
		case "ABORTED":
			summary.AbortedRuns++
			isFailed = true
		case "ERROR":
			summary.ErrorRuns++
			isFailed = true
		default:
			summary.UnknownRuns++
			isFailed = true
		}

		// Track failures per job type
		if isFailed && run.JobType != "" {
			jobTypeFailures[run.JobType]++
		}

		// Count test failures and collect all failures
		for _, failure := range run.Failures {
			summary.TestFailures[failure.Name]++
			summary.AllFailures = append(summary.AllFailures, failure)
		}
		
		// For infrastructure failures without test failures, create placeholder entries
		if run.Status != "SUCCESS" && len(run.Failures) == 0 {
			placeholderName := fmt.Sprintf("Infrastructure failure (%s)", run.Status)
			summary.TestFailures[placeholderName]++
			placeholder := Testcase{
				Name: placeholderName,
				URL:  run.URL,
			}
			summary.AllFailures = append(summary.AllFailures, placeholder)
		}
	}

	// Calculate failure rates
	if summary.TotalRuns > 0 {
		totalFailedRuns := summary.FailedRuns + summary.AbortedRuns + summary.ErrorRuns + summary.UnknownRuns
		summary.FailureRate = float64(totalFailedRuns) / float64(summary.TotalRuns) * 100
	}

	// Calculate failure rate per job type
	for jobType, totalCount := range summary.JobTypeStats {
		if totalCount > 0 {
			failedCount := jobTypeFailures[jobType]
			summary.JobTypeFailureRate[jobType] = float64(failedCount) / float64(totalCount) * 100
		}
	}

	// Analyze failure patterns
	summary.TopFailures = analyzeFailurePatterns(summary.TestFailures, len(summary.AllFailures))

	// Calculate infrastructure failure rate based on categorized failures
	infrastructureFailures := 0
	for _, failure := range summary.AllFailures {
		category := categorizeTest(failure.Name)
		if category == "infrastructure" || category == "infra-timeout" || category == "infra-error" {
			infrastructureFailures++
		}
	}
	
	if len(summary.AllFailures) > 0 {
		summary.InfrastructureFailureRate = float64(infrastructureFailures) / float64(len(summary.AllFailures)) * 100
	}

	// Calculate time range
	summary.FirstRunTime, summary.LastRunTime = calculateTimeRange(runs)

	return summary, nil
}

// extractTimestampFromURL attempts to extract a timestamp from a Prow URL
// For merge command URLs, we may need to fetch the job metadata to get the timestamp
func extractTimestampFromURL(url string) string {
	// For now, we can't easily extract timestamps from ci-health URLs
	// This would require fetching job metadata from Prow which is complex
	// We'll return empty string to include all results for merge command
	return ""
}

// FilterRunsByTimePeriod filters job runs by the given time period
func FilterRunsByTimePeriod(runs []JobRun, timePeriod time.Duration) []JobRun {
	if timePeriod == 0 {
		return runs
	}

	var filtered []JobRun
	for _, run := range runs {
		if IsWithinTimePeriod(run.Timestamp, timePeriod) {
			filtered = append(filtered, run)
		}
	}
	return filtered
}

// analyzeFailurePatterns analyzes test failures to identify patterns and categories
func analyzeFailurePatterns(testFailures map[string]int, totalFailures int) []TestFailurePattern {
	var patterns []TestFailurePattern

	// Convert to sortable slice
	for testName, count := range testFailures {
		percentage := float64(count) / float64(totalFailures) * 100
		if totalFailures == 0 {
			percentage = 0
		}

		pattern := TestFailurePattern{
			TestName:   testName,
			Count:      count,
			Percentage: percentage,
			Category:   categorizeTest(testName),
		}
		patterns = append(patterns, pattern)
	}

	// Sort by count (descending)
	sort.Slice(patterns, func(i, j int) bool {
		return patterns[i].Count > patterns[j].Count
	})

	// Return top 5 patterns
	if len(patterns) > 5 {
		patterns = patterns[:5]
	}

	return patterns
}

// categorizeTest attempts to categorize a test based on its name
func categorizeTest(testName string) string {
	testLower := strings.ToLower(testName)

	// Check for infrastructure failures first
	if strings.Contains(testLower, "infrastructure failure") {
		if strings.Contains(testLower, "aborted") {
			return "infra-timeout"
		} else if strings.Contains(testLower, "error") {
			return "infra-error"
		}
		return "infrastructure"
	}

	// Check for common categories based on test name patterns
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

// calculateTimeRange finds the earliest and latest run timestamps
func calculateTimeRange(runs []JobRun) (string, string) {
	if len(runs) == 0 {
		return "", ""
	}

	var firstTime, lastTime time.Time
	var firstTimeStr, lastTimeStr string
	
	for _, run := range runs {
		if run.Timestamp == "" {
			continue
		}
		
		t, err := time.Parse(time.RFC3339, run.Timestamp)
		if err != nil {
			continue
		}

		if firstTime.IsZero() || t.Before(firstTime) {
			firstTime = t
			firstTimeStr = run.Timestamp
		}
		
		if lastTime.IsZero() || t.After(lastTime) {
			lastTime = t
			lastTimeStr = run.Timestamp
		}
	}

	return firstTimeStr, lastTimeStr
}

// MergeSummary contains aggregate statistics for merge command results
type MergeSummary struct {
	TotalTests       int                     `json:"total_tests"`
	TotalFailures    int                     `json:"total_failures"`
	UniqueTests      int                     `json:"unique_tests"`
	TopFailures      []TestFailurePattern   `json:"top_failures"`
	CategoryBreakdown map[string]int        `json:"category_breakdown"`
	JobBreakdown     map[string]int         `json:"job_breakdown"`
	JobTypeStats     map[string]int         `json:"job_type_stats"` // Breakdown of failures by job type
}

// GenerateMergeSummary creates a summary from ProcessorResult
func GenerateMergeSummary(result *ProcessorResult) *MergeSummary {
	summary := &MergeSummary{
		UniqueTests:       len(result.FailedTests),
		CategoryBreakdown: make(map[string]int),
		JobBreakdown:      make(map[string]int),
		JobTypeStats:      make(map[string]int),
	}

	// Count total failures and analyze patterns
	testFailureCounts := make(map[string]int)

	for testName, testcases := range result.FailedTests {
		summary.TotalFailures += len(testcases)
		testFailureCounts[testName] = len(testcases)

		// Categorize failures
		category := categorizeTest(testName)
		summary.CategoryBreakdown[category] += len(testcases)

		// Extract job information and job type from testcases
		for _, testcase := range testcases {
			jobName := extractJobNameFromURL(testcase.URL)
			if jobName != "" {
				summary.JobBreakdown[jobName]++
			}

			// Count job types
			if testcase.JobType != "" {
				summary.JobTypeStats[testcase.JobType]++
			}
		}
	}

	summary.TotalTests = summary.TotalFailures

	// Generate top failure patterns
	summary.TopFailures = analyzeFailurePatterns(testFailureCounts, summary.TotalFailures)

	return summary
}

// extractJobNameFromURL extracts job name from a Prow URL
func extractJobNameFromURL(url string) string {
	// Extract job name from URL like: https://prow.ci.kubevirt.io//view/gs/kubevirt-prow/pr-logs/pull/kubevirt_kubevirt/15434/pull-kubevirt-e2e-arm64/1955736656627634176
	parts := strings.Split(url, "/")
	for i, part := range parts {
		if part == "pr-logs" && i+3 < len(parts) {
			return parts[i+3] // Job name should be 3 positions after "pr-logs"
		}
	}
	return ""
}
