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
- `./healthcheck merge -j compute` - Filter by job regex
- `./healthcheck merge -c` - Count failures
- `./healthcheck merge --lane-run` - Group by lane run UUID
- `./healthcheck merge -f` - Show failure details
- `./healthcheck merge --output json` - Output structured JSON data
- `./healthcheck lane pull-kubevirt-unit-test-arm64` - Analyze recent runs for a specific job
- `./healthcheck lane pull-kubevirt-e2e-k8s-1.32-sig-compute --limit 5` - Analyze 5 recent runs
- `./healthcheck lane pull-kubevirt-unit-test-arm64 -c` - Count failures in recent runs
- `./healthcheck lane pull-kubevirt-unit-test-arm64 -n` - Show only test names
- `./healthcheck lane pull-kubevirt-unit-test-arm64 -u` - Show only job URLs
- `./healthcheck lane pull-kubevirt-unit-test-arm64 --since 24h` - Show failures from last 24 hours
- `./healthcheck lane pull-kubevirt-unit-test-arm64 --summary` - Show concise summary with patterns
- `./healthcheck lane pull-kubevirt-unit-test-arm64 --output json` - Output structured JSON data for machine processing
- `./healthcheck merge -j compute --since 2d` - Show compute failures from last 2 days
- `./healthcheck mcp` - Start MCP server for LLM integration
- `./healthcheck mcp --debug` - Start MCP server with debug output

## MCP Server Feature

The tool now includes an MCP (Model Context Protocol) server that exposes CI health analysis functionality to Large Language Models. This enables AI-powered workflows for intelligent failure analysis.

### MCP Commands to Test

- `./healthcheck mcp --help` - Show MCP command help
- `./healthcheck mcp --debug` - Start server and show available tools
- `timeout 5s ./healthcheck mcp` - Test server startup (will timeout after 5 seconds)

### MCP Tools Available

The MCP server provides 6 tools:
1. `analyze_job_lane` - Job failure analysis with patterns
2. `get_job_failures` - Detailed failure information  
3. `analyze_merge_failures` - Cross-job failure analysis
4. `search_failure_patterns` - Pattern search across jobs
5. `compare_time_periods` - Time-based failure comparison
6. `get_failure_source_context` - Enhanced junit failure parsing with GitHub URL generation (IMPROVED)

### Integration Points

- All tools reuse existing healthcheck package functionality
- Data formats are optimized for LLM consumption
- JSON responses include health status, trends, and recommendations
- Comprehensive error handling for robust AI integration
- Enhanced failure source context with GitHub URL generation for code inspection
- Multi-format parsing support for both simple and complex failure text patterns

## JSON Output Support

Both lane and merge commands now support `--output json` for structured data output:

### JSON Output Features

- **Machine-readable format**: Structured JSON output suitable for automation and integration
- **All filter modes supported**: Works with -c, -u, -n, --lane-run, --summary, and other flags
- **Complete data preservation**: Captures all failure information without truncation
- **Automation-friendly**: Enables scripting, monitoring, and external tool integration

### JSON Output Commands to Test

- `./healthcheck merge -j compute --output json` - Export compute failures as JSON
- `./healthcheck lane job-name --limit 10 -c --output json` - Export lane failure counts as JSON
- `./healthcheck merge -u --output json | jq -r '.urls[]'` - Extract URLs with jq
- `./healthcheck lane job-name --summary --output json` - Export lane summary for trending
- `./healthcheck merge --lane-run --output json` - Export grouped failures for analysis

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