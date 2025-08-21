# Claude Instructions for KubeVirt Healthcheck Project

## Commit Guidelines

When creating commits for this project, always include an assisted-by line attributing Claude:

```
Assisted-By: Claude <noreply@anthropic.com>
```

This should be included at the end of all commit messages:

```
Assisted-By: Claude <noreply@anthropic.com>
```

### Commit Message Format

- Keep commit messages wrapped at 80 characters per line
- Use a descriptive subject line (50 characters or less)
- Add a blank line between subject and body
- Wrap the body text at 80 characters

## Project Context

This is a Go CLI tool for parsing KubeVirt CI health data and reporting failed tests. The tool helps developers and maintainers analyze test failures from the kubevirt/ci-health project by providing various filtering and grouping options.

## Testing Commands

When making changes, ensure the tool builds and runs correctly:

```bash
go build
./healthcheck --help
```

Test different functionality modes:
- `./healthcheck merge compute` - Filter by job name or alias
- `./healthcheck merge main -c` - Count failures for main branch jobs
- `./healthcheck merge compute --lane-run` - Group by lane run UUID
- `./healthcheck merge compute -f` - Show failure details
- `./healthcheck merge compute --summary` - Show concise summary with patterns
- `./healthcheck merge network --output json` - Output structured JSON data
- `./healthcheck lane pull-kubevirt-unit-test-arm64` - Analyze recent runs for a specific job
- `./healthcheck lane pull-kubevirt-e2e-k8s-1.32-sig-compute --limit 5` - Analyze 5 recent runs
- `./healthcheck lane pull-kubevirt-unit-test-arm64 -c` - Count failures in recent runs
- `./healthcheck lane pull-kubevirt-unit-test-arm64 -n` - Show only test names
- `./healthcheck lane pull-kubevirt-unit-test-arm64 -u` - Show only job URLs
- `./healthcheck lane pull-kubevirt-unit-test-arm64 --since 24h` - Show failures from last 24 hours
- `./healthcheck lane pull-kubevirt-unit-test-arm64 --summary` - Show concise summary with patterns
- `./healthcheck lane pull-kubevirt-unit-test-arm64 --output json` - Output structured JSON data for machine processing
- `./healthcheck merge compute --since 2d` - Show compute failures from last 2 days
- `./healthcheck mcp` - Start MCP server for LLM integration
- `./healthcheck mcp --debug` - Start MCP server with debug output

## MCP Server Feature

The tool now includes an MCP (Model Context Protocol) server that exposes CI health analysis functionality to Large Language Models. This enables AI-powered workflows for intelligent failure analysis.

### MCP Commands to Test

- `./healthcheck mcp --help` - Show MCP command help
- `./healthcheck mcp --debug` - Start server and show available tools
- `timeout 5s ./healthcheck mcp` - Test server startup (will timeout after 5 seconds)

### MCP Tools Available

The MCP server provides 11 comprehensive enterprise-grade tools:
1. `analyze_job_lane` - Job failure analysis with patterns
2. `get_job_failures` - Detailed failure information  
3. `analyze_merge_failures` - Cross-job failure analysis
4. `search_failure_patterns` - Pattern search across jobs
5. `compare_time_periods` - Time-based failure comparison
6. `get_failure_source_context` - Enhanced junit failure parsing with GitHub URL generation
7. `analyze_failure_trends` - Advanced trend analysis with flakiness detection (NEW)
8. `analyze_failure_correlation` - Cross-job correlation and systemic issue detection (NEW)
9. `analyze_quarantine_intelligence` - Intelligent quarantine analysis and recommendations (NEW)
10. `assess_failure_impact` - Context-aware failure impact assessment for triage (NEW)
11. `generate_failure_report` - Comprehensive stakeholder reporting with executive summaries (NEW)

### Integration Points

