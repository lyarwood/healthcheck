package cmd

import (
	"encoding/json"
	"fmt"
	"regexp"

	"healthcheck/pkg/healthcheck"

	"github.com/spf13/cobra"
)

var (
	testRegex            string
	countFailures        bool
	displayOnlyURLs      bool
	displayOnlyTestNames bool
	displayFailures      bool
	groupByLaneRun       bool
	checkQuarantine      bool
	sincePeriod          string
	outputFormat         string
	summary              bool
)

var mergeCmd = &cobra.Command{
	Use:   "merge [job-name-or-alias]",
	Short: "Parse KubeVirt CI health data and report failed tests",
	Args:  cobra.ExactArgs(1),
	RunE: func(_ *cobra.Command, args []string) error {
		jobName := args[0]

		// Parse time period if provided
		timePeriod, err := healthcheck.ParseTimePeriod(sincePeriod)
		if err != nil {
			return fmt.Errorf("invalid time period: %w", err)
		}

		// Resolve job regex aliases
		if _, ok := healthcheck.JobRegexAliases[jobName]; ok {
			jobName = healthcheck.JobRegexAliases[jobName]
		}

		// Compile regexes
		jobRegexCompiled, err := regexp.Compile(jobName)
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
			TimePeriod:           timePeriod,
			SuppressOutput:       outputFormat == "json", // Suppress output for JSON formatting
			Summary:              summary,
		}

		// Process failures
		result, err := healthcheck.ProcessFailures(results, config)
		if err != nil {
			return err
		}

		// Output results
		if outputFormat == "json" {
			return outputMergeJSON(result, config)
		} else {
			if summary {
				healthcheck.FormatMergeSummary(result)
			} else if groupByLaneRun {
				healthcheck.FormatLaneRunOutput(result.LaneRunFailures, displayFailures)
			} else if countFailures {
				healthcheck.FormatCountedOutput(result.FailedTests, displayFailures)
			} else {
				// Default output: display all failed tests
				healthcheck.FormatCountedOutput(result.FailedTests, displayFailures)
			}
			return nil
		}
	},
}

func init() {
	mergeCmd.Flags().StringVarP(&testRegex, "test", "t", "", "Test name regex")
	mergeCmd.Flags().BoolVarP(&countFailures, "count", "c", false, "Count specific test failures")
	mergeCmd.Flags().BoolVarP(&displayOnlyURLs, "url", "u", false, "Display only failed job URLs")
	mergeCmd.Flags().BoolVarP(&displayOnlyTestNames, "name", "n", false, "Display only failed test names")
	mergeCmd.Flags().BoolVarP(&displayFailures, "failures", "f", false, "print any captured failure context")
	mergeCmd.Flags().BoolVarP(&groupByLaneRun, "lane-run", "l", false, "Group failures by lane run UUID")
	mergeCmd.Flags().BoolVarP(&checkQuarantine, "quarantine", "q", false, "Check and highlight quarantined tests")
	mergeCmd.Flags().StringVarP(&sincePeriod, "since", "s", "", "Limit results to given time period (e.g., 24h, 2d, 1w)")
	mergeCmd.Flags().StringVarP(&outputFormat, "output", "o", "text", "Output format: text or json")
	mergeCmd.Flags().BoolVar(&summary, "summary", false, "Display a concise summary of failures and patterns")

	rootCmd.AddCommand(mergeCmd)
}

// outputMergeJSON outputs merge data in JSON format
func outputMergeJSON(result *healthcheck.ProcessorResult, config healthcheck.ProcessorConfig) error {
	var output interface{}

	if config.DisplayOnlyURLs {
		// Extract URLs from all test cases
		var urls []string
		for _, testcases := range result.FailedTests {
			for _, testcase := range testcases {
				if testcase.URL != "" {
					urls = append(urls, testcase.URL)
				}
			}
		}
		output = map[string]interface{}{
			"urls": urls,
		}
	} else if config.DisplayOnlyTestNames {
		// Extract unique test names
		testNames := make([]string, 0, len(result.FailedTests))
		for testName := range result.FailedTests {
			testNames = append(testNames, testName)
		}
		output = map[string]interface{}{
			"test_names": testNames,
		}
	} else if config.GroupByLaneRun {
		// Group by lane run UUID
		output = map[string]interface{}{
			"lane_run_failures": result.LaneRunFailures,
		}
	} else if config.CountFailures {
		// Count failures by test name
		testCounts := make(map[string]int)
		for testName, testcases := range result.FailedTests {
			testCounts[testName] = len(testcases)
		}
		output = map[string]interface{}{
			"test_failure_counts": testCounts,
			"failed_tests":        result.FailedTests,
		}
	} else if config.Summary {
		// Generate summary statistics
		summary := healthcheck.GenerateMergeSummary(result)
		output = map[string]interface{}{
			"summary":       summary,
			"failed_tests":  result.FailedTests,
		}
	} else {
		// Default: all failed tests with details
		output = map[string]interface{}{
			"failed_tests":      result.FailedTests,
			"lane_run_failures": result.LaneRunFailures,
		}
	}

	jsonBytes, err := json.MarshalIndent(output, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal JSON output: %w", err)
	}

	fmt.Println(string(jsonBytes))
	return nil
}