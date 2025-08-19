package healthcheck

import (
	"cmp"
	"fmt"
	"maps"
	"slices"
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
			if displayFailures {
				fmt.Printf("\t%s\n\n", *test.Failure)
			}
			fmt.Printf("\t%s\n\n", test.URL)
		}
		fmt.Println("")
	}
}
