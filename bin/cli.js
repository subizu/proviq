#!/usr/bin/env node

const { spawn } = require('child_process');
const path = require('path');
const fs = require('fs');

// Check if native binary exists locally
const isWindows = process.platform === 'win32';
const binName = isWindows ? 'agent-proof.exe' : 'agent-proof';
const localBin = path.join(__dirname, '..', 'bin', binName);

if (fs.existsSync(localBin)) {
  const child = spawn(localBin, process.argv.slice(2), { stdio: 'inherit' });
  child.on('exit', (code) => process.exit(code || 0));
} else {
  // Try invoking agent-proof from system PATH
  const child = spawn(binName, process.argv.slice(2), { stdio: 'inherit' });
  child.on('error', () => {
    console.error(`\x1b[31m[agent-proof]\x1b[0m Native binary '${binName}' not found in PATH.`);
    console.error(`Please install via Go:\n  go install github.com/subizu/proviq/cmd/agent-proof@latest`);
    console.error(`Or via installer:\n  curl -fsSL https://raw.githubusercontent.com/subizu/proviq/main/install.sh | bash\n`);
    process.exit(1);
  });
  child.on('exit', (code) => process.exit(code || 0));
}
