package cmd

import (
	"fmt"
	"regexp"

	"healthcheck/pkg/healthcheck"

	"github.com/spf13/cobra"
)

var (
	jobRegex             string
	testRegex            string
	countFailures        bool
	displayOnlyURLs      bool
	displayOnlyTestNames bool
	displayFailures      bool
	groupByLaneRun       bool
	checkQuarantine      bool
)

var mergeCmd = &cobra.Command{
	Use:   "merge",
	Short: "Parse KubeVirt CI health data and report failed tests",
	RunE: func(_ *cobra.Command, _ []string) error {

		// Resolve job regex aliases
		if _, ok := healthcheck.JobRegexAliases[jobRegex]; ok {
			jobRegex = healthcheck.JobRegexAliases[jobRegex]
		}

		// Compile regexes
		jobRegexCompiled, err := regexp.Compile(jobRegex)
		if err != nil {
			return fmt.Errorf("invalid job name regex provided: %w", err)
		}

		testRegexCompiled, err := regexp.Compile(testRegex)
		if err != nil {
			return fmt.Errorf("invalid test name regex provided: %w", err)
		}

		// Fetch CI health data
		results, err := healthcheck.FetchResults(healthcheck.HealthURL)
		if err != nil {
			return err
		}

		// Configure processor
		config := healthcheck.ProcessorConfig{
			JobRegex:             jobRegexCompiled,
			TestRegex:            testRegexCompiled,
			DisplayOnlyURLs:      displayOnlyURLs,
			DisplayOnlyTestNames: displayOnlyTestNames,
			DisplayFailures:      displayFailures,
			CountFailures:        countFailures,
			GroupByLaneRun:       groupByLaneRun,
			CheckQuarantine:      checkQuarantine,
		}

		// Process failures
		result, err := healthcheck.ProcessFailures(results, config)
		if err != nil {
			return err
		}

		// Output results
		if groupByLaneRun {
			healthcheck.FormatLaneRunOutput(result.LaneRunFailures, displayFailures)
			return nil
		}

		if countFailures {
			healthcheck.FormatCountedOutput(result.FailedTests, displayFailures)
		}
		return nil
	},
}

func init() {
	mergeCmd.Flags().StringVarP(&jobRegex, "job", "j", "main", "Job name regex")
	mergeCmd.Flags().StringVarP(&testRegex, "test", "t", "", "Test name regex")
	mergeCmd.Flags().BoolVarP(&countFailures, "count", "c", false, "Count specific test failures")
	mergeCmd.Flags().BoolVarP(&displayOnlyURLs, "url", "u", false, "Display only failed job URLs")
	mergeCmd.Flags().BoolVarP(&displayOnlyTestNames, "name", "n", false, "Display only failed test names")
	mergeCmd.Flags().BoolVarP(&displayFailures, "failures", "f", false, "print any captured failure context")
	mergeCmd.Flags().BoolVarP(&groupByLaneRun, "lane-run", "l", false, "Group failures by lane run UUID")
	mergeCmd.Flags().BoolVarP(&checkQuarantine, "quarantine", "q", false, "Check and highlight quarantined tests")

	rootCmd.AddCommand(mergeCmd)
}