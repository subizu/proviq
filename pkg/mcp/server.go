package mcp

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/agent-proof/agent-proof/pkg/crypto"
	"github.com/agent-proof/agent-proof/pkg/git"
	"github.com/agent-proof/agent-proof/pkg/receipt"
)

// JSONRPCRequest represents a JSON-RPC 2.0 request or notification.
type JSONRPCRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      any             `json:"id,omitempty"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

// JSONRPCResponse represents a JSON-RPC 2.0 response.
type JSONRPCResponse struct {
	JSONRPC string        `json:"jsonrpc"`
	ID      any           `json:"id"`
	Result  any           `json:"result,omitempty"`
	Error   *JSONRPCError `json:"error,omitempty"`
}

type JSONRPCError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

type Tool struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	InputSchema map[string]any `json:"inputSchema"`
}

type TextContent struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

type ToolCallResult struct {
	Content []TextContent `json:"content"`
	IsError bool          `json:"isError,omitempty"`
}

// Server runs an MCP stdio server.
type Server struct {
	reader *bufio.Reader
	writer io.Writer
}

func NewServer(r io.Reader, w io.Writer) *Server {
	return &Server{
		reader: bufio.NewReader(r),
		writer: w,
	}
}

func (s *Server) Run() error {
	for {
		line, err := s.reader.ReadBytes('\n')
		if err != nil {
			if err == io.EOF {
				return nil
			}
			return err
		}

		if len(line) == 0 || line[0] == '\n' || line[0] == '\r' {
			continue
		}

		var req JSONRPCRequest
		if err := json.Unmarshal(line, &req); err != nil {
			continue
		}

		resp := s.handleRequest(req)
		if resp != nil {
			respBytes, err := json.Marshal(resp)
			if err == nil {
				respBytes = append(respBytes, '\n')
				_, _ = s.writer.Write(respBytes)
			}
		}
	}
}

func (s *Server) handleRequest(req JSONRPCRequest) *JSONRPCResponse {
	// Notifications don't need a response
	if req.ID == nil {
		return nil
	}

	switch req.Method {
	case "initialize":
		return &JSONRPCResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Result: map[string]any{
				"protocolVersion": "2024-11-05",
				"capabilities": map[string]any{
					"tools": map[string]any{},
				},
				"serverInfo": map[string]any{
					"name":    "agent-proof",
					"version": "0.1.0",
				},
			},
		}

	case "tools/list":
		return &JSONRPCResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Result: map[string]any{
				"tools": getAvailableTools(),
			},
		}

	case "tools/call":
		var callParams struct {
			Name      string                 `json:"name"`
			Arguments map[string]interface{} `json:"arguments"`
		}
		if err := json.Unmarshal(req.Params, &callParams); err != nil {
			return &JSONRPCResponse{
				JSONRPC: "2.0",
				ID:      req.ID,
				Error:   &JSONRPCError{Code: -32602, Message: "Invalid params"},
			}
		}

		res := handleToolCall(callParams.Name, callParams.Arguments)
		return &JSONRPCResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Result:  res,
		}

	default:
		return &JSONRPCResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Error:   &JSONRPCError{Code: -32601, Message: fmt.Sprintf("Method %s not found", req.Method)},
		}
	}
}

func getAvailableTools() []Tool {
	return []Tool{
		{
			Name:        "record_agent_action",
			Description: "Creates a signed, tamper-evident cryptographic receipt for an agent action",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"agent": map[string]any{
						"type":        "string",
						"description": "Identifier of the agent (e.g., 'claude-code', 'cline', 'cursor')",
					},
					"intent": map[string]any{
						"type":        "string",
						"description": "The goal or human intent behind this action",
					},
					"action": map[string]any{
						"type":        "string",
						"description": "Action type (e.g. 'git.commit', 'file.edit', 'db.migration')",
					},
					"input_files": map[string]any{
						"type":        "array",
						"items":       map[string]any{"type": "string"},
						"description": "List of input/modified file paths to hash as evidence",
					},
					"output_file": map[string]any{
						"type":        "string",
						"description": "Optional path where receipt JSON will be saved",
					},
					"private_key_path": map[string]any{
						"type":        "string",
						"description": "Optional private key path. If not found, a key pair is generated automatically.",
					},
				},
				"required": []string{"agent", "intent", "action"},
			},
		},
		{
			Name:        "verify_receipt",
			Description: "Verifies the cryptographic signature, hashes, and integrity of a receipt",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"receipt_path": map[string]any{
						"type":        "string",
						"description": "Path to the receipt JSON file to verify",
					},
				},
				"required": []string{"receipt_path"},
			},
		},
		{
			Name:        "generate_key_pair",
			Description: "Generates a new Ed25519 key pair for agent signing",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"key_name": map[string]any{
						"type":        "string",
						"description": "Prefix name for the generated key files (default: 'agent-proof')",
					},
				},
			},
		},
	}
}

func handleToolCall(name string, args map[string]interface{}) ToolCallResult {
	switch name {
	case "record_agent_action":
		agent, _ := args["agent"].(string)
		intent, _ := args["intent"].(string)
		action, _ := args["action"].(string)
		outputFile, _ := args["output_file"].(string)
		keyPath, _ := args["private_key_path"].(string)

		if keyPath == "" {
			keyPath = "agent-proof.key"
		}

		var privKey crypto.PrivateKey
		var pubKey crypto.PublicKey
		var err error

		if _, statErr := os.Stat(keyPath); os.IsNotExist(statErr) {
			pubKey, privKey, err = crypto.GenerateEd25519Key()
			if err != nil {
				return errorResult(fmt.Sprintf("failed to generate key: %v", err))
			}
			_ = crypto.WritePrivateKey(privKey, keyPath)
			_ = crypto.WritePublicKey(pubKey, "agent-proof.pub")
		} else {
			privKey, err = crypto.LoadPrivateKey(keyPath)
			if err != nil {
				return errorResult(fmt.Sprintf("failed to load private key %s: %v", keyPath, err))
			}
		}

		inputs := make(map[string]receipt.Evidence)
		if inputFiles, ok := args["input_files"].([]any); ok {
			for _, item := range inputFiles {
				if fPath, ok := item.(string); ok {
					ev, err := receipt.HashFile(fPath)
					if err == nil {
						inputs[filepath.Base(fPath)] = ev
					}
				}
			}
		}

		gitCtx, _ := git.CaptureContext(".")

		r := receipt.Receipt{
			Version:         "0.1",
			Agent:           agent,
			Intent:          intent,
			Action:          action,
			Inputs:          inputs,
			GitContext:      &receipt.GitContext{
				Repository: gitCtx.Repository,
				Branch:     gitCtx.Branch,
				Commit:     gitCtx.Commit,
				Author:     gitCtx.Author,
				Committer:  gitCtx.Committer,
				Timestamp:  gitCtx.Timestamp,
			},
			ActionTimestamp: time.Now().UTC().Format(time.RFC3339),
		}

		signed, err := crypto.SignReceipt(r, privKey)
		if err != nil {
			return errorResult(fmt.Sprintf("failed to sign receipt: %v", err))
		}

		receiptJSON, err := json.MarshalIndent(signed, "", "  ")
		if err != nil {
			return errorResult(fmt.Sprintf("failed to serialize receipt: %v", err))
		}

		if outputFile != "" {
			_ = os.WriteFile(outputFile, receiptJSON, 0644)
		}

		return ToolCallResult{
			Content: []TextContent{
				{
					Type: "text",
					Text: fmt.Sprintf("✓ Action receipt recorded successfully!\n\n%s", string(receiptJSON)),
				},
			},
		}

	case "verify_receipt":
		receiptPath, _ := args["receipt_path"].(string)
		if receiptPath == "" {
			return errorResult("receipt_path is required")
		}

		data, err := os.ReadFile(receiptPath)
		if err != nil {
			return errorResult(fmt.Sprintf("reading receipt file: %v", err))
		}

		var signed receipt.SignedReceipt
		if err := json.Unmarshal(data, &signed); err != nil {
			return errorResult(fmt.Sprintf("parsing receipt JSON: %v", err))
		}

		pubKey, err := crypto.PublicKeyFromBase64(signed.PublicKey)
		if err != nil {
			return errorResult(fmt.Sprintf("invalid embedded public key: %v", err))
		}

		if err := crypto.VerifyReceipt(signed, pubKey); err != nil {
			return errorResult(fmt.Sprintf("✗ Signature verification FAILED: %v", err))
		}

		return ToolCallResult{
			Content: []TextContent{
				{
					Type: "text",
					Text: fmt.Sprintf("✓ Receipt VERIFIED: Valid signature by agent '%s' at %s (action: %s)",
						signed.Agent, signed.ActionTimestamp, signed.Action),
				},
			},
		}

	case "generate_key_pair":
		keyName, _ := args["key_name"].(string)
		if keyName == "" {
			keyName = "agent-proof"
		}

		pub, priv, err := crypto.GenerateEd25519Key()
		if err != nil {
			return errorResult(fmt.Sprintf("key generation failed: %v", err))
		}

		privPath := fmt.Sprintf("%s.key", keyName)
		pubPath := fmt.Sprintf("%s.pub", keyName)

		if err := crypto.WritePrivateKey(priv, privPath); err != nil {
			return errorResult(fmt.Sprintf("writing private key: %v", err))
		}
		if err := crypto.WritePublicKey(pub, pubPath); err != nil {
			return errorResult(fmt.Sprintf("writing public key: %v", err))
		}

		return ToolCallResult{
			Content: []TextContent{
				{
					Type: "text",
					Text: fmt.Sprintf("✓ Generated Ed25519 key pair:\n- Private key: %s\n- Public key: %s", privPath, pubPath),
				},
			},
		}

	default:
		return errorResult(fmt.Sprintf("Unknown tool: %s", name))
	}
}

func errorResult(msg string) ToolCallResult {
	return ToolCallResult{
		Content: []TextContent{
			{
				Type: "text",
				Text: msg,
			},
		},
		IsError: true,
	}
}
