# Code Sandbox MCP 🐳
[![smithery badge](https://smithery.ai/badge/@Automata-Labs-team/code-sandbox-mcp)](https://smithery.ai/server/@Automata-Labs-team/code-sandbox-mcp)

A secure sandbox environment for executing code within Docker containers. This MCP server provides AI applications with a safe and isolated environment for running code while maintaining security through containerization.
![Screenshot from 2025-01-26 02-37-42](https://github.com/user-attachments/assets/c3fcf202-24a2-488a-818f-ffab6f881849)
## 🌟 Features

- **Multi-Language Support**: Run Python, Go, and Node.js code in isolated Docker containers
- **TypeScript Support**: Built-in support for TypeScript and JSX/TSX files
- **Dependency Management**: Automatic handling of project dependencies (pip, go mod, npm)
- **Flexible Execution**: Custom entrypoints for both single-file code and full projects
- **Background Mode**: Run long-running services in the background
- **Real-time Output**: Capture and stream container logs in real-time

## 🚀 Installation

### Prerequisites

- Docker installed and running
  - [Install Docker for Linux](https://docs.docker.com/engine/install/)
  - [Install Docker Desktop for macOS](https://docs.docker.com/desktop/install/mac/)
  - [Install Docker Desktop for Windows](https://docs.docker.com/desktop/install/windows-install/)

### Quick Install

#### Linux, MacOS
```bash
curl -fsSL https://raw.githubusercontent.com/Automata-Labs-team/code-sandbox-mcp/main/install.sh | bash
```

Example output:
```
Downloading latest release...
Installing to /home/user/.local/share/code-sandbox-mcp/code-sandbox-mcp...
Adding to Claude Desktop configuration...
Added code-sandbox-mcp to /home/user/.config/Claude/claude_desktop_config.json
Installation complete!
You can now use code-sandbox-mcp with Claude Desktop or other AI applications.
```

#### Windows
```powershell
# Run in PowerShell
irm https://raw.githubusercontent.com/Automata-Labs-team/code-sandbox-mcp/main/install.ps1 | iex
```

The installer will:
1. Check for Docker installation
2. Download the appropriate binary for your system
3. Create Claude Desktop configuration

### Manual Installation (Not necesary if automated installation is used)

1. Download the latest release for your platform from the [releases page](https://github.com/Automata-Labs-team/code-sandbox-mcp/releases)
2. Place the binary in a directory in your PATH
3. Make it executable (Unix-like systems only):
   ```bash
   chmod +x code-sandbox-mcp
   ```
## 🛠️ Available Tools

#### `run_code`
Executes code snippets in an isolated Docker container.

**Parameters:**
- `code` (string, required): The code to run
- `language` (enum, required): Programming language to use
  - Supported values: `python`, `go`, `nodejs`
  - Note: If your Python code requires external dependencies, it is recommended to use the `run_project` tool instead. Go and Node.js script dependencies are automatically installed.
- `cleanup` (string, required): Automatically removes the container after execution when set to `true`.
  - Supported values: `false` (default), `true`

**Returns:**
- Container execution output (stdout + stderr)
- Created container ID

**Features:**
- Automatic dependency detection and installation
  - Python: Detects imports and installs via pip
  - Node.js: Detects require/import statements and installs via npm
  - Go: Detects imports and installs via go get
- Automatic language-specific Docker image selection
- TypeScript/JSX support with appropriate flags
- Special handling for Go (code written to temporary file)
- Real-time output streaming

#### `run_project`
Executes a project directory in a containerized environment.

**Parameters:**
- `project_dir` (string, required): Directory containing the project to run
- `language` (enum, required): Programming language to use
  - Supported values: `python`, `go`, `nodejs`
- `entrypointCmd` (string, required): Command to run the project
  - Examples:
    - Python: `python main.py`
    - Node.js: `node index.js`
    - Go: `go run main.go`

**Returns:**
- The resource URI of the container logs.

#### `cleanup_container`
Manages removal of Docker containers after execution. Automatically called by `run_code` when cleanup is enabled.

**Parameters:**
- `containerId` (string, required): Container ID from run_code/run_project response
- `force` (string, optional): Set to 'true' to force removal of running containers

**Example:**
```json
{
  "tool": "cleanup_container",
  "arguments": {
    "containerId": "abc123",
    "force": "true"
  }
}
```

**Features:**
- Automatic dependency detection and installation
- Volume mounting of project directory
- Language-specific configuration handling
- Real-time log streaming

## 🔧 Configuration

### Claude Desktop

The installer automatically creates the configuration file. If you need to manually configure it:

#### Linux
```json
// ~/.config/Claude/claude_desktop_config.json
{
    "mcpServers": {
        "code-sandbox-mcp": {
            "command": "/path/to/code-sandbox-mcp",
            "args": [],
            "env": {}
        }
    }
}
```

#### macOS
```json
// ~/Library/Application Support/Claude/claude_desktop_config.json
{
    "mcpServers": {
        "code-sandbox-mcp": {
            "command": "/path/to/code-sandbox-mcp",
            "args": [],
            "env": {}
        }
    }
}
```

#### Windows
```json
// %APPDATA%\Claude\claude_desktop_config.json
{
    "mcpServers": {
        "code-sandbox-mcp": {
            "command": "C:\\path\\to\\code-sandbox-mcp.exe",
            "args": [],
            "env": {}
        }
    }
}
```

### Other AI Applications

For other AI applications that support MCP servers, configure them to use the `code-sandbox-mcp` binary as their code execution backend.

## 🔧 Technical Details

### Supported Languages

| Language | File Extensions | Docker Image |
|----------|----------------|--------------|
| Python | .py | python:3.12-slim-bookworm |
| Go | .go | golang:1.21-alpine |
| Node.js | .js, .ts, .tsx, .jsx | node:23-slim |

### Dependency Management

The sandbox automatically detects and installs dependencies:

- **Python**: 
  - Detects imports like `import requests`, `from PIL import Image`
  - Handles aliased imports (e.g., `PIL` → `pillow`)
  - Filters out standard library imports
  - Supports both direct imports and `__import__()` calls

- **Node.js**: 
  - Detects `require()` statements and ES6 imports
  - Handles scoped packages (e.g., `@org/package`)
  - Supports dynamic imports (`import()`)
  - Filters out built-in Node.js modules

- **Go**: 
  - Detects package imports in both single-line and grouped formats
  - Handles named and dot imports
  - Filters out standard library packages
  - Supports external dependencies via `go get`

For project execution, the following files are used:
- **Python**: requirements.txt, pyproject.toml, setup.py
- **Go**: go.mod
- **Node.js**: package.json

### TypeScript Support

Node.js 23+ includes built-in TypeScript support:
- `--experimental-strip-types`: Enabled by default for .ts files
- `--experimental-transform-types`: Used for .tsx files

## 🔐 Security Features

- Isolated execution environment using Docker containers
- Resource limitations through Docker container constraints
- Separate stdout and stderr streams
- Automatic container cleanup after execution (enabled by default in `run_code`)
- Manual cleanup option via `cleanup_container` tool
- Force removal capability for stuck containers
- Project files mounted read-only in containers

## 🛠️ Development

If you want to build the project locally or contribute to its development, see [DEVELOPMENT.md](DEVELOPMENT.md).

## 📝 License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.
