package cmd

import (
	"fmt"
	"os"
	"regexp"

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
)


var rootCmd = &cobra.Command{
	Use:   "",
	Short: "Parse KubeVirt CI health data and report failed tests",
	RunE: func(cmd *cobra.Command, args []string) error {

		// Resolve job regex aliases
		if _, ok := jobRegexAliases[jobRegex]; ok {
			jobRegex = jobRegexAliases[jobRegex]
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
		results, err := fetchResults(healthURL)
		if err != nil {
			return err
		}

		// Configure processor
		config := ProcessorConfig{
			JobRegex:             jobRegexCompiled,
			TestRegex:            testRegexCompiled,
			DisplayOnlyURLs:      displayOnlyURLs,
			DisplayOnlyTestNames: displayOnlyTestNames,
			DisplayFailures:      displayFailures,
			CountFailures:        countFailures,
			GroupByLaneRun:       groupByLaneRun,
		}

		// Process failures
		result, err := processFailures(results, config)
		if err != nil {
			return err
		}

		// Output results
		if groupByLaneRun {
			formatLaneRunOutput(result.LaneRunFailures, displayFailures)
			return nil
		}

		if countFailures {
			formatCountedOutput(result.FailedTests, displayFailures)
		}
		return nil
	},
}

func init() {
	rootCmd.Flags().StringVarP(&jobRegex, "job", "j", "main", "Job name regex")
	rootCmd.Flags().StringVarP(&testRegex, "test", "t", "", "Test name regex")
	rootCmd.Flags().BoolVarP(&countFailures, "count", "c", false, "Count specific test failures")
	rootCmd.Flags().BoolVarP(&displayOnlyURLs, "url", "u", false, "Display only failed job URLs")
	rootCmd.Flags().BoolVarP(&displayOnlyTestNames, "name", "n", false, "Display only failed test names")
	rootCmd.Flags().BoolVarP(&displayFailures, "failures", "f", false, "print any captured failure context")
	rootCmd.Flags().BoolVarP(&groupByLaneRun, "lane-run", "l", false, "Group failures by lane run UUID")
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

