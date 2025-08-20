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

	// Tool 6: Get failure source context
	getFailureSourceContextTool := mcp.NewTool(
		"get_failure_source_context",
		mcp.WithDescription("Parse junit failure output and generate GitHub URLs for source code context"),
		mcp.WithString("failure_text", mcp.Description("JUnit failure text containing file paths and line numbers"), mcp.Required()),
		mcp.WithString("job_url", mcp.Description("Job URL to extract repository and commit information"), mcp.Required()),
		mcp.WithBoolean("include_stack_trace", mcp.Description("Include parsed stack trace information"), mcp.DefaultBool(true)),
	)
	mcpServer.AddTool(getFailureSourceContextTool, s.getFailureSourceContext)

	// Tool 7: Analyze failure trends over time
	analyzeFailureTrendsTool := mcp.NewTool(
		"analyze_failure_trends",
		mcp.WithDescription("Analyze failure trends and patterns over time periods"),
		mcp.WithString("job_name", mcp.Description("Name of the CI job to analyze"), mcp.Required()),
		mcp.WithString("trend_period", mcp.Description("Time period for trend analysis (e.g., '7d', '14d', '30d')"), mcp.DefaultString("14d")),
		mcp.WithBoolean("include_flakiness", mcp.Description("Include flakiness analysis"), mcp.DefaultBool(true)),
	)
	mcpServer.AddTool(analyzeFailureTrendsTool, s.analyzeFailureTrends)

	// Tool 8: Cross-job failure correlation
	analyzeFailureCorrelationTool := mcp.NewTool(
		"analyze_failure_correlation",
		mcp.WithDescription("Analyze failures across multiple jobs to identify systemic issues"),
		mcp.WithString("job_pattern", mcp.Description("Job pattern or alias to analyze (e.g., 'compute', 'storage')"), mcp.DefaultString(".*")),
		mcp.WithString("time_window", mcp.Description("Time window for correlation analysis"), mcp.DefaultString("24h")),
		mcp.WithBoolean("include_environment_analysis", mcp.Description("Include environment-specific failure analysis"), mcp.DefaultBool(true)),
	)
	mcpServer.AddTool(analyzeFailureCorrelationTool, s.analyzeFailureCorrelation)

	// Tool 9: Quarantine intelligence
	analyzeQuarantineIntelligenceTool := mcp.NewTool(
		"analyze_quarantine_intelligence",
		mcp.WithDescription("Provide intelligent analysis of quarantined tests and recommendations"),
		mcp.WithString("scope", mcp.Description("Analysis scope: 'all', 'job', or specific job name"), mcp.DefaultString("all")),
		mcp.WithBoolean("include_recommendations", mcp.Description("Include quarantine action recommendations"), mcp.DefaultBool(true)),
	)
	mcpServer.AddTool(analyzeQuarantineIntelligenceTool, s.analyzeQuarantineIntelligence)

	// Tool 10: Failure impact assessment
	assessFailureImpactTool := mcp.NewTool(
		"assess_failure_impact",
		mcp.WithDescription("Assess the impact and priority of test failures for triage"),
		mcp.WithString("failure_data", mcp.Description("JSON failure data from lane or merge commands"), mcp.Required()),
		mcp.WithString("context", mcp.Description("Context: 'pre-release', 'development', 'production'"), mcp.DefaultString("development")),
		mcp.WithBoolean("include_triage_recommendations", mcp.Description("Include triage priority recommendations"), mcp.DefaultBool(true)),
	)
	mcpServer.AddTool(assessFailureImpactTool, s.assessFailureImpact)

	// Tool 11: Generate failure report
	generateFailureReportTool := mcp.NewTool(
		"generate_failure_report",
		mcp.WithDescription("Generate comprehensive failure analysis report for stakeholders"),
		mcp.WithString("scope", mcp.Description("Report scope: 'daily', 'weekly', 'release', or specific job"), mcp.DefaultString("daily")),
		mcp.WithString("format", mcp.Description("Report format: 'summary', 'detailed', 'executive'"), mcp.DefaultString("summary")),
		mcp.WithBoolean("include_recommendations", mcp.Description("Include actionable recommendations"), mcp.DefaultBool(true)),
	)
	mcpServer.AddTool(generateFailureReportTool, s.generateFailureReport)
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

// getFailureSourceContext implements the get_failure_source_context tool
func (s *HealthcheckMCPServer) getFailureSourceContext(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	failureText := mcp.ParseString(request, "failure_text", "")
	if failureText == "" {
		return mcp.NewToolResultError("failure_text parameter is required"), nil
	}

	jobURL := mcp.ParseString(request, "job_url", "")
	if jobURL == "" {
		return mcp.NewToolResultError("job_url parameter is required"), nil
	}

	includeStackTrace := mcp.ParseBoolean(request, "include_stack_trace", true)

	// Parse failure information
	failureInfo, err := ParseFailureText(failureText)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to parse failure text: %v", err)), nil
	}

	// Extract repository and commit information from job URL
	repoInfo, err := ExtractRepositoryInfo(jobURL)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to extract repository info: %v", err)), nil
	}

	// Generate GitHub URLs for source context
	response := FormatFailureSourceContextForLLM(failureInfo, repoInfo, includeStackTrace)

	jsonResponse, err := json.MarshalIndent(response, "", "  ")
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to marshal response: %v", err)), nil
	}

	return mcp.NewToolResultText(string(jsonResponse)), nil
}

