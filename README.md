# healthcheck

A command line tool to analyze KubeVirt CI failures using two complementary data sources and analysis approaches.

## Data Sources & Approaches

This tool provides two distinct commands that use different data sources:

### **`merge`** - CI-Health Aggregated Data
- **Data Source**: Uses pre-aggregated failure data from [kubevirt/ci-health](https://github.com/kubevirt/ci-health) JSON API
- **Coverage**: Analyzes failures across all merge-time jobs (main branch, release branches)
- **Time Range**: Limited to the data available in ci-health (typically recent failures)
- **Performance**: Fast - processes pre-computed aggregations
- **Use Case**: Quick overview of current CI health across all job types

### **`lane`** - Live Prow Data Crawling  
- **Data Source**: Crawls live Prow web pages and fetches individual job artifacts
- **Coverage**: Analyzes any specific job lane in real-time
- **Time Range**: Flexible - can go back weeks/months with automatic pagination
- **Performance**: Slower - fetches and parses individual job data on-demand
- **Use Case**: Deep dive analysis of specific job lanes with historical data

## Installation

```shell
go build
./healthcheck --help
```

---

## Lane Command - Live Prow Data Analysis

Analyze recent job runs for a specific CI lane by crawling live Prow web pages and artifacts. Provides real-time data with flexible time ranges and automatic pagination.

### Basic Usage

```shell
# Analyze recent runs for a specific job
$ healthcheck lane pull-kubevirt-unit-test-arm64

# Limit to specific number of runs (default: 10)
$ healthcheck lane pull-kubevirt-e2e-k8s-1.32-sig-compute --limit 20
```

### Output Formats

```shell
# Count test failures across runs
$ healthcheck lane pull-kubevirt-unit-test-arm64 --limit 5 -c
2	VirtualMachineInstance migration target DomainNotifyServerRestarts should establish a notify server pipe should be resilient to notify server restarts

	https://prow.ci.kubevirt.io/view/gs/kubevirt-prow/pr-logs/pull/kubevirt_kubevirt/15455/pull-kubevirt-unit-test-arm64/1958202806657617920

	https://prow.ci.kubevirt.io/view/gs/kubevirt-prow/pr-logs/pull/kubevirt_kubevirt/15447/pull-kubevirt-unit-test-arm64/1958193812496977920

1	Migration watcher Migration backoff should not be applied if it is not an evacuation with workload update annotation

	https://prow.ci.kubevirt.io/view/gs/kubevirt-prow/pr-logs/pull/kubevirt_kubevirt/15388/pull-kubevirt-unit-test-arm64/1958193968416034816

# Show only test names
$ healthcheck lane pull-kubevirt-unit-test-arm64 --limit 3 -n
VirtualMachineInstance migration target DomainNotifyServerRestarts should establish a notify server pipe should be resilient to notify server restarts
Migration watcher Migration backoff should not be applied if it is not an evacuation with workload update annotation

# Show only failed job URLs
$ healthcheck lane pull-kubevirt-unit-test-arm64 --limit 3 -u
https://prow.ci.kubevirt.io/view/gs/kubevirt-prow/pr-logs/pull/kubevirt_kubevirt/15455/pull-kubevirt-unit-test-arm64/1958202806657617920
https://prow.ci.kubevirt.io/view/gs/kubevirt-prow/pr-logs/pull/kubevirt_kubevirt/15388/pull-kubevirt-unit-test-arm64/1958193968416034816

# Show failure details with context
$ healthcheck lane pull-kubevirt-unit-test-arm64 --limit 3 -f
VirtualMachineInstance migration target DomainNotifyServerRestarts should establish a notify server pipe should be resilient to notify server restarts
goroutine 1847 [running]:
testing.tRunner.func1.2({0x2b2e5a0, 0xc001638690})
	/opt/hostedtoolcache/go/1.21.13/x64/lib/go/src/testing/testing.go:1631 +0x2ff
...

https://prow.ci.kubevirt.io/view/gs/kubevirt-prow/pr-logs/pull/kubevirt_kubevirt/15455/pull-kubevirt-unit-test-arm64/1958202806657617920
```

### Time-Based Analysis (Automatic Pagination)

The `--since` flag automatically paginates to find ALL results within the time period, ignoring `--limit`.

```shell
# Find all failures in the last hour
$ healthcheck lane pull-kubevirt-unit-test-arm64 --since 1h -c
1	VirtualMachineInstance migration target DomainNotifyServerRestarts should establish a notify server pipe should be resilient to notify server restarts

	https://prow.ci.kubevirt.io/view/gs/kubevirt-prow/pr-logs/pull/kubevirt_kubevirt/15455/pull-kubevirt-unit-test-arm64/1958202806657617920

# Analyze longer time periods - automatically finds all results
$ healthcheck lane pull-kubevirt-unit-test-arm64 --since 2d --summary
Lane Summary: pull-kubevirt-unit-test-arm64
===========================================

Time Range:
  First Run:  2025-08-18 19:03:13 UTC
  Last Run:   2025-08-20 16:22:12 UTC
  Duration:   1.9 days

Test Run Statistics:
  Total Runs:     92
  Successful:     62
  Failed:         15
  Unknown:        15
  Failure Rate:   16.3%

Test Failure Statistics:
  Total Failures: 78
  Unique Tests:   70

Failure Categories:
  migration : 8 (10.3%)
  general   : 3 (3.8%)
  storage   : 2 (2.6%)

Most Frequent Failures:
  1. [migration] VirtualMachineInstance migration target DomainNotifyServe... (8 failures, 10.3%)
  2. [general] VirtualMachineInstance watcher On valid VirtualMachineIns... (2 failures, 2.6%)
  3. [storage] VirtualMachineInstance watcher On valid VirtualMachineIns... (1 failures, 1.3%)

Pattern Analysis:
  🟢 Very low failure rate - stable
  🔀 Diverse failure patterns - no clear dominant issue

# Time period examples
$ healthcheck lane pull-kubevirt-e2e-k8s-1.32-sig-compute --since 6h    # Last 6 hours
$ healthcheck lane pull-kubevirt-unit-test-arm64 --since 3d             # Last 3 days  
$ healthcheck lane pull-kubevirt-e2e-k8s-1.31-sig-storage --since 1w    # Last week
```

### Summary Analysis

```shell
# Get comprehensive failure pattern analysis
$ healthcheck lane pull-kubevirt-unit-test-arm64 --limit 25 --summary
Lane Summary: pull-kubevirt-unit-test-arm64
===========================================

Time Range:
  First Run:  2025-08-20 09:48:24 UTC
  Last Run:   2025-08-20 16:22:12 UTC
  Duration:   6.6 hours

Test Run Statistics:
  Total Runs:     25
  Successful:     16
  Failed:         7
  Unknown:        2
  Failure Rate:   28.0%

Test Failure Statistics:
  Total Failures: 28
  Unique Tests:   25

Failure Categories:
  migration : 4 (14.3%)
  general   : 1 (3.6%)
  storage   : 3 (10.7%)

Most Frequent Failures:
  1. [migration] VirtualMachineInstance migration target DomainNotifyServe... (4 failures, 14.3%)
  2. [general] VirtualMachineInstance watcher On valid VirtualMachineIns... (1 failures, 3.6%)
  3. [storage] VirtualMachineInstance watcher Aggregating DataVolume con... (1 failures, 3.6%)

Pattern Analysis:
  🟠 Low failure rate - normal fluctuation
  🔀 Diverse failure patterns - no clear dominant issue
```

---

## Merge Command - CI-Health Aggregated Analysis  

Analyze test failures across all merge-time jobs using pre-computed data from the ci-health project. Fast analysis of current CI health trends.

### Job Filtering

```shell
# Filter by job regex patterns
$ healthcheck merge -j compute                    # sig-compute jobs
$ healthcheck merge -j "sig-compute.*arm64"       # ARM64 compute jobs
$ healthcheck merge -j network                    # sig-network jobs
$ healthcheck merge -j "1.6"                      # release-1.6 jobs
$ healthcheck merge -j main                       # main branch jobs

# Available job aliases:
# - main: main branch jobs
# - compute: sig-compute related jobs  
# - network: sig-network jobs
# - storage: sig-storage jobs
# - 1.6, 1.5, 1.4: release branch jobs
```

### Output Formats

```shell
# Count failures by test name
$ healthcheck merge -c -j compute
3	[sig-compute]VirtualMachinePool should respect maxUnavailable strategy during updates

	https://prow.ci.kubevirt.io//view/gs/kubevirt-prow/pr-logs/pull/kubevirt_kubevirt/15098/pull-kubevirt-e2e-k8s-1.32-sig-compute/1944655730044833792

	https://prow.ci.kubevirt.io//view/gs/kubevirt-prow/pr-logs/pull/kubevirt_kubevirt/15182/pull-kubevirt-e2e-k8s-1.31-sig-compute/1945105449749581824

2	[virtctl] [crit:medium][vendor:cnv-qe@redhat.com][level:component][sig-compute] usbredir Should work several times

	https://prow.ci.kubevirt.io//view/gs/kubevirt-prow/pr-logs/pull/kubevirt_kubevirt/15110/pull-kubevirt-e2e-k8s-1.32-sig-compute/1943363976574275584

# Show only test names for external processing
$ healthcheck merge -n -j compute | head -5
[sig-compute]VirtualMachinePool should respect maxUnavailable strategy during updates
[sig-compute] Infrastructure cluster profiler for pprof data aggregation when ClusterProfiler configuration is enabled it should allow subresource access
[virtctl] [crit:medium][vendor:cnv-qe@redhat.com][level:component][sig-compute] usbredir Should work several times
[sig-compute]VirtualMachinePool pool should scale to five, to six and then to zero replicas
[sig-compute] [rfe_id:1177][crit:medium] VirtualMachine with paused vmi [test_id:3229]should gracefully handle being started again

# Show only URLs for browser opening
$ healthcheck merge -u -j compute | head -3
https://prow.ci.kubevirt.io//view/gs/kubevirt-prow/pr-logs/pull/kubevirt_kubevirt/15098/pull-kubevirt-e2e-k8s-1.32-sig-compute/1944655730044833792
https://prow.ci.kubevirt.io//view/gs/kubevirt-prow/pr-logs/pull/kubevirt_kubevirt/15182/pull-kubevirt-e2e-k8s-1.31-sig-compute/1945105449749581824
https://prow.ci.kubevirt.io//view/gs/kubevirt-prow/pr-logs/pull/kubevirt_kubevirt/15122/pull-kubevirt-e2e-k8s-1.33-sig-compute/1943094557549793280

# Show failure context and stack traces
$ healthcheck merge -c -f -j compute
3	[sig-compute]VirtualMachinePool should respect maxUnavailable strategy during updates

	Failure tests/pool_test.go:701
	Expected
	    <int>: 3
	to equal
	    <int>: 4
	tests/pool_test.go:760

	https://prow.ci.kubevirt.io//view/gs/kubevirt-prow/pr-logs/pull/kubevirt_kubevirt/15098/pull-kubevirt-e2e-k8s-1.32-sig-compute/1944655730044833792
```

### Advanced Features

```shell
# Group by lane run UUID for failure correlation
$ healthcheck merge --lane-run -j compute
Lane Run 1944655730044833792 (3 failures)

	[sig-compute]VirtualMachinePool should respect maxUnavailable strategy during updates
	https://prow.ci.kubevirt.io//view/gs/kubevirt-prow/pr-logs/pull/kubevirt_kubevirt/15098/pull-kubevirt-e2e-k8s-1.32-sig-compute/1944655730044833792

# Highlight quarantined tests
$ healthcheck merge -c -j compute --quarantine
2	[QUARANTINED] [sig-compute] should include VMI infos for a running VM

	https://prow.ci.kubevirt.io//view/gs/kubevirt-prow/pr-logs/pull/kubevirt_kubevirt/15098/pull-kubevirt-e2e-k8s-1.32-sig-compute/1944655730044833792

# Time filtering (limited to available ci-health data - typically last ~48 hours)
$ healthcheck merge -j compute --since 2d       # Filter by time period
```

---

## Practical Workflows

### Daily Failure Triage

```shell
# Quick overview of current failures across all jobs (ci-health data)
$ healthcheck merge -c -j compute | head -10

# Deep dive into a specific failing job with historical context (live Prow data)
$ healthcheck lane pull-kubevirt-e2e-k8s-1.32-sig-compute --since 24h --summary

# Open all failure URLs in browser tabs
$ healthcheck merge -u -j compute | sort | uniq | xargs google-chrome
```

### Trend Analysis

```shell
# Compare failure rates over different time periods (live Prow data)
$ healthcheck lane pull-kubevirt-unit-test-arm64 --since 24h --summary
$ healthcheck lane pull-kubevirt-unit-test-arm64 --since 1w --summary

# Identify most frequent failures across all jobs (ci-health data)
$ healthcheck merge -n | sort | uniq -c | sort -rn | head -10
```

### Debugging Specific Issues

```shell
# Find all occurrences of a specific test failure
$ healthcheck merge -n | grep -i "migration"

# Get failure context for debugging
$ healthcheck merge -f -j compute | grep -A5 -B5 "timeout"

# Analyze quarantined tests
$ healthcheck merge --quarantine -c
```

### CI Health Monitoring

```shell
# Monitor overall health of different job categories (live Prow data with historical context)
$ healthcheck lane pull-kubevirt-e2e-k8s-1.32-sig-compute --since 1d --summary
$ healthcheck lane pull-kubevirt-e2e-k8s-1.32-sig-network --since 1d --summary  
$ healthcheck lane pull-kubevirt-e2e-k8s-1.32-sig-storage --since 1d --summary

# Track specific job stability over time (weeks of historical data)
$ healthcheck lane pull-kubevirt-unit-test-arm64 --since 1w --summary
```

## Command Reference

### Lane Command Flags (Live Prow Data)

- `--limit, -l`: Number of recent runs to analyze (ignored when --since is used)
- `--since, -s`: Fetch all results within time period (e.g., 24h, 2d, 1w) with automatic pagination
- `--count, -c`: Count specific test failures
- `--url, -u`: Display only failed job URLs
- `--name, -n`: Display only failed test names  
- `--failures, -f`: Print captured failure context
- `--summary`: Display concise summary with failure patterns and statistics

### Merge Command Flags (CI-Health Data)

- `--job, -j`: Filter by job regex or alias (compute, network, storage, main, 1.6, 1.5, 1.4)
- `--test, -t`: Filter by test name regex
- `--count, -c`: Count specific test failures
- `--url, -u`: Display only failure URLs
- `--name, -n`: Display only test names
- `--failures, -f`: Print captured failure context
- `--lane-run`: Group failures by lane run UUID
- `--quarantine`: Highlight quarantined tests
- `--since, -s`: Filter results by time period (limited to available ci-health data ~48h)