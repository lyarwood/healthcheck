package cmd

import (
	"cmp"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"maps"
	"net"
	"net/http"
	"os"
	"regexp"
	"slices"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

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

type Testsuites struct {
	XMLName   xml.Name    `xml:"testsuites"`
	Testsuite []Testsuite `xml:"testsuite"`
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
	XMLName   xml.Name `xml:"testcase"`
	Classname string   `xml:"classname,attr"`
	Name      string   `xml:"name,attr"`
	Time      string   `xml:"time,attr"`
	Failure   *Failure `xml:"failure,omitempty"`
	URL       string   `xml:"url,omitempty"`
}

type Failure struct {
	XMLName xml.Name `xml:"failure"`
	Message string   `xml:"message,attr"`
	Type    string   `xml:"type,attr"`
	Value   string   `xml:",chardata"`
}

var (
	jobRegex             string
	testRegex            string
	countFailures        bool
	displayOnlyURLs      bool
	displayOnlyTestNames bool
	displayFailures      bool
)

const healthURL = "https://kubevirt.io/ci-health/output/kubevirt/kubevirt/results.json"

var rootCmd = &cobra.Command{
	Use:   "",
	Short: "Parse KubeVirt CI health data and report failed tests",
	RunE: func(cmd *cobra.Command, args []string) error {

		jobRegex, err := regexp.Compile(jobRegex)
		if err != nil {
			return fmt.Errorf("invalid job name regex provided: %w", err)
		}

		testRegex, err := regexp.Compile(testRegex)
		if err != nil {
			return fmt.Errorf("invalid test name regex provided: %w", err)
		}

		results, err := fetchResults(healthURL)
		if err != nil {
			return err
		}

		failedTests := make(map[string][]Testcase)

		for _, job := range results.Data.SIGRetests.FailedJobLeaderBoard {
			if !jobRegex.MatchString(job.JobName) {
				continue
			}
			for _, failureURL := range job.FailureURLs {
				junitURL := constructJunitURL(failureURL)
				testsuite, err := fetchJunit(junitURL)
				if err != nil {
					return err
				}
				for _, testcase := range testsuite.Testcase {
					if testcase.Failure == nil {
						continue
					}
					if !testRegex.MatchString(testcase.Name) {
						continue
					}
					if displayOnlyURLs {
						fmt.Println(failureURL)
						continue
					}
					if displayOnlyTestNames {
						fmt.Println(testcase.Name)
						continue
					}
					testcase.URL = failureURL
					failedTests[testcase.Name] = append(failedTests[testcase.Name], testcase)
					if !countFailures {
						fmt.Println(testcase.Name)
						if displayFailures {
							fmt.Printf("%s\n", testcase.Failure)
						}
						fmt.Println(failureURL)
					}
				}
			}
		}
		if countFailures {
			failedTestsKeys := slices.Sorted(maps.Keys(failedTests))
			slices.SortFunc(failedTestsKeys, func(a, b string) int {
				return cmp.Compare(len(failedTests[a]), len(failedTests[b]))
			})
			slices.Reverse(failedTestsKeys)

			for _, name := range failedTestsKeys {
				fmt.Printf("%d\t%s\n\n", len(failedTests[name]), name)
				for _, test := range failedTests[name] {
					if displayFailures {
						fmt.Printf("\t%s\n", *test.Failure)
					}
					fmt.Printf("\t%s\n\n", test.URL)
				}
				fmt.Println("")
			}
		}
		return nil
	},
}

func init() {
	rootCmd.Flags().StringVarP(&jobRegex, "job", "j", "sig-compute$", "job name regex")
	rootCmd.Flags().StringVarP(&testRegex, "test", "t", "", "test name regex")
	rootCmd.Flags().BoolVarP(&countFailures, "count", "c", false, "Count specific test failures based on the test-name-regex positional argument")
	rootCmd.Flags().BoolVarP(&displayOnlyURLs, "url", "u", false, "display only failed job URLs")
	rootCmd.Flags().BoolVarP(&displayOnlyTestNames, "name", "n", false, "display only failed test names")
	rootCmd.Flags().BoolVarP(&displayFailures, "failures", "f", false, "print test failures")
}

func fetchResults(url string) (*Results, error) {
	resp, err := http.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch results.json: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read results.json body: %w", err)
	}

	var results Results
	if err := json.Unmarshal(body, &results); err != nil {
		return nil, fmt.Errorf("failed to unmarshal results.json: %w", err)
	}

	return &results, nil
}

// constructJunitURL builds the JUnit URL from the original prow URL
func constructJunitURL(originalURL string) string {
	junitURL := strings.Replace(originalURL, "prow.ci.kubevirt.io//view/gs", "gcsweb.ci.kubevirt.io/gcs", 1)
	if !strings.HasSuffix(junitURL, "/") {
		junitURL += "/"
	}
	junitURL += "artifacts/junit.functest.xml"
	return junitURL
}

func fetchJunit(url string) (*Testsuite, error) {
	client := &http.Client{
		Timeout: 60 * time.Second, // Increased timeout to 60 seconds
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return nil
		},
		Transport: &http.Transport{
			DialContext: (&net.Dialer{
				Timeout:   15 * time.Second,
				KeepAlive: 30 * time.Second,
			}).DialContext,
			MaxIdleConns:        100,
			IdleConnTimeout:     90 * time.Second,
			TLSHandshakeTimeout: 15 * time.Second,
		},
	}

	resp, err := client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch junit.functest.xml: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to fetch junit.functest.xml: status code %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read junit.functest.xml body: %w", err)
	}

	var testsuite Testsuite
	if err := xml.Unmarshal(body, &testsuite); err == nil {
		return &testsuite, nil
	}

	return nil, fmt.Errorf("failed to unmarshal junit.functest.xml as <testsuites> or <testsuite>")
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}
