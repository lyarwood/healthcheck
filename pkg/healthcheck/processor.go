package healthcheck

import (
	"fmt"
	"regexp"
	"strings"
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

func processJobFailure(job Job, failureURL string, config ProcessorConfig,
	result *ProcessorResult, quarantinedTests map[string]bool) error {
	testsuite, err := fetchTestSuite(failureURL)
	if err != nil {
		return err
	}

	if testsuite == nil {
		return handleMissingTestsuite(job, failureURL, config, result, quarantinedTests)
	}

	return processTestcases(testsuite, failureURL, config, result, quarantinedTests)
}

func handleMissingTestsuite(job Job, failureURL string, config ProcessorConfig,
	result *ProcessorResult, _ map[string]bool) error {
	if config.DisplayOnlyURLs {
		fmt.Println(failureURL)
		return nil
	}
	if config.DisplayOnlyTestNames {
		fmt.Printf("%s (no junit file to parse)\n", job.JobName)
		return nil
	}
	if config.GroupByLaneRun {
		laneRunUUID := ExtractLaneRunUUID(failureURL)
		if laneRunUUID != "" {
			placeholder := Testcase{Name: fmt.Sprintf("%s (no junit file to parse)", job.JobName), URL: failureURL}
			result.LaneRunFailures[laneRunUUID] = append(result.LaneRunFailures[laneRunUUID], placeholder)
		}
		return nil
	}
	fmt.Printf("%s (no junit file to parse)\n", job.JobName)
	fmt.Printf("%s\n\n", failureURL)
	return nil
}

func processTestcases(testsuite *Testsuite, failureURL string, config ProcessorConfig,
	result *ProcessorResult, quarantinedTests map[string]bool) error {
	for _, testcase := range testsuite.Testcase {
		if testcase.Failure == nil || !config.TestRegex.MatchString(testcase.Name) {
			continue
		}

		if config.DisplayOnlyURLs {
			fmt.Println(failureURL)
			continue
		}
		if config.DisplayOnlyTestNames {
			fmt.Println(testcase.Name)
			continue
		}

		testcase.URL = failureURL

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
	fmt.Println(testcase.Name)
	if config.DisplayFailures {
		fmt.Printf("%s\n\n", testcase.Failure)
	}
	fmt.Printf("%s\n\n", testcase.URL)
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
