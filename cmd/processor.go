package cmd

import (
	"fmt"
	"regexp"
	"strings"
)

var jobRegexAliases = map[string]string{
	"main":    "sig-[a-zA-Z0-9_-]+$",
	"1.6":     "release-1.6$",
	"1.5":     "release-1.5$",
	"1.4":     "release-1.4$",
	"compute": "sig-compute$|sig-compute-serial$|sig-compute-migrations$|sig-operator$",
	"network": "sig-network$",
	"storage": "sig-storage$",
}

type ProcessorConfig struct {
	JobRegex             *regexp.Regexp
	TestRegex            *regexp.Regexp
	DisplayOnlyURLs      bool
	DisplayOnlyTestNames bool
	DisplayFailures      bool
	CountFailures        bool
	GroupByLaneRun       bool
}

type ProcessorResult struct {
	FailedTests     map[string][]Testcase
	LaneRunFailures map[string][]Testcase
}

func extractLaneRunUUID(failureURL string) string {
	parts := strings.Split(failureURL, "/")
	if len(parts) > 0 {
		return parts[len(parts)-1]
	}
	return ""
}

func processFailures(results *Results, config ProcessorConfig) (*ProcessorResult, error) {
	result := &ProcessorResult{
		FailedTests:     make(map[string][]Testcase),
		LaneRunFailures: make(map[string][]Testcase),
	}

	for _, job := range results.Data.SIGRetests.FailedJobLeaderBoard {
		if !config.JobRegex.MatchString(job.JobName) {
			continue
		}

		for _, failureURL := range job.FailureURLs {
			if err := processJobFailure(job, failureURL, config, result); err != nil {
				return nil, err
			}
		}
	}

	return result, nil
}

func processJobFailure(job Job, failureURL string, config ProcessorConfig, result *ProcessorResult) error {
	testsuite, err := fetchTestSuite(failureURL)
	if err != nil {
		return err
	}

	if testsuite == nil {
		return handleMissingTestsuite(job, failureURL, config, result)
	}

	return processTestcases(testsuite, failureURL, config, result)
}

func handleMissingTestsuite(job Job, failureURL string, config ProcessorConfig, result *ProcessorResult) error {
	if config.DisplayOnlyURLs {
		fmt.Println(failureURL)
		return nil
	}
	if config.DisplayOnlyTestNames {
		fmt.Printf("%s (no junit file to parse)\n", job.JobName)
		return nil
	}
	if config.GroupByLaneRun {
		laneRunUUID := extractLaneRunUUID(failureURL)
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

func processTestcases(testsuite *Testsuite, failureURL string, config ProcessorConfig, result *ProcessorResult) error {
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
		processTestcase(testcase, config, result)
	}
	return nil
}

func processTestcase(testcase Testcase, config ProcessorConfig, result *ProcessorResult) {
	if config.GroupByLaneRun {
		laneRunUUID := extractLaneRunUUID(testcase.URL)
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
