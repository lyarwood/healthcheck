package cmd

import (
	"fmt"
	"os"

	"healthcheck/pkg/mcp"

	"github.com/spf13/cobra"
)

var (
	mcpPort   int
	mcpHost   string
	mcpStdio  bool
	mcpDebug  bool
)

var mcpCmd = &cobra.Command{
	Use:   "mcp",
	Short: "Start MCP server for LLM-assisted CI health analysis",
	Long: `Start a Model Context Protocol (MCP) server that exposes healthcheck functionality 
to Large Language Models for intelligent CI failure analysis.

The MCP server provides tools for:
- Analyzing job lane failures with pattern recognition
- Getting detailed failure information with stack traces  
- Searching for failure patterns across jobs
- Comparing failure rates between time periods
- Cross-job analysis using ci-health data

This enables LLM-powered workflows like:
- "Analyze recent failures in pull-kubevirt-e2e-k8s-1.32-sig-compute"
- "Compare this week's failure rate to last week for unit tests"
- "Find all migration-related failures across all jobs"
- "Generate a release health report for all SIG areas"`,
	RunE: func(_ *cobra.Command, _ []string) error {
		// Create and configure MCP server
		server := mcp.NewHealthcheckMCPServer()
		
		if mcpDebug {
			fmt.Fprintf(os.Stderr, "Starting healthcheck MCP server...\n")
			fmt.Fprintf(os.Stderr, "Available tools:\n")
			fmt.Fprintf(os.Stderr, "- analyze_job_lane: Analyze job failures with patterns\n")
			fmt.Fprintf(os.Stderr, "- get_job_failures: Get detailed failure information\n")
			fmt.Fprintf(os.Stderr, "- analyze_merge_failures: Cross-job failure analysis\n")
			fmt.Fprintf(os.Stderr, "- search_failure_patterns: Find patterns across jobs\n")
			fmt.Fprintf(os.Stderr, "- compare_time_periods: Compare failure rates over time\n")
			fmt.Fprintf(os.Stderr, "\n")
		}
		
		// Start the server
		if err := server.Serve(); err != nil {
			return fmt.Errorf("MCP server failed: %w", err)
		}
		
		return nil
	},
}

func init() {
	mcpCmd.Flags().IntVarP(&mcpPort, "port", "p", 0, "Port to listen on (0 for stdio)")
	mcpCmd.Flags().StringVarP(&mcpHost, "host", "H", "localhost", "Host to bind to")
	mcpCmd.Flags().BoolVarP(&mcpStdio, "stdio", "s", true, "Use stdio transport (default)")
	mcpCmd.Flags().BoolVarP(&mcpDebug, "debug", "d", false, "Enable debug logging")

	rootCmd.AddCommand(mcpCmd)
}