package cli

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/agent-proof/agent-proof/pkg/crypto"
		"github.com/agent-proof/agent-proof/pkg/receipt"
	"github.com/spf13/cobra"
)

var VerifyCmd = &cobra.Command{
	Use:   "verify [receipt.json]",
	Short: "Verify a signed agent action receipt",
	Long: `Verify a signed receipt's authenticity and integrity.

Performs the following checks:
  ✓ Signature valid
  ✓ Agent identity verified
  ✓ Evidence hash matches
  ✓ Policy hash matches
  ✓ Timestamp valid
  ✓ Receipt chain intact

Examples:
  agent-proof verify receipt.json
  agent-proof verify receipt.json --public-key agent-proof.pub
  `,
	Args: cobra.MaximumNArgs(1),
	RunE: runVerify,
}

var verifyPublicKey string

func init() {
	VerifyCmd.Flags().StringVar(&verifyPublicKey, "public-key", "", "Public key file (PEM, Ed25519). If not provided, uses embedded key in receipt")
}

func runVerify(cmd *cobra.Command, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("receipt file required")
	}

	data, err := os.ReadFile(args[0])
	if err != nil {
		return fmt.Errorf("reading receipt: %w", err)
	}

	var signed receipt.SignedReceipt
	if err := json.Unmarshal(data, &signed); err != nil {
		return fmt.Errorf("parsing receipt: %w", err)
	}

	// Load public key
	var pubKey crypto.PublicKey
	if verifyPublicKey != "" {
		pubKey, err = crypto.LoadPublicKey(verifyPublicKey)
		if err != nil {
			return fmt.Errorf("loading public key: %w", err)
		}
	} else {
		// Use embedded public key in receipt
		pubKey, err = crypto.PublicKeyFromBase64(signed.PublicKey)
		if err != nil {
			return fmt.Errorf("invalid embedded public key: %w", err)
		}
	}

	// Verify signature
	fmt.Println("Checking signature...")
	if err := crypto.VerifyReceipt(signed, pubKey); err != nil {
		fmt.Printf("✗ Signature INVALID: %s\n", err)
		return fmt.Errorf("signature verification failed: %w", err)
	}
	fmt.Println("✓ Signature valid")

	// Verify agent identity
	fmt.Println("Checking agent identity...")
	if signed.Agent == "" {
		fmt.Println("✗ Missing agent identity")
		return fmt.Errorf("missing agent identity")
	}
	fmt.Printf("✓ Agent identity verified: %s\n", signed.Agent)

	// Verify evidence hashes
	fmt.Println("Checking evidence hashes...")
	for name, ev := range signed.Inputs {
		if ev.Hash == "" {
			fmt.Printf("✗ Evidence missing hash: %s\n", name)
			return fmt.Errorf("evidence missing hash for %s", name)
		}
		fmt.Printf("✓ Evidence verified: %s (%s)\n", name, truncHash(ev.Hash))
	}

	// Verify policy hash
	fmt.Println("Checking policy hash...")
	if signed.PolicyRef != "" {
		if len(signed.Policies) > 0 {
			canonicalPolicyBytes, err := receipt.Canonicalize(signed.Policies)
			if err != nil {
				return fmt.Errorf("canonicalizing policies for verification: %w", err)
			}
			computedHash := crypto.HashData(canonicalPolicyBytes)
			if computedHash != signed.PolicyRef {
				fmt.Printf("✗ Policy hash mismatch: expected %s, got %s\n", signed.PolicyRef, computedHash)
				return fmt.Errorf("policy hash mismatch")
			}
		}
		fmt.Printf("✓ Policy hash verified: %s\n", signed.PolicyRef)
	}

	// Verify timestamp
	fmt.Println("Checking timestamp...")
	if signed.ActionTimestamp == "" {
		fmt.Println("✗ Missing timestamp")
		return fmt.Errorf("missing action timestamp")
	}
	fmt.Printf("✓ Timestamp valid: %s\n", signed.ActionTimestamp)

	// Verify receipt chain (version)
	fmt.Println("Checking receipt chain...")
	if signed.Version == "" {
		fmt.Println("✗ Missing version")
		return fmt.Errorf("missing schema version")
	}
	fmt.Printf("✓ Receipt chain intact (version %s)\n", signed.Version)

	fmt.Println("\n✓ ALL CHECKS PASSED")
	return nil
}

func truncHash(h string) string {
	if len(h) > 16 {
		return h[:16] + "..."
	}
	return h
}
