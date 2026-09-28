package cli

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/agent-proof/agent-proof/pkg/crypto"
	"github.com/agent-proof/agent-proof/pkg/git"
	"github.com/agent-proof/agent-proof/pkg/receipt"
	"github.com/spf13/cobra"
)

var GitCmd = &cobra.Command{
	Use:   "git",
	Short: "Git workflow integration for agent-proof",
	Long:  `Manage git hooks and automate commit provenance recording.`,
}

var hookInstallCmd = &cobra.Command{
	Use:   "install-hook",
	Short: "Install git post-commit hook for automatic provenance capture",
	RunE: func(cmd *cobra.Command, args []string) error {
		path, err := git.InstallHook(".")
		if err != nil {
			return err
		}
		fmt.Printf("✓ Installed git post-commit hook at: %s\n", path)
		return nil
	},
}

var (
	gitRecordAgent   string
	gitRecordKeyFile string
	gitRecordOutput  string
)

var gitRecordCmd = &cobra.Command{
	Use:   "record-commit",
	Short: "Record provenance for the latest Git commit (HEAD)",
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx, err := git.CaptureContext(".")
		if err != nil || ctx.Repository == "" {
			return fmt.Errorf("not in a git repository")
		}
		if ctx.Commit == "" {
			return fmt.Errorf("no commits found in repository")
		}

		if gitRecordAgent == "" {
			gitRecordAgent = os.Getenv("AGENT_NAME")
			if gitRecordAgent == "" {
				gitRecordAgent = "coding-agent"
			}
		}

		if gitRecordKeyFile == "" {
			gitRecordKeyFile = os.Getenv("AGENT_PROOF_KEY")
			if gitRecordKeyFile == "" {
				gitRecordKeyFile = "agent-proof.key"
			}
		}

		privKey, err := crypto.LoadPrivateKey(gitRecordKeyFile)
		if err != nil {
			return fmt.Errorf("loading private key (%s): %w", gitRecordKeyFile, err)
		}

		diff, _ := git.GetCommitDiff(".", ctx.Commit)
		inputs := make(map[string]receipt.Evidence)
		if diff != "" {
			inputs["git.commit_diff"] = receipt.Evidence{
				Hash: receipt.HashBytes([]byte(diff)),
			}
		}

		r := receipt.Receipt{
			Version: "0.1",
			Agent:   gitRecordAgent,
			Intent:  ctx.Message,
			Action:  "git.commit",
			Inputs:  inputs,
			GitContext: &receipt.GitContext{
				Repository: ctx.Repository,
				Branch:     ctx.Branch,
				Commit:     ctx.Commit,
				Author:     ctx.Author,
				Committer:  ctx.Committer,
				Timestamp:  ctx.Timestamp,
			},
			ActionTimestamp: ctx.Timestamp,
		}

		signed, err := crypto.SignReceipt(r, privKey)
		if err != nil {
			return fmt.Errorf("signing receipt: %w", err)
		}

		outDir := git.DefaultReceiptsDir(ctx.Repository)
		if err := os.MkdirAll(outDir, 0755); err != nil {
			return fmt.Errorf("creating receipts directory: %w", err)
		}

		outFile := gitRecordOutput
		if outFile == "" {
			outFile = filepath.Join(outDir, fmt.Sprintf("%s.json", ctx.Commit))
		}

		canonicalJSON, err := receipt.Canonicalize(signed)
		if err != nil {
			return fmt.Errorf("serializing receipt: %w", err)
		}

		if err := os.WriteFile(outFile, canonicalJSON, 0644); err != nil {
			return fmt.Errorf("writing receipt file: %w", err)
		}

		fmt.Printf("✓ Recorded commit provenance receipt at %s\n", outFile)
		return nil
	},
}

func init() {
	gitRecordCmd.Flags().StringVar(&gitRecordAgent, "agent", "", "Agent name (or AGENT_NAME env)")
	gitRecordCmd.Flags().StringVar(&gitRecordKeyFile, "private-key", "", "Private key file (or AGENT_PROOF_KEY env)")
	gitRecordCmd.Flags().StringVar(&gitRecordOutput, "output", "", "Output file path (default: .agent-proof/receipts/<commit>.json)")

	GitCmd.AddCommand(hookInstallCmd)
	GitCmd.AddCommand(gitRecordCmd)
}