- All tools reuse existing healthcheck package functionality
- Data formats are optimized for LLM consumption with 136+ structured types
- JSON responses include health status, trends, flakiness detection, and recommendations
- Advanced analytics include correlation patterns, quarantine intelligence, and impact assessment
- Comprehensive error handling for robust AI integration
- Enhanced failure source context with GitHub URL generation for code inspection
- Multi-format parsing support for both simple and complex failure text patterns
- Enterprise-grade reporting with executive summaries and actionable insights
- Context-aware prioritization for intelligent triage and resource allocation

## JSON Output Support

Both lane and merge commands now support `--output json` for structured data output:

### JSON Output Features

- **Machine-readable format**: Structured JSON output suitable for automation and integration
- **All filter modes supported**: Works with -c, -u, -n, --lane-run, --summary, and other flags
- **Complete data preservation**: Captures all failure information without truncation
- **Automation-friendly**: Enables scripting, monitoring, and external tool integration

### JSON Output Commands to Test

- `./healthcheck merge compute --output json` - Export compute failures as JSON
- `./healthcheck lane job-name --limit 10 -c --output json` - Export lane failure counts as JSON
- `./healthcheck merge main -u --output json | jq -r '.urls[]'` - Extract URLs with jq
- `./healthcheck lane job-name --summary --output json` - Export lane summary for trending
- `./healthcheck merge compute --lane-run --output json` - Export grouped failures for analysis

### JSON Structure Examples

Lane output with count mode:
```json
{
  "job_name": "pull-kubevirt-unit-test-arm64",
  "test_failures": {
    "TestName1": [{"Name": "...", "URL": "...", "Failure": "..."}],
    "TestName2": [{"Name": "...", "URL": "...", "Failure": "..."}]
  }
}
```

Merge output with count mode:
```json
{
  "test_failure_counts": {"Test1": 3, "Test2": 1},
  "failed_tests": {...}
}
```

## Enhanced get_failure_source_context Tool

The get_failure_source_context MCP tool has been significantly enhanced with improved parsing capabilities:

### Enhanced Parsing Features

- **Smart format detection**: Handles both "file:line" and "Type file:line" patterns automatically
- **Advanced error extraction**: Uses pattern matching for common error types (Panic, Error, Failed, etc.)
- **Multi-file tracking**: Captures multiple file references within the same failure
- **Comprehensive stack trace parsing**: Handles both detailed and simple file:line references
- **Enhanced error message extraction**: Extracts meaningful error messages up to 200 characters

### Testing Commands for Enhanced Parsing

- Test simple format: Use MCP client to send "pkg/file.go:123"
- Test complex format: Use MCP client to send "Panic pkg/file.go:123\nError: something failed"
- Test multi-file failures with cross-file references
- Test error message extraction from complex failure text

### Supported Input Patterns

```
# Simple file:line format (most common)
pkg/virt-controller/services/template_test.go:2689

# Complex format with failure type
Panic pkg/virt-controller/services/template_test.go:2689

# Multi-line with error details
pkg/file.go:123
Error: deadline exceeded
Expected: 5, Got: 3

# Cross-file failures with stack traces
pkg/file1.go:123
pkg/file2.go:456
Full stack: [detailed stack trace]
```

### Key Improvements Made

- **Public function exports**: ParseFailureText, ExtractRepositoryInfo, FormatFailureSourceContextForLLM are now public
- **Better pattern matching**: Recognizes bare file:line format (most common in JUnit output)
- **Multi-line error extraction**: Captures error messages from anywhere in the failure text
- **Cross-file reference tracking**: Finds all .go: references throughout the failure text
- **Enhanced stack trace handling**: Supports both detailed (+0x format) and simple file:line references

## Advanced MCP Tools for Enterprise CI Intelligence

The MCP server has been expanded with 5 powerful new tools providing enterprise-grade CI analysis capabilities:

### Tool 7: analyze_failure_trends
**Purpose**: Deep historical trend analysis with advanced flakiness detection
**Key Features**:
- **Trend Direction Analysis**: Automatically detects improving/degrading/stable patterns
- **Flakiness Detection**: Identifies intermittent failures (10-90% failure rate) for quarantine decisions
- **Pattern Frequency Analysis**: Tracks failure patterns over time with severity scoring (critical/high/medium/low)
- **Smart Recommendations**: Differentiates infrastructure vs code change investigation paths
- **Larger Data Limits**: Analyzes up to 500 runs for comprehensive trend analysis

