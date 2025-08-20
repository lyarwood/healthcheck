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
- `./healthcheck lane pull-kubevirt-unit-test-arm64` - Analyze recent runs for a specific job
- `./healthcheck lane pull-kubevirt-e2e-k8s-1.32-sig-compute --limit 5` - Analyze 5 recent runs
- `./healthcheck lane pull-kubevirt-unit-test-arm64 -c` - Count failures in recent runs
- `./healthcheck lane pull-kubevirt-unit-test-arm64 -n` - Show only test names
- `./healthcheck lane pull-kubevirt-unit-test-arm64 -u` - Show only job URLs
- `./healthcheck lane pull-kubevirt-unit-test-arm64 --since 24h` - Show failures from last 24 hours
- `./healthcheck lane pull-kubevirt-unit-test-arm64 --summary` - Show concise summary with patterns
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
6. `get_failure_source_context` - Parse junit failures and generate GitHub URLs

### Integration Points

- All tools reuse existing healthcheck package functionality
- Data formats are optimized for LLM consumption
- JSON responses include health status, trends, and recommendations
- Comprehensive error handling for robust AI integration