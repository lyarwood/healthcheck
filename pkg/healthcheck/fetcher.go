package healthcheck

import (
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"net"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"
)

const HealthURL = "https://kubevirt.io/ci-health/output/kubevirt/kubevirt/results.json"
const QuarantinedTestsURL = "https://storage.googleapis.com/kubevirt-prow/reports/" +
	"quarantined-tests/kubevirt/kubevirt/index.html"

func FetchResults(url string) (*Results, error) {
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

func fetchTestSuite(failureURL string) (*Testsuite, error) {
	url := constructJunitURL(failureURL)
	client := &http.Client{
		Timeout: 60 * time.Second, // Increased timeout to 60 seconds
		CheckRedirect: func(_ *http.Request, _ []*http.Request) error {
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
		return nil, fmt.Errorf("failed to fetch %s: %w", url, err)
	}
	defer resp.Body.Close()

	// Ignore missing junit files as it suggests an issue with the job
	if resp.StatusCode == http.StatusNotFound {
		return nil, nil
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to fetch %s: status code %d", url, resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read %s body: %w", url, err)
	}

	var testsuite Testsuite
	if err := xml.Unmarshal(body, &testsuite); err == nil {
		return &testsuite, nil
	}

	return nil, fmt.Errorf("failed to unmarshal junit.functest.xml as <testsuites> or <testsuite>")
}

// FetchQuarantinedTests fetches the list of quarantined test names from the kubevirt prow reports
func FetchQuarantinedTests() (map[string]bool, error) {
	resp, err := http.Get(QuarantinedTestsURL)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch quarantined tests: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to fetch quarantined tests: status code %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read quarantined tests body: %w", err)
	}

	return parseQuarantinedTests(string(body)), nil
}

// parseQuarantinedTests extracts test names from the quarantined tests HTML page
func parseQuarantinedTests(htmlContent string) map[string]bool {
	quarantinedTests := make(map[string]bool)

	// Pre-defined list based on the current quarantined tests
	// This could be enhanced to parse the HTML dynamically in the future
	knownQuarantined := []string{
		"should include VMI infos for a running VM",
		"it should fetch logs for a running VM with logs API",
		"it should not skip any log line even trying to flood the serial console for QOSGuaranteed VMs",
		"should report an error status when image pull error occurs",
		"should have kubevirt_vmi_info correctly configured with guest OS labels",
		"Should force restart a VM with terminationGracePeriodSeconds>0",
		"should permanently add hotplug volume when added to VM, but still unpluggable after restart",
		"should live migrate a container disk vm, with an additional PVC mounted, should stay mounted after migration",
		"should live migrate regular disk several times",
		"should live migrate a container disk vm, several times",
		"should migrate with a downwardMetrics channel",
		"should successfully upgrade virt-handler",
		"should run guest attestation",
	}

	for _, test := range knownQuarantined {
		quarantinedTests[test] = true
	}

	// Also try to extract from HTML content for dynamic parsing
	// Look for patterns that might contain test names
	re := regexp.MustCompile(`(?i)(?:\[QUARANTINE\]|\[test_id:\d+\])\s*([^<\[\n]+)`)
	matches := re.FindAllStringSubmatch(htmlContent, -1)

	for _, match := range matches {
		if len(match) > 1 {
			testName := strings.TrimSpace(match[1])
			if testName != "" && !strings.Contains(testName, "[") {
				quarantinedTests[testName] = true
			}
		}
	}

	return quarantinedTests
}

// FetchJobHistory fetches recent job runs from the Prow job history page with pagination support
func FetchJobHistory(jobName string, limit int) ([]JobRun, error) {
	var allRuns []JobRun
	baseURL := fmt.Sprintf("https://prow.ci.kubevirt.io/job-history/gs/kubevirt-prow/pr-logs/directory/%s", jobName)
	currentURL := baseURL
	
	for len(allRuns) < limit {
		// Fetch current page
		resp, err := http.Get(currentURL)
		if err != nil {
			return nil, fmt.Errorf("failed to fetch job history: %w", err)
		}

		if resp.StatusCode != http.StatusOK {
			resp.Body.Close()
			return nil, fmt.Errorf("failed to fetch job history: status code %d", resp.StatusCode)
		}

		body, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			return nil, fmt.Errorf("failed to read job history body: %w", err)
		}

		// Parse this page's job runs
		pageRuns, nextBuildID, err := parseJobHistoryPage(string(body), jobName)
		if err != nil {
			return nil, err
		}

		// Add runs from this page (up to our limit)
		remaining := limit - len(allRuns)
		if len(pageRuns) <= remaining {
			allRuns = append(allRuns, pageRuns...)
		} else {
			allRuns = append(allRuns, pageRuns[:remaining]...)
			break
		}

		// If we have enough runs or no more pages, stop
		if len(allRuns) >= limit || nextBuildID == "" {
			break
		}

		// Prepare URL for next page
		currentURL = fmt.Sprintf("%s?buildId=%s", baseURL, nextBuildID)
	}

	return allRuns, nil
}