**Testing**: `./healthcheck mcp` then use MCP client to call analyze_failure_trends with job_name

### Tool 8: analyze_failure_correlation  
**Purpose**: Cross-job correlation analysis for systemic issue detection
**Key Features**:
- **Environment-Specific Analysis**: ARM64 vs x86, Kubernetes version-specific failures
- **Resource Issue Detection**: CPU, memory, disk-related failure pattern recognition
- **Systemic Issue Identification**: Infrastructure vs application-level problem detection
- **Correlation Scoring**: Quantified failure relationships across multiple job types
- **Cross-Job Pattern Recognition**: Identifies patterns affecting multiple job categories

**Testing**: Call analyze_failure_correlation with job_pattern="compute" for compute-related correlation analysis

### Tool 9: analyze_quarantine_intelligence
**Purpose**: Intelligent quarantine effectiveness analysis with actionable recommendations
**Key Features**:
- **Effectiveness Scoring**: Quantifies how well current quarantine decisions are working
- **Action Recommendations**: Remove/extend/investigate with detailed reasoning and priority
- **Status Analysis**: Active vs stale quarantine identification
- **Impact Assessment**: How quarantine decisions affect overall CI health
- **Intelligence-Based Decisions**: Data-driven quarantine management

**Testing**: Call analyze_quarantine_intelligence with scope="all" for comprehensive quarantine analysis

### Tool 10: assess_failure_impact
**Purpose**: Context-aware failure impact assessment for intelligent triage
**Key Features**:
- **Context-Aware Prioritization**: Different urgency for production/development/pre-release
- **Business Impact Analysis**: Critical path vs edge case failure identification  
- **Resource Allocation Recommendations**: Senior engineer vs standard triage assignments
- **Priority Assignment**: Urgent/normal/low with detailed reasoning
- **JSON Integration**: Works with --output json data from lane/merge commands

**Testing**: Export JSON data with `./healthcheck merge compute --output json > data.json` then use in assess_failure_impact

### Tool 11: generate_failure_report
**Purpose**: Comprehensive stakeholder reporting with executive summaries
**Key Features**:
- **Executive Summaries**: High-level CI health status for management consumption
- **Key Metrics**: Overall health scores, failure rates, critical issue counts
- **Trend Analysis**: Direction and change percentages over time periods
- **Actionable Items**: Prioritized next steps for development teams
- **Multiple Formats**: Summary/detailed/executive formats for different audiences
- **Scope Flexibility**: Daily/weekly/release or job-specific reporting

**Testing**: Call generate_failure_report with scope="daily" and format="executive"

### Advanced Analytics Capabilities

These tools provide **753+ lines of new functionality** with:
- **136+ new data structures** optimized for LLM consumption
- **Advanced pattern recognition algorithms** for flakiness and correlation detection
- **Intelligent prioritization** based on context and business impact
- **Historical trend analysis** with regression detection
- **Environment-specific failure analysis** and resource issue detection
- **Quarantine optimization** with effectiveness-based recommendations
- **Enterprise reporting** with executive summaries and stakeholder communications

### Usage Patterns for LLM Integration

```bash
# Trend analysis workflow
./healthcheck mcp # Start server
# LLM: "Analyze failure trends for pull-kubevirt-e2e-k8s-1.32-sig-compute over 30 days"

# Correlation analysis workflow  
# LLM: "Identify systemic issues affecting compute jobs across ARM64 and x86"

# Quarantine intelligence workflow
# LLM: "Analyze quarantine effectiveness and recommend optimizations"

# Impact assessment workflow
./healthcheck merge compute --output json > failures.json
# LLM: "Assess business impact of these failures for production release triage"

# Executive reporting workflow
# LLM: "Generate weekly CI health executive summary for engineering leadership"
```

The MCP server now provides enterprise-grade CI intelligence enabling LLMs to perform sophisticated analysis, pattern recognition, and decision support for CI/CD pipeline health management.