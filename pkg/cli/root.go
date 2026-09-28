package cli

import (
	"os"

	"github.com/spf13/cobra"
)

var RootCmd = &cobra.Command{
	Use:   "agent-proof",
	Short: "Verifiable agent action receipts — signed, portable, independently verifiable",
	Long: `agent-proof generates and verifies tamper-evident receipts for AI agent actions.

No account • No SaaS • No database
Just a portable, signed evidence artifact.

  agent-proof record ... > receipt.json
  agent-proof verify receipt.json`,
}

func Execute() error {
	return RootCmd.Execute()
}

func init() {
	RootCmd.SetOut(os.Stdout)
	RootCmd.SetErr(os.Stderr)
	RootCmd.AddCommand(RecordCmd)
	RootCmd.AddCommand(VerifyCmd)
	RootCmd.AddCommand(GitCmd)
}
