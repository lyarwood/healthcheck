package healthcheck

import (
	"cmp"
	"fmt"
	"maps"
	"slices"
	"strings"
	"time"
)

func FormatLaneRunOutput(laneRunFailures map[string][]Testcase, displayFailures bool) {
	laneRunKeys := slices.Sorted(maps.Keys(laneRunFailures))
	slices.SortFunc(laneRunKeys, func(a, b string) int {
		return cmp.Compare(len(laneRunFailures[a]), len(laneRunFailures[b]))
	})
	slices.Reverse(laneRunKeys)

	for _, laneRunUUID := range laneRunKeys {
		fmt.Printf("Lane Run %s (%d failures)\n\n", laneRunUUID, len(laneRunFailures[laneRunUUID]))
		for _, test := range laneRunFailures[laneRunUUID] {
			if test.IsQuarantined {
				fmt.Printf("\t[QUARANTINED] %s\n", test.Name)
			} else {
				fmt.Printf("\t%s\n", test.Name)
			}
			if displayFailures && test.Failure != nil {
				fmt.Printf("\t%s\n\n", *test.Failure)
			}
			fmt.Printf("\t%s\n\n", test.URL)
		}
		fmt.Println("")
	}
}

func FormatCountedOutput(failedTests map[string][]Testcase, displayFailures bool) {
	failedTestsKeys := slices.Sorted(maps.Keys(failedTests))
	slices.SortFunc(failedTestsKeys, func(a, b string) int {
		return cmp.Compare(len(failedTests[a]), len(failedTests[b]))
	})
	slices.Reverse(failedTestsKeys)

	for _, name := range failedTestsKeys {
		// Check if any instance of this test is quarantined
		isQuarantined := false
		for _, test := range failedTests[name] {
			if test.IsQuarantined {
				isQuarantined = true
				break
			}
		}

		if isQuarantined {
			fmt.Printf("%d\t[QUARANTINED] %s\n\n", len(failedTests[name]), name)
		} else {
			fmt.Printf("%d\t%s\n\n", len(failedTests[name]), name)
		}

		for _, test := range failedTests[name] {
			if displayFailures && test.Failure != nil {
				fmt.Printf("\t%s\n\n", *test.Failure)
			}
			fmt.Printf("\t%s\n\n", test.URL)
		}
		fmt.Println("")
	}
}

// FormatLaneOutput displays lane analysis results in various formats
func FormatLaneOutput(jobName string, summary *LaneSummary, config LaneDisplayConfig) {
	// Handle summary output
	if config.Summary {
		FormatLaneSummary(jobName, summary)
		return
	}

	// Handle URL-only output
	if config.DisplayOnlyURLs {
		for _, run := range summary.Runs {
			if run.Status == "FAILURE" {
				fmt.Println(run.URL)
			}
		}
		return
	}

	// Handle test names-only output
	if config.DisplayOnlyTestNames {
		for _, failure := range summary.AllFailures {
			fmt.Println(failure.Name)
		}
		return
	}

	// Handle count failures output (similar to merge command)
	if config.CountFailures {
		// Group failures by test name like merge command does
		failedTests := make(map[string][]Testcase)
		for _, failure := range summary.AllFailures {
			failedTests[failure.Name] = append(failedTests[failure.Name], failure)
		}
		
		FormatCountedOutput(failedTests, config.DisplayFailures)
		return
	}

	// Default output: simple list of failures
	for _, failure := range summary.AllFailures {
		fmt.Println(failure.Name)
		if config.DisplayFailures && failure.Failure != nil {
			fmt.Printf("%s\n\n", *failure.Failure)
		}
		fmt.Printf("%s\n\n", failure.URL)
	}
}

