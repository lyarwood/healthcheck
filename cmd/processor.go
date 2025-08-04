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
	failedTests := make(map[string][]Testcase)
	laneRunFailures := make(map[string][]Testcase)

	for _, job := range results.Data.SIGRetests.FailedJobLeaderBoard {
		if !config.JobRegex.MatchString(job.JobName) {
			continue
		}

		for _, failureURL := range job.FailureURLs {
			testsuite, err := fetchTestSuite(failureURL)
			if err != nil {
				return nil, err
			}

			// Handle missing testsuite
			if testsuite == nil {
				if config.DisplayOnlyURLs {
					fmt.Println(failureURL)
					continue
				}
				if config.DisplayOnlyTestNames {
					fmt.Printf("%s (no junit file to parse)\n", job.JobName)
					continue
				}
				if config.GroupByLaneRun {
					laneRunUUID := extractLaneRunUUID(failureURL)
					if laneRunUUID != "" {
						placeholder := Testcase{Name: fmt.Sprintf("%s (no junit file to parse)", job.JobName), URL: failureURL}
						laneRunFailures[laneRunUUID] = append(laneRunFailures[laneRunUUID], placeholder)
					}
					continue
				}
				fmt.Printf("%s (no junit file to parse)\n", job.JobName)
				fmt.Printf("%s\n\n", failureURL)
				continue
			}

			// Process test cases
			for _, testcase := range testsuite.Testcase {
				if testcase.Failure == nil {
					continue
				}
				if !config.TestRegex.MatchString(testcase.Name) {
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

				if config.GroupByLaneRun {
					laneRunUUID := extractLaneRunUUID(failureURL)
					if laneRunUUID != "" {
						laneRunFailures[laneRunUUID] = append(laneRunFailures[laneRunUUID], testcase)
					}
					continue
				}
				if config.CountFailures {
					failedTests[testcase.Name] = append(failedTests[testcase.Name], testcase)
					continue
				}

				// Default output for non-count, non-grouped mode
				fmt.Println(testcase.Name)
				if config.DisplayFailures {
					fmt.Printf("%s\n\n", testcase.Failure)
				}
				fmt.Printf("%s\n\n", failureURL)
			}
		}
	}

	return &ProcessorResult{
		FailedTests:     failedTests,
		LaneRunFailures: laneRunFailures,
	}, nil
}