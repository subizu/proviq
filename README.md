# agent-proof

> **Verifiable Agent Action Receipts** — signed, portable, independently verifiable evidence for AI coding agents.

`agent-proof` provides the foundational evidence layer for AI agents. It generates tamper-evident cryptographic receipts for agent actions without requiring accounts, databases, or SaaS dependencies.

```bash
# Verify a receipt independently anywhere
agent-proof verify receipt.json
→
✓ Signature valid
✓ Agent identity verified: claude-code
✓ Evidence hash matches
✓ Policy hash matches
✓ Action timestamp valid
✓ Receipt chain intact
```

---

## ⚡ Quick Start (5 Minutes)

### 1. Installation

```bash
go install github.com/subizu/proviq/cmd/agent-proof@latest
```
*Or build from source:*
```bash
go build -o bin/agent-proof ./cmd/agent-proof
```

---

### 2. Standalone CLI Usage (Phase 0)

#### Generate Keys and Record an Action:
```bash
agent-proof record \
  --agent "claude-code" \
  --intent "implement user authentication" \
  --action "git.commit" \
  --input-file src/auth.go \
  --generate-key \
  --output receipt.json
```

#### Verify the Receipt:
```bash
agent-proof verify receipt.json
```

---

### 3. Git Workflow Integration (Phase 1)

#### Auto-record Commit Provenance:
Install the post-commit hook into your repository:
```bash
agent-proof git install-hook
```

Or manually record provenance for `HEAD`:
```bash
agent-proof git record-commit --agent "cline" --private-key agent-proof.key
# Creates .agent-proof/receipts/<commit-hash>.json
```

#### GitHub Action for PR Verification:
Add to `.github/workflows/verify-agent.yml`:
```yaml
name: Verify AI Agent Provenance
on: [pull_request]

jobs:
  verify:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: ./github/actions/verify-receipt
        with:
          receipts-dir: ".agent-proof/receipts"
```

---

### 4. Agent & MCP Integration (Phase 2)

`agent-proof` includes a native **Model Context Protocol (MCP)** stdio server compatible with **Claude Code, Cline, Cursor, and Roo Code**.

#### Configure Cline / Claude Desktop (`claude_desktop_config.json`):
```json
{
  "mcpServers": {
    "agent-proof": {
      "command": "agent-proof",
      "args": ["mcp"]
    }
  }
}
```

#### Exposed MCP Tools:
- `record_agent_action`: Sign and record action metadata, file inputs, and git context automatically.
- `verify_receipt`: Check the integrity and cryptographic signature of any receipt.
- `generate_key_pair`: Create new Ed25519 key pairs for agent signing.

---

## 🏗️ Architecture & Receipt Format

Receipts use **Ed25519 signatures** over deterministic **RFC 8785 (JCS)** canonical JSON:

```json
{
  "version": "0.1",
  "agent": "claude-code",
  "intent": "implement user authentication",
  "action": "git.commit",
  "inputs": {
    "auth.go": {
      "hash": "15dfc4f51b4ed902b93822...",
      "uri": "src/auth.go"
    }
  },
  "git_context": {
    "repository": "/path/to/repo",
    "branch": "main",
    "commit": "a3f12c8e...",
    "author": "AI Agent <agent@local>",
    "timestamp": "2026-09-28T20:45:00Z"
  },
  "action_timestamp": "2026-09-28T20:45:00Z",
  "signature": "lM34eX/v2+...",
  "public_key": "xK82g1..."
}
```

---

## 📁 Project Structure

```
agent-proof/
├── cmd/
│   └── agent-proof/            # CLI entry point (main.go)
├── pkg/
│   ├── cli/                    # CLI commands (record, verify, git, mcp)
│   ├── crypto/                 # Ed25519 signing & SHA-256 helpers
│   ├── git/                    # Git repository context & hook installer
│   ├── mcp/                    # Model Context Protocol stdio server
│   └── receipt/                # Receipt schema & RFC 8785 canonical JSON
├── .github/
│   ├── actions/verify-receipt/ # Composite GitHub Action
│   └── workflows/ci.yml        # CI test & build pipeline
├── Makefile
├── go.mod
└── README.md
```

---

## 🧪 Testing

Run the full automated test suite:
```bash
go test -v ./...
```

---

## 📜 License

Apache-2.0
