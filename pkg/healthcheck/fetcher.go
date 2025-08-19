package healthcheck

import (
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"net"
	"net/http"
	"regexp"
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
