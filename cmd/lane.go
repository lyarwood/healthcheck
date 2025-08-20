package cmd

import (
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

		// Analyze each run
		summary, err := healthcheck.AnalyzeLaneRuns(runs)
		if err != nil {
			return fmt.Errorf("failed to analyze lane runs: %w", err)
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
		healthcheck.FormatLaneOutput(jobName, summary, config)
		return nil
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

	rootCmd.AddCommand(laneCmd)
}