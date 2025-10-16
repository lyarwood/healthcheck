package healthcheck

import "encoding/xml"

type Results struct {
	Data struct {
		SIGRetests struct {
			FailedJobLeaderBoard []Job `json:"FailedJobLeaderBoard"`
		} `json:"SIGRetests"`
	} `json:"Data"`
}

type Job struct {
	JobName      string   `json:"JobName"`
	FailureCount int      `json:"FailureCount"`
	SuccessCount int      `json:"SuccessCount"`
	FailureURLs  []string `json:"FailureURLs"`
}

type Testsuite struct {
	XMLName  xml.Name   `xml:"testsuite"`
	Failures string     `xml:"failures,attr"`
	Name     string     `xml:"name,attr"`
	Tests    string     `xml:"tests,attr"`
	Time     string     `xml:"time,attr"`
	Testcase []Testcase `xml:"testcase"`
}

type Testcase struct {
	XMLName       xml.Name `xml:"testcase"`
	Classname     string   `xml:"classname,attr"`
	Name          string   `xml:"name,attr"`
	Time          string   `xml:"time,attr"`
	Failure       *Failure `xml:"failure,omitempty"`
	URL           string   `xml:"url,omitempty"`
	JobType       string   `xml:"-"` // Prow job type: presubmit, batch, postsubmit, periodic
	IsQuarantined bool     `xml:"-"`
}

type Failure struct {
	XMLName xml.Name `xml:"failure"`
	Message string   `xml:"message,attr"`
	Type    string   `xml:"type,attr"`
	Value   string   `xml:",chardata"`
}

type JobRun struct {
	ID        string
	URL       string
	Status    string
	JobType   string // Prow job type: presubmit, batch, postsubmit, periodic
	Timestamp string
	Failures  []Testcase
}

type LaneSummary struct {
	TotalRuns     int
	SuccessfulRuns int
	FailedRuns    int
	AbortedRuns   int         // Jobs aborted due to infrastructure issues
	ErrorRuns     int         // Jobs with system errors
	UnknownRuns   int         // Jobs with unknown status
	JobTypeStats  map[string]int // Breakdown of runs by job type (presubmit, batch, etc.)
	TestFailures  map[string]int
	Runs          []JobRun
	AllFailures   []Testcase  // All test failures across all runs
	FailureRate   float64     // Percentage of runs that failed
	InfrastructureFailureRate float64 // Percentage of failures due to infrastructure
	TopFailures   []TestFailurePattern // Most common failure patterns
	FirstRunTime  string      // Timestamp of earliest run
	LastRunTime   string      // Timestamp of latest run
}

type TestFailurePattern struct {
	TestName    string
	Count       int
	Percentage  float64
	Category    string // e.g., "compute", "network", "storage"
}

type LaneDisplayConfig struct {
	CountFailures        bool
	DisplayOnlyURLs      bool
	DisplayOnlyTestNames bool
	DisplayFailures      bool
	Summary              bool
}