// analyzeFailureTrends implements the analyze_failure_trends tool
func (s *HealthcheckMCPServer) analyzeFailureTrends(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	jobName := mcp.ParseString(request, "job_name", "")
	if jobName == "" {
		return mcp.NewToolResultError("job_name parameter is required"), nil
	}

	trendPeriod := mcp.ParseString(request, "trend_period", "14d")
	includeFlakiness := mcp.ParseBoolean(request, "include_flakiness", true)

	// Parse trend period
	trendDuration, err := healthcheck.ParseTimePeriod(trendPeriod)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Invalid trend period: %v", err)), nil
	}

	// Fetch historical data for trend analysis
	runs, err := healthcheck.FetchJobHistoryWithTimePeriod(jobName, trendDuration, 500) // Larger limit for trend analysis
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to fetch job history: %v", err)), nil
	}

	// Analyze trends
	trendAnalysis := analyzeTrendsFromRuns(runs, includeFlakiness)
	trendAnalysis.JobName = jobName
	trendAnalysis.TrendPeriod = trendPeriod

	jsonResponse, err := json.MarshalIndent(trendAnalysis, "", "  ")
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to marshal response: %v", err)), nil
	}

	return mcp.NewToolResultText(string(jsonResponse)), nil
}

// analyzeFailureCorrelation implements the analyze_failure_correlation tool
func (s *HealthcheckMCPServer) analyzeFailureCorrelation(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	jobPattern := mcp.ParseString(request, "job_pattern", ".*")
	timeWindow := mcp.ParseString(request, "time_window", "24h")
	includeEnvironmentAnalysis := mcp.ParseBoolean(request, "include_environment_analysis", true)

	// Fetch ci-health results for cross-job analysis
	results, err := healthcheck.FetchResults(healthcheck.HealthURL)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to fetch ci-health results: %v", err)), nil
	}

	// Analyze correlation across jobs
	correlationAnalysis := analyzeFailureCorrelationAcrossJobs(results, jobPattern, timeWindow, includeEnvironmentAnalysis)

	jsonResponse, err := json.MarshalIndent(correlationAnalysis, "", "  ")
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to marshal response: %v", err)), nil
	}

	return mcp.NewToolResultText(string(jsonResponse)), nil
}

// analyzeQuarantineIntelligence implements the analyze_quarantine_intelligence tool
func (s *HealthcheckMCPServer) analyzeQuarantineIntelligence(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	scope := mcp.ParseString(request, "scope", "all")
	includeRecommendations := mcp.ParseBoolean(request, "include_recommendations", true)

	// Fetch quarantined tests
	quarantinedTests, err := healthcheck.FetchQuarantinedTests()
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to fetch quarantined tests: %v", err)), nil
	}

	// Fetch current ci-health data for analysis
	results, err := healthcheck.FetchResults(healthcheck.HealthURL)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to fetch ci-health results: %v", err)), nil
	}

	// Analyze quarantine intelligence
	quarantineAnalysis := analyzeQuarantineEffectiveness(quarantinedTests, results, scope, includeRecommendations)

	jsonResponse, err := json.MarshalIndent(quarantineAnalysis, "", "  ")
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to marshal response: %v", err)), nil
	}

	return mcp.NewToolResultText(string(jsonResponse)), nil
}

// assessFailureImpact implements the assess_failure_impact tool
func (s *HealthcheckMCPServer) assessFailureImpact(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	failureData := mcp.ParseString(request, "failure_data", "")
	if failureData == "" {
		return mcp.NewToolResultError("failure_data parameter is required"), nil
	}

	context := mcp.ParseString(request, "context", "development")
	includeTriageRecommendations := mcp.ParseBoolean(request, "include_triage_recommendations", true)

	// Parse the JSON failure data
	impactAnalysis, err := assessFailureImpactFromJSON(failureData, context, includeTriageRecommendations)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to assess failure impact: %v", err)), nil
	}

	jsonResponse, err := json.MarshalIndent(impactAnalysis, "", "  ")
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to marshal response: %v", err)), nil
	}

	return mcp.NewToolResultText(string(jsonResponse)), nil
}

// generateFailureReport implements the generate_failure_report tool
func (s *HealthcheckMCPServer) generateFailureReport(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	scope := mcp.ParseString(request, "scope", "daily")
	format := mcp.ParseString(request, "format", "summary")
	includeRecommendations := mcp.ParseBoolean(request, "include_recommendations", true)

	// Generate comprehensive failure report
	report, err := generateComprehensiveFailureReport(scope, format, includeRecommendations)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to generate failure report: %v", err)), nil
	}

	jsonResponse, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to marshal response: %v", err)), nil
	}

	return mcp.NewToolResultText(string(jsonResponse)), nil
}