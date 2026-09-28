package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/agent-proof/agent-proof/pkg/crypto"
	"github.com/agent-proof/agent-proof/pkg/git"
	"github.com/agent-proof/agent-proof/pkg/receipt"
	"github.com/spf13/cobra"
)

var (
	recordAgent     string
	recordIntent    string
	recordAction    string
	recordInputs    []string
	recordInputDir  string
	recordOutput    string
	recordPolicy    string
	recordPolicyDir string
	recordKeyFile   string
	recordGenerate  bool
)

var RecordCmd = &cobra.Command{
	Use:   "record",
	Short: "Generate a signed agent action receipt",
	Long: `Create a tamper-evident, signed receipt for an agent action.

Examples:
  agent-proof record \
    --agent "claude-code" \
    --intent "modify-authentication-module" \
    --action "git.commit" \
    --input-file diff.patch \
    --policy-file policy.rego \
    --private-key key.pem
  `,
	RunE: runRecord,
}

func init() {
	RecordCmd.Flags().StringVar(&recordAgent, "agent", "", "Agent identifier (e.g. 'claude-code')")
	RecordCmd.Flags().StringVar(&recordIntent, "intent", "", "Human-readable intent for the action")
	RecordCmd.Flags().StringVar(&recordAction, "action", "", "Action type (e.g. 'git.commit', 'file.edit')")
	RecordCmd.Flags().StringSliceVar(&recordInputs, "input-file", nil, "Input files to include as evidence")
	RecordCmd.Flags().StringVar(&recordInputDir, "input-dir", "", "Directory of input files to include")
	RecordCmd.Flags().StringVar(&recordOutput, "output", "", "Output file (default: stdout)")
	RecordCmd.Flags().StringVar(&recordPolicy, "policy-file", "", "Policy file to reference")
	RecordCmd.Flags().StringVar(&recordPolicyDir, "policy-dir", "", "Directory of files to use as policy context")
	RecordCmd.Flags().StringVar(&recordKeyFile, "private-key", "", "Private key file for signing (PEM, Ed25519)")
	RecordCmd.Flags().BoolVar(&recordGenerate, "generate-key", false, "Generate a new Ed25519 key pair")

	RecordCmd.MarkFlagRequired("agent")
	RecordCmd.MarkFlagRequired("intent")
	RecordCmd.MarkFlagRequired("action")
}

func runRecord(cmd *cobra.Command, _ []string) error {
	// Generate key pair if requested
	if recordGenerate {
		pub, priv, err := crypto.GenerateEd25519Key()
		if err != nil {
			return fmt.Errorf("failed to generate key: %w", err)
		}
		privFile := "agent-proof.key"
		pubFile := "agent-proof.pub"
		if err := crypto.WritePrivateKey(priv, privFile); err != nil {
			return fmt.Errorf("failed to write private key: %w", err)
		}
		if err := crypto.WritePublicKey(pub, pubFile); err != nil {
			return fmt.Errorf("failed to write public key: %w", err)
		}
		fmt.Fprintf(os.Stderr, "Generated key pair: %s, %s\n", privFile, pubFile)
		recordKeyFile = privFile
	}

	// Load private key
	if recordKeyFile == "" {
		return fmt.Errorf("--private-key is required (or use --generate-key)")
	}

	privKey, err := crypto.LoadPrivateKey(recordKeyFile)
	if err != nil {
		return fmt.Errorf("failed to load private key: %w", err)
	}

	// Collect inputs
	inputs, err := collectInputs()
	if err != nil {
		return fmt.Errorf("failed to collect inputs: %w", err)
	}

	// Collect policy
	policies, policyHash, err := collectPolicy()
	if err != nil {
		return fmt.Errorf("failed to collect policy: %w", err)
	}

	// Auto-capture git context
	gitCtx, _ := git.CaptureContext(".")

	// Build receipt
	r := receipt.Receipt{
		Version:   "0.1",
		Agent:     recordAgent,
		Intent:    recordIntent,
		Action:    recordAction,
		Inputs:    inputs,
		Policies:  policies,
		PolicyRef: policyHash,
		GitContext: &receipt.GitContext{
			Repository: gitCtx.Repository,
			Branch:     gitCtx.Branch,
			Commit:     gitCtx.Commit,
			Author:     gitCtx.Author,
			Committer:  gitCtx.Committer,
			Timestamp:  gitCtx.Timestamp,
		},
		ActionTimestamp: time.Now().UTC().Format(time.RFC3339),
	}

	// Sign and serialize
	signed, err := crypto.SignReceipt(r, privKey)
	if err != nil {
		return fmt.Errorf("failed to sign receipt: %w", err)
	}

	output, err := json.MarshalIndent(signed, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to serialize receipt: %w", err)
	}

	if recordOutput != "" {
		if err := os.WriteFile(recordOutput, output, 0644); err != nil {
			return fmt.Errorf("failed to write output: %w", err)
		}
		fmt.Fprintf(os.Stderr, "Receipt written to %s\n", recordOutput)
	} else {
		fmt.Println(string(output))
	}

	return nil
}

func collectInputs() (map[string]receipt.Evidence, error) {
	inputs := make(map[string]receipt.Evidence)

	// Individual files
	for _, f := range recordInputs {
		ev, err := receipt.HashFile(f)
		if err != nil {
			return nil, err
		}
		inputs[filepath.Base(f)] = ev
	}

	// Directory of files
	if recordInputDir != "" {
		entries, err := os.ReadDir(recordInputDir)
		if err != nil {
			return nil, fmt.Errorf("reading input dir: %w", err)
		}
		for _, entry := range entries {
			if entry.IsDir() {
				continue
			}
			path := filepath.Join(recordInputDir, entry.Name())
			ev, err := receipt.HashFile(path)
			if err != nil {
				continue
			}
			inputs[path] = ev
		}
	}

	return inputs, nil
}

func collectPolicy() (map[string]string, string, error) {
	policies := make(map[string]string)

	if recordPolicy != "" {
		ev, err := receipt.HashFile(recordPolicy)
		if err != nil {
			return nil, "", err
		}
		policies[filepath.Base(recordPolicy)] = ev.Hash
	}

	if recordPolicyDir != "" {
		entries, err := os.ReadDir(recordPolicyDir)
		if err != nil {
			return nil, "", fmt.Errorf("reading policy dir: %w", err)
		}
		for _, entry := range entries {
			if entry.IsDir() {
				continue
			}
			path := filepath.Join(recordPolicyDir, entry.Name())
			ev, err := receipt.HashFile(path)
			if err != nil {
				continue
			}
			policies[entry.Name()] = ev.Hash
		}
	}

	if len(policies) == 0 {
		return policies, "", nil
	}

	canonicalPolicyBytes, err := receipt.Canonicalize(policies)
	if err != nil {
		return nil, "", fmt.Errorf("canonicalizing policy: %w", err)
	}

	policyHash := crypto.HashData(canonicalPolicyBytes)
	return policies, policyHash, nil
}
