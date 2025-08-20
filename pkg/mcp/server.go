package mcp

import (
	"context"
	"encoding/json"
	"fmt"

	"healthcheck/pkg/healthcheck"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// HealthcheckMCPServer provides MCP tools for KubeVirt CI health analysis
type HealthcheckMCPServer struct {
	server *server.MCPServer
}

// NewHealthcheckMCPServer creates a new MCP server for healthcheck analysis
func NewHealthcheckMCPServer() *HealthcheckMCPServer {
	s := &HealthcheckMCPServer{}
	
	mcpServer := server.NewMCPServer(
		"healthcheck-mcp",
		"1.0.0",
		server.WithToolCapabilities(false), // No tool list change notifications needed
	)

	// Register all available tools
	s.registerTools(mcpServer)
	s.server = mcpServer
	
	return s
}

// Serve starts the MCP server using stdio transport
func (s *HealthcheckMCPServer) Serve() error {
	return server.ServeStdio(s.server)
}

// registerTools registers all available MCP tools
func (s *HealthcheckMCPServer) registerTools(mcpServer *server.MCPServer) {
	// Tool 1: Analyze job lane with summary
	analyzeJobLaneTool := mcp.NewTool(
		"analyze_job_lane",
		mcp.WithDescription("Analyze recent job runs for a specific CI lane with failure patterns and statistics"),
		mcp.WithString("job_name", mcp.Description("Name of the CI job to analyze"), mcp.Required()),
		mcp.WithString("since", mcp.Description("Time period to analyze (e.g., '24h', '7d', '1w')"), mcp.DefaultString("24h")),
		mcp.WithBoolean("include_details", mcp.Description("Include detailed failure information"), mcp.DefaultBool(true)),
	)
	mcpServer.AddTool(analyzeJobLaneTool, s.analyzeJobLane)

	// Tool 2: Get specific job failures with details
	getJobFailuresTool := mcp.NewTool(
		"get_job_failures",
		mcp.WithDescription("Get detailed failure information for a specific job"),
		mcp.WithString("job_name", mcp.Description("Name of the CI job"), mcp.Required()),
		mcp.WithNumber("limit", mcp.Description("Number of recent runs to analyze"), mcp.DefaultNumber(10), mcp.Min(1), mcp.Max(100)),
		mcp.WithBoolean("include_stack_traces", mcp.Description("Include failure stack traces"), mcp.DefaultBool(false)),
	)
	mcpServer.AddTool(getJobFailuresTool, s.getJobFailures)

	// Tool 3: Analyze merge failures across jobs
	analyzeMergeFailuresTool := mcp.NewTool(
		"analyze_merge_failures",
		mcp.WithDescription("Analyze test failures across all merge-time jobs using ci-health data"),
		mcp.WithString("job_filter", mcp.Description("Job filter regex or alias (compute, network, storage, main, etc.)"), mcp.DefaultString(".*")),
		mcp.WithString("test_filter", mcp.Description("Test name filter regex"), mcp.DefaultString(".*")),
		mcp.WithBoolean("include_quarantined", mcp.Description("Include quarantined test information"), mcp.DefaultBool(true)),
	)
	mcpServer.AddTool(analyzeMergeFailuresTool, s.analyzeMergeFailures)

	// Tool 4: Search for failure patterns
	searchFailurePatternsTool := mcp.NewTool(
		"search_failure_patterns",
		mcp.WithDescription("Search for specific failure patterns across jobs"),
		mcp.WithString("pattern", mcp.Description("Regex pattern to search for in test names or failure messages"), mcp.Required()),
		mcp.WithString("job_filter", mcp.Description("Job filter regex or alias"), mcp.DefaultString(".*")),
		mcp.WithString("search_in", mcp.Description("Where to search for the pattern"), mcp.Enum("test_names", "failure_messages", "both"), mcp.DefaultString("test_names")),
	)
	mcpServer.AddTool(searchFailurePatternsTool, s.searchFailurePatterns)

	// Tool 5: Compare time periods
	compareTimePeriodsool := mcp.NewTool(
		"compare_time_periods",
		mcp.WithDescription("Compare failure rates between two time periods for a job"),
		mcp.WithString("job_name", mcp.Description("Name of the CI job to analyze"), mcp.Required()),
		mcp.WithString("recent_period", mcp.Description("Recent time period (e.g., '24h', '7d')"), mcp.DefaultString("24h")),
		mcp.WithString("comparison_period", mcp.Description("Comparison time period (e.g., '7d', '14d')"), mcp.DefaultString("7d")),
	)
	mcpServer.AddTool(compareTimePeriodsool, s.compareTimePeriods)
}

// analyzeJobLane implements the analyze_job_lane tool
func (s *HealthcheckMCPServer) analyzeJobLane(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	jobName := mcp.ParseString(request, "job_name", "")
	if jobName == "" {
		return mcp.NewToolResultError("job_name parameter is required"), nil
	}

	since := mcp.ParseString(request, "since", "24h")
	includeDetails := mcp.ParseBoolean(request, "include_details", true)

	// Parse time period
	timePeriod, err := healthcheck.ParseTimePeriod(since)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Invalid time period: %v", err)), nil
	}

	// Fetch job history
	runs, err := healthcheck.FetchJobHistoryWithTimePeriod(jobName, timePeriod, 1000)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to fetch job history: %v", err)), nil
	}

	// Analyze runs
	summary, err := healthcheck.AnalyzeLaneRuns(runs)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to analyze lane runs: %v", err)), nil
	}

	// Format response for LLM
	response := formatLaneSummaryForLLM(jobName, summary, includeDetails)
	
	jsonResponse, err := json.MarshalIndent(response, "", "  ")
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to marshal response: %v", err)), nil
	}

	return mcp.NewToolResultText(string(jsonResponse)), nil
}