// FormatLaneSummary displays a concise summary of lane analysis
func FormatLaneSummary(jobName string, summary *LaneSummary) {
	fmt.Printf("Lane Summary: %s\n", jobName)
	fmt.Printf("=" + strings.Repeat("=", len(jobName)+13) + "\n\n")

	// Time range information
	if summary.FirstRunTime != "" && summary.LastRunTime != "" {
		firstTime := formatTimestamp(summary.FirstRunTime)
		lastTime := formatTimestamp(summary.LastRunTime)
		duration := calculateDuration(summary.FirstRunTime, summary.LastRunTime)
		
		fmt.Printf("Time Range:\n")
		fmt.Printf("  First Run:  %s\n", firstTime)
		fmt.Printf("  Last Run:   %s\n", lastTime)
		if duration != "" {
			fmt.Printf("  Duration:   %s\n", duration)
		}
		fmt.Println()
	}

	// Overall statistics
	fmt.Printf("Test Run Statistics:\n")
	fmt.Printf("  Total Runs:     %d\n", summary.TotalRuns)
	fmt.Printf("  Successful:     %d\n", summary.SuccessfulRuns)
	fmt.Printf("  Failed:         %d\n", summary.FailedRuns)
	fmt.Printf("  Unknown:        %d\n", summary.TotalRuns-summary.SuccessfulRuns-summary.FailedRuns)
	fmt.Printf("  Failure Rate:   %.1f%%\n\n", summary.FailureRate)

	// Test failure statistics
	if len(summary.AllFailures) > 0 {
		fmt.Printf("Test Failure Statistics:\n")
		fmt.Printf("  Total Failures: %d\n", len(summary.AllFailures))
		fmt.Printf("  Unique Tests:   %d\n\n", len(summary.TestFailures))

		// Category breakdown
		if len(summary.TopFailures) > 0 {
			fmt.Printf("Failure Categories:\n")
			categories := make(map[string]int)
			for _, pattern := range summary.TopFailures {
				categories[pattern.Category] += pattern.Count
			}
			
			for category, count := range categories {
				percentage := float64(count) / float64(len(summary.AllFailures)) * 100
				fmt.Printf("  %-10s: %d (%.1f%%)\n", category, count, percentage)
			}
			fmt.Println()

			// Top failing tests
			fmt.Printf("Most Frequent Failures:\n")
			for i, pattern := range summary.TopFailures {
				if i >= 3 { // Show only top 3
					break
				}
				fmt.Printf("  %d. [%s] %s (%d failures, %.1f%%)\n", 
					i+1, pattern.Category, truncateTestName(pattern.TestName, 60), 
					pattern.Count, pattern.Percentage)
			}
			fmt.Println()
		}

		// Pattern insights
		fmt.Printf("Pattern Analysis:\n")
		if summary.FailureRate > 80 {
			fmt.Printf("  🔴 High failure rate - investigate systemic issues\n")
		} else if summary.FailureRate > 50 {
			fmt.Printf("  🟡 Moderate failure rate - monitor trends\n")
		} else if summary.FailureRate > 20 {
			fmt.Printf("  🟠 Low failure rate - normal fluctuation\n")
		} else {
			fmt.Printf("  🟢 Very low failure rate - stable\n")
		}

		if len(summary.TopFailures) > 0 {
			topFailure := summary.TopFailures[0]
			if topFailure.Percentage > 50 {
				fmt.Printf("  🎯 Single dominant failure pattern (%s)\n", topFailure.Category)
			} else if len(summary.TopFailures) >= 2 && summary.TopFailures[1].Percentage > 25 {
				fmt.Printf("  📊 Multiple significant failure patterns\n")
			} else {
				fmt.Printf("  🔀 Diverse failure patterns - no clear dominant issue\n")
			}
		}
	} else {
		fmt.Printf("🎉 No test failures detected!\n")
	}
}

// truncateTestName truncates long test names for display
func truncateTestName(name string, maxLen int) string {
	if len(name) <= maxLen {
		return name
	}
	return name[:maxLen-3] + "..."
}

// formatTimestamp converts RFC3339 timestamp to a readable format
func formatTimestamp(timestamp string) string {
	if timestamp == "" {
		return "Unknown"
	}
	
	t, err := time.Parse(time.RFC3339, timestamp)
	if err != nil {
		return timestamp // Return original if parsing fails
	}
	
	// Format as: "2025-08-20 16:22:12 UTC"
	return t.UTC().Format("2006-01-02 15:04:05 UTC")
}