// parseJobHistoryPage extracts job run information from a single Prow history HTML page
func parseJobHistoryPage(htmlContent, jobName string) ([]JobRun, string, error) {
	var runs []JobRun

	// Look for allBuilds JSON array in the JavaScript
	re := regexp.MustCompile(`allBuilds\s*=\s*(\[.*?\]);`)
	match := re.FindStringSubmatch(htmlContent)
	
	if len(match) < 2 {
		return runs, "", fmt.Errorf("could not find allBuilds JSON in page content")
	}

	// Parse the JSON array
	var buildData []map[string]interface{}
	if err := json.Unmarshal([]byte(match[1]), &buildData); err != nil {
		return runs, "", fmt.Errorf("failed to parse builds JSON: %w", err)
	}

	var nextBuildID string
	for _, build := range buildData {
		// Extract build information
		buildID, ok := build["ID"].(string)
		if !ok {
			continue
		}

		spyglassLink, ok := build["SpyglassLink"].(string)
		if !ok {
			continue
		}

		// Convert SpyglassLink to prow URL format
		runURL := spyglassLink
		if !strings.HasPrefix(runURL, "https://") {
			if strings.HasPrefix(runURL, "/") {
				runURL = "https://prow.ci.kubevirt.io" + runURL
			} else {
				runURL = "https://prow.ci.kubevirt.io/" + runURL
			}
		}

		// Extract timestamp if available
		timestamp := ""
		if started, ok := build["Started"].(string); ok {
			timestamp = started
		}

		run := JobRun{
			ID:        buildID,
			URL:       runURL,
			Timestamp: timestamp,
		}

		runs = append(runs, run)
		
		// Keep track of the last (oldest) buildID for pagination
		nextBuildID = buildID
	}

	return runs, nextBuildID, nil
}

// fetchJobArtifacts fetches test results from a specific job run's artifacts
func fetchJobArtifacts(jobRun *JobRun) error {
	// Convert prow URL to direct Google Storage URL
	artifactsURL := strings.Replace(jobRun.URL, "prow.ci.kubevirt.io/view/gs", "storage.googleapis.com", 1)
	if !strings.HasSuffix(artifactsURL, "/") {
		artifactsURL += "/"
	}
	
	// Try different possible junit file locations
	junitPaths := []string{
		"artifacts/junit/junit.unittests.xml",  // Unit tests
		"artifacts/junit.functest.xml",         // Functional tests
		"artifacts/junit.xml",                  // Generic
		"artifacts/tests/junit.xml",            // Alternative location
	}

	for _, path := range junitPaths {
		junitURL := artifactsURL + path
		testsuite, err := fetchTestSuiteFromURL(junitURL)
		if err == nil && testsuite != nil {
			// Extract failed tests
			for _, testcase := range testsuite.Testcase {
				if testcase.Failure != nil {
					testcase.URL = jobRun.URL
					jobRun.Failures = append(jobRun.Failures, testcase)
				}
			}
			
			// Determine job status
			if len(jobRun.Failures) > 0 {
				jobRun.Status = "FAILURE"
			} else {
				jobRun.Status = "SUCCESS"
			}
			return nil
		}
	}

	// If no junit file found, check if job failed for other reasons
	jobRun.Status = "UNKNOWN"
	return nil
}

// fetchTestSuiteFromURL fetches and parses a junit XML file from a specific URL
func fetchTestSuiteFromURL(url string) (*Testsuite, error) {
	client := &http.Client{
		Timeout: 30 * time.Second,
		CheckRedirect: func(_ *http.Request, _ []*http.Request) error {
			return nil
		},
	}

	resp, err := client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch %s: %w", url, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, nil
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to fetch %s: status code %d", url, resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read %s body: %w", url, err)
	}

	var testsuite Testsuite
	if err := xml.Unmarshal(body, &testsuite); err == nil {
		return &testsuite, nil
	}

	return nil, fmt.Errorf("failed to unmarshal junit XML from %s", url)
}

// ParseTimePeriod parses time period strings like "24h", "2d", "1w" into a time.Duration
func ParseTimePeriod(period string) (time.Duration, error) {
	if period == "" {
		return 0, nil
	}

	re := regexp.MustCompile(`^(\d+)([hdw])$`)
	matches := re.FindStringSubmatch(strings.ToLower(period))
	
	if len(matches) != 3 {
		return 0, fmt.Errorf("invalid time period format: %s (expected format: 24h, 2d, 1w)", period)
	}

	value, err := strconv.Atoi(matches[1])
	if err != nil {
		return 0, fmt.Errorf("invalid number in time period: %s", matches[1])
	}

	unit := matches[2]
	switch unit {
	case "h":
		return time.Duration(value) * time.Hour, nil
	case "d":
		return time.Duration(value) * 24 * time.Hour, nil
	case "w":
		return time.Duration(value) * 7 * 24 * time.Hour, nil
	default:
		return 0, fmt.Errorf("invalid time unit: %s (supported: h, d, w)", unit)
	}
}

// IsWithinTimePeriod checks if a timestamp is within the given time period from now
func IsWithinTimePeriod(timestamp string, period time.Duration) bool {
	if period == 0 {
		return true // No time filter
	}

	// Parse the timestamp (ISO 8601 format from Prow)
	t, err := time.Parse(time.RFC3339, timestamp)
	if err != nil {
		// If we can't parse the timestamp, include it to be safe
		return true
	}

	cutoff := time.Now().UTC().Add(-period)
	return t.After(cutoff)
}
