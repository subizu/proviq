package cli

import (
	"os"

	"github.com/subizu/proviq/pkg/mcp"
	"github.com/spf13/cobra"
)

var McpCmd = &cobra.Command{
	Use:     "mcp",
	Aliases: []string{"serve"},
	Short:   "Start Model Context Protocol (MCP) server for Cline/Claude Code/Cursor",
	Long: `Starts a stdio MCP server exposing agent-proof tools (record_agent_action, verify_receipt, generate_key_pair)
directly to AI coding agents.

Configuration in Cline / Claude Desktop (claude_desktop_config.json):
{
  "mcpServers": {
    "agent-proof": {
      "command": "agent-proof",
      "args": ["mcp"]
    }
  }
}
`,
	RunE: func(cmd *cobra.Command, args []string) error {
		srv := mcp.NewServer(os.Stdin, os.Stdout)
		return srv.Run()
	},
}

func init() {
	RootCmd.AddCommand(McpCmd)
}