// getJobFailures implements the get_job_failures tool
func (s *HealthcheckMCPServer) getJobFailures(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	jobName := mcp.ParseString(request, "job_name", "")
	if jobName == "" {
		return mcp.NewToolResultError("job_name parameter is required"), nil
	}

	limit := int(mcp.ParseFloat64(request, "limit", 10))
	includeStackTraces := mcp.ParseBoolean(request, "include_stack_traces", false)

	// Fetch job history
	runs, err := healthcheck.FetchJobHistory(jobName, limit)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to fetch job history: %v", err)), nil
	}

	// Format detailed failure information
	response := formatJobFailuresForLLM(jobName, runs, includeStackTraces)
	
	jsonResponse, err := json.MarshalIndent(response, "", "  ")
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to marshal response: %v", err)), nil
	}

	return mcp.NewToolResultText(string(jsonResponse)), nil
}

// analyzeMergeFailures implements the analyze_merge_failures tool
func (s *HealthcheckMCPServer) analyzeMergeFailures(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	jobFilter := mcp.ParseString(request, "job_filter", ".*")
	testFilter := mcp.ParseString(request, "test_filter", ".*")
	includeQuarantined := mcp.ParseBoolean(request, "include_quarantined", true)

	// Fetch ci-health results
	results, err := healthcheck.FetchResults(healthcheck.HealthURL)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to fetch ci-health results: %v", err)), nil
	}

	// Build processor config
	config, err := buildProcessorConfig(jobFilter, testFilter, includeQuarantined)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to build processor config: %v", err)), nil
	}

	// Process failures
	result, err := healthcheck.ProcessFailures(results, config)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to process failures: %v", err)), nil
	}

	// Format response for LLM
	response := formatMergeFailuresForLLM(result, jobFilter, testFilter)
	
	jsonResponse, err := json.MarshalIndent(response, "", "  ")
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to marshal response: %v", err)), nil
	}

	return mcp.NewToolResultText(string(jsonResponse)), nil
}

// searchFailurePatterns implements the search_failure_patterns tool
func (s *HealthcheckMCPServer) searchFailurePatterns(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	pattern := mcp.ParseString(request, "pattern", "")
	if pattern == "" {
		return mcp.NewToolResultError("pattern parameter is required"), nil
	}

	jobFilter := mcp.ParseString(request, "job_filter", ".*")
	searchIn := mcp.ParseString(request, "search_in", "test_names")

	// Fetch ci-health results
	results, err := healthcheck.FetchResults(healthcheck.HealthURL)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to fetch ci-health results: %v", err)), nil
	}

	// Search for patterns
	matches := searchPatternsInResults(results, pattern, jobFilter, searchIn)

	// Format response for LLM
	response := formatPatternSearchForLLM(pattern, matches, searchIn)
	
	jsonResponse, err := json.MarshalIndent(response, "", "  ")
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to marshal response: %v", err)), nil
	}

	return mcp.NewToolResultText(string(jsonResponse)), nil
}

// compareTimePeriods implements the compare_time_periods tool
func (s *HealthcheckMCPServer) compareTimePeriods(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	jobName := mcp.ParseString(request, "job_name", "")
	if jobName == "" {
		return mcp.NewToolResultError("job_name parameter is required"), nil
	}

	recentPeriod := mcp.ParseString(request, "recent_period", "24h")
	comparisonPeriod := mcp.ParseString(request, "comparison_period", "7d")

	// Parse time periods
	recentDuration, err := healthcheck.ParseTimePeriod(recentPeriod)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Invalid recent period: %v", err)), nil
	}

	comparisonDuration, err := healthcheck.ParseTimePeriod(comparisonPeriod)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Invalid comparison period: %v", err)), nil
	}

	// Fetch data for both periods
	recentRuns, err := healthcheck.FetchJobHistoryWithTimePeriod(jobName, recentDuration, 1000)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to fetch recent data: %v", err)), nil
	}

	comparisonRuns, err := healthcheck.FetchJobHistoryWithTimePeriod(jobName, comparisonDuration, 1000)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to fetch comparison data: %v", err)), nil
	}

	// Analyze both periods
	recentSummary, err := healthcheck.AnalyzeLaneRuns(recentRuns)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to analyze recent data: %v", err)), nil
	}

	comparisonSummary, err := healthcheck.AnalyzeLaneRuns(comparisonRuns)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to analyze comparison data: %v", err)), nil
	}

	// Format comparison response
	response := formatTimeComparisonForLLM(jobName, recentSummary, comparisonSummary, recentPeriod, comparisonPeriod)
	
	jsonResponse, err := json.MarshalIndent(response, "", "  ")
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to marshal response: %v", err)), nil
	}

	return mcp.NewToolResultText(string(jsonResponse)), nil
}