// calculateDuration calculates and formats the duration between first and last run
func calculateDuration(firstTime, lastTime string) string {
	if firstTime == "" || lastTime == "" {
		return ""
	}
	
	first, err := time.Parse(time.RFC3339, firstTime)
	if err != nil {
		return ""
	}
	
	last, err := time.Parse(time.RFC3339, lastTime)
	if err != nil {
		return ""
	}
	
	duration := last.Sub(first)
	
	// Format duration in a human-readable way
	if duration == 0 {
		return "Single run"
	}
	
	if duration < time.Hour {
		return fmt.Sprintf("%.0f minutes", duration.Minutes())
	} else if duration < 24*time.Hour {
		return fmt.Sprintf("%.1f hours", duration.Hours())
	} else {
		days := duration.Hours() / 24
		return fmt.Sprintf("%.1f days", days)
	}
}

// FormatMergeSummary displays a concise summary of merge command results
func FormatMergeSummary(result *ProcessorResult) {
	summary := GenerateMergeSummary(result)
	
	fmt.Printf("Merge Summary\n")
	fmt.Printf("=============\n\n")

	// Overall statistics
	fmt.Printf("Test Failure Statistics:\n")
	fmt.Printf("  Total Failures:    %d\n", summary.TotalFailures)
	fmt.Printf("  Unique Tests:       %d\n", summary.UniqueTests)
	if summary.UniqueTests > 0 {
		avgFailures := float64(summary.TotalFailures) / float64(summary.UniqueTests)
		fmt.Printf("  Avg per Test:       %.1f\n", avgFailures)
	}
	fmt.Println()

	// Category breakdown
	if len(summary.CategoryBreakdown) > 0 {
		fmt.Printf("Failure Categories:\n")
		for category, count := range summary.CategoryBreakdown {
			percentage := float64(count) / float64(summary.TotalFailures) * 100
			fmt.Printf("  %-10s: %d (%.1f%%)\n", category, count, percentage)
		}
		fmt.Println()
	}

	// Top failing tests
	if len(summary.TopFailures) > 0 {
		fmt.Printf("Most Frequent Failures:\n")
		for i, pattern := range summary.TopFailures {
			if i >= 5 { // Show only top 5
				break
			}
			fmt.Printf("  %d. [%s] %s (%d failures, %.1f%%)\n", 
				i+1, pattern.Category, truncateTestName(pattern.TestName, 60), 
				pattern.Count, pattern.Percentage)
		}
		fmt.Println()
	}

	// Top affected jobs
	if len(summary.JobBreakdown) > 0 {
		fmt.Printf("Most Affected Jobs:\n")
		
		// Convert to sortable slice
		type jobCount struct {
			name  string
			count int
		}
		var jobs []jobCount
		for jobName, count := range summary.JobBreakdown {
			jobs = append(jobs, jobCount{name: jobName, count: count})
		}
		
		// Sort by count (descending)
		for i := 0; i < len(jobs)-1; i++ {
			for j := i + 1; j < len(jobs); j++ {
				if jobs[j].count > jobs[i].count {
					jobs[i], jobs[j] = jobs[j], jobs[i]
				}
			}
		}
		
		// Show top 5 jobs
		for i, job := range jobs {
			if i >= 5 {
				break
			}
			percentage := float64(job.count) / float64(summary.TotalFailures) * 100
			fmt.Printf("  %d. %s (%d failures, %.1f%%)\n", 
				i+1, truncateTestName(job.name, 50), job.count, percentage)
		}
		fmt.Println()
	}

	// Pattern insights
	fmt.Printf("Pattern Analysis:\n")
	if summary.TotalFailures == 0 {
		fmt.Printf("  🎉 No test failures detected!\n")
	} else if len(summary.TopFailures) > 0 {
		topFailure := summary.TopFailures[0]
		if topFailure.Percentage > 50 {
			fmt.Printf("  🎯 Single dominant failure pattern (%s)\n", topFailure.Category)
		} else if len(summary.TopFailures) >= 2 && summary.TopFailures[1].Percentage > 25 {
			fmt.Printf("  📊 Multiple significant failure patterns\n")
		} else {
			fmt.Printf("  🔀 Diverse failure patterns - no clear dominant issue\n")
		}
		
		// Category analysis
		maxCategory := ""
		maxCount := 0
		for category, count := range summary.CategoryBreakdown {
			if count > maxCount {
				maxCount = count
				maxCategory = category
			}
		}
		
		if maxCount > 0 {
			percentage := float64(maxCount) / float64(summary.TotalFailures) * 100
			if percentage > 60 {
				fmt.Printf("  🔍 Focus area: %s category (%.1f%% of failures)\n", maxCategory, percentage)
			}
		}
	}
}
