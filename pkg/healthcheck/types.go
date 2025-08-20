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
	Timestamp string
	Failures  []Testcase
}

type LaneSummary struct {
	TotalRuns     int
	SuccessfulRuns int
	FailedRuns    int
	TestFailures  map[string]int
	Runs          []JobRun
	AllFailures   []Testcase  // All test failures across all runs
}

type LaneDisplayConfig struct {
	CountFailures        bool
	DisplayOnlyURLs      bool
	DisplayOnlyTestNames bool
	DisplayFailures      bool
}
