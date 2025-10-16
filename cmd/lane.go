package cmd

import (
	"encoding/json"
	"fmt"

	"healthcheck/pkg/healthcheck"

	"github.com/spf13/cobra"
)

var (
	laneLimit            int
	laneCountFailures    bool
	laneDisplayOnlyURLs  bool
	laneDisplayOnlyTestNames bool
	laneDisplayFailures  bool
	laneSincePeriod      string
	laneSummary          bool
	laneOutputFormat     string
	laneJobType          string
)

var laneCmd = &cobra.Command{
	Use:   "lane [job-name]",
	Short: "Analyze recent job runs for a specific lane",
	Args:  cobra.ExactArgs(1),
	RunE: func(_ *cobra.Command, args []string) error {
		jobName := args[0]

		// Parse time period if provided
		timePeriod, err := healthcheck.ParseTimePeriod(laneSincePeriod)
		if err != nil {
			return fmt.Errorf("invalid time period: %w", err)
		}

		// Fetch job history with smart pagination based on time period
		var runs []healthcheck.JobRun
		if timePeriod > 0 {
			// Use time-based pagination with reasonable max limit to prevent runaway
			maxLimit := 1000 // Safety limit to prevent excessive API calls
			runs, err = healthcheck.FetchJobHistoryWithTimePeriod(jobName, timePeriod, maxLimit)
		} else {
			// Use regular limit-based fetching
			runs, err = healthcheck.FetchJobHistory(jobName, laneLimit)
		}
		if err != nil {
			return fmt.Errorf("failed to fetch job history for %s: %w", jobName, err)
		}

		// Analyze each run (this populates JobType field)
		summary, err := healthcheck.AnalyzeLaneRuns(runs)
		if err != nil {
			return fmt.Errorf("failed to analyze lane runs: %w", err)
		}

		// Filter by job type if specified (after analysis to ensure JobType is populated)
		if laneJobType != "" {
			summary = healthcheck.FilterLaneSummaryByJobType(summary, laneJobType)
		}

		// Configure lane display options
		config := healthcheck.LaneDisplayConfig{
			CountFailures:        laneCountFailures,
			DisplayOnlyURLs:      laneDisplayOnlyURLs,
			DisplayOnlyTestNames: laneDisplayOnlyTestNames,
			DisplayFailures:      laneDisplayFailures,
			Summary:              laneSummary,
		}

		// Display results
		if laneOutputFormat == "json" {
			return outputLaneJSON(jobName, summary, config)
		} else {
			healthcheck.FormatLaneOutput(jobName, summary, config)
			return nil
		}
	},
}

func init() {
	laneCmd.Flags().IntVarP(&laneLimit, "limit", "l", 10, "Number of recent runs to analyze (ignored when --since is used)")
	laneCmd.Flags().BoolVarP(&laneCountFailures, "count", "c", false, "Count specific test failures")
	laneCmd.Flags().BoolVarP(&laneDisplayOnlyURLs, "url", "u", false, "Display only failed job URLs")
	laneCmd.Flags().BoolVarP(&laneDisplayOnlyTestNames, "name", "n", false, "Display only failed test names")
	laneCmd.Flags().BoolVarP(&laneDisplayFailures, "failures", "f", false, "Print any captured failure context")
	laneCmd.Flags().StringVarP(&laneSincePeriod, "since", "s", "", "Fetch all results within time period (e.g., 24h, 2d, 1w) with automatic pagination")
	laneCmd.Flags().BoolVar(&laneSummary, "summary", false, "Display a concise summary of test runs and failure patterns")
	laneCmd.Flags().StringVarP(&laneOutputFormat, "output", "o", "text", "Output format: text or json")
	laneCmd.Flags().StringVarP(&laneJobType, "type", "t", "", "Filter jobs by type (e.g., batch, presubmit, periodic, postsubmit)")

	rootCmd.AddCommand(laneCmd)
}

// outputLaneJSON outputs lane data in JSON format
func outputLaneJSON(jobName string, summary *healthcheck.LaneSummary, config healthcheck.LaneDisplayConfig) error {
	var output interface{}

	if config.DisplayOnlyURLs {
		// Extract URLs from failed runs
		var urls []string
		for _, run := range summary.Runs {
			if run.Status == "FAILURE" {
				urls = append(urls, run.URL)
			}
		}
		output = map[string]interface{}{
			"job_name": jobName,
			"urls":     urls,
		}
	} else if config.DisplayOnlyTestNames {
		// Extract unique test names
		testNames := make([]string, 0, len(summary.TestFailures))
		for testName := range summary.TestFailures {
			testNames = append(testNames, testName)
		}
		output = map[string]interface{}{
			"job_name":   jobName,
			"test_names": testNames,
		}
	} else if config.CountFailures {
		// Count failures by test name
		output = map[string]interface{}{
			"job_name":      jobName,
			"test_failures": summary.TestFailures,
		}
	} else if config.Summary {
		// Full summary with statistics
		output = map[string]interface{}{
			"job_name":        jobName,
			"total_runs":      summary.TotalRuns,
			"successful_runs": summary.SuccessfulRuns,
			"failed_runs":     summary.FailedRuns,
			"failure_rate":    summary.FailureRate,
			"test_failures":   summary.TestFailures,
			"all_failures":    summary.AllFailures,
			"top_failures":    summary.TopFailures,
			"first_run_time":  summary.FirstRunTime,
			"last_run_time":   summary.LastRunTime,
			"runs":            summary.Runs,
		}
	} else {
		// Default: all failures with details
		output = map[string]interface{}{
			"job_name":     jobName,
			"all_failures": summary.AllFailures,
		}
	}

	jsonBytes, err := json.MarshalIndent(output, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal JSON output: %w", err)
	}

	fmt.Println(string(jsonBytes))
	return nil
}