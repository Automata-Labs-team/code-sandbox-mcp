package tools

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	deps "github.com/Automata-Labs-team/code-sandbox-mcp/languages"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/image"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/moby/moby/client"
)

func RunProjectSandbox(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	var progressToken mcp.ProgressToken
	if request.Params.Meta != nil && request.Params.Meta.ProgressToken != nil {
		progressToken = request.Params.Meta.ProgressToken
	}

	language, ok := request.Params.Arguments["language"].(string)
	if !ok {
		return nil, fmt.Errorf("invalid language")
	}
	entrypoint, ok := request.Params.Arguments["entrypointCmd"].(string)
	if !ok {
		return nil, fmt.Errorf("invalid entrypoint")
	}
	projectDir, ok := request.Params.Arguments["projectDir"].(string)
	if !ok {
		return nil, fmt.Errorf("invalid projectDir")
	}

	// Validate project directory
	projectDir = filepath.Clean(projectDir)
	if _, err := os.Stat(projectDir); os.IsNotExist(err) {
		return nil, fmt.Errorf("project directory does not exist: %s", projectDir)
	}

	config := deps.SupportedLanguages[deps.Language(language)]
	containerId, err := runProjectInDocker(ctx, progressToken, strings.Fields(entrypoint), config.Image, projectDir, deps.Language(language))
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Error: %v", err)), nil
	}

	return mcp.NewToolResultText(fmt.Sprintf("Resource URI: containers://%s/logs", containerId)), nil
}

func runProjectInDocker(ctx context.Context, progressToken mcp.ProgressToken, cmd []string, dockerImage string, projectDir string, language deps.Language) (string, error) {
	server := server.ServerFromContext(ctx)
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return "", fmt.Errorf("failed to create Docker client: %w", err)
	}
	defer cli.Close()

	if progressToken != "" {
		if err := server.SendNotificationToClient(
			"notifications/progress",
			map[string]interface{}{
				"progress":      10,
				"progressToken": progressToken,
			},
		); err != nil {
			return "", fmt.Errorf("failed to send progress notification: %w", err)
		}
	}

	// Pull the Docker image
	_, err = cli.ImagePull(ctx, dockerImage, image.PullOptions{})
	if err != nil {
		return "", fmt.Errorf("failed to pull Docker image %s: %w", dockerImage, err)
	}

	// Check for dependency files and prepare install command
	var hasDepFile bool
	var depFile string
	for _, file := range deps.SupportedLanguages[language].DependencyFiles {
		if _, err := os.Stat(filepath.Join(projectDir, file)); err == nil {
			hasDepFile = true
			depFile = file
			break
		}
	}

	// Create container config with working directory set to /app
	containerConfig := &container.Config{
		Image:      dockerImage,
		WorkingDir: "/app",
		// AttachStdout: true,
		// AttachStderr: true,
		Tty: false,
	}

	// If we have dependencies, modify the command to install them first
	if hasDepFile {
		switch language {
		case deps.Python:
			if depFile == "requirements.txt" {
				containerConfig.Cmd = []string{
					"/bin/sh", "-c", fmt.Sprintf(" export DEBIAN_FRONTEND=noninteractive && set -e && apt-get update && apt-get install -y apt-utils && apt-get install -y build-essential && uv venv && . .venv/bin/activate && uv pip install -r %s && %s", depFile, strings.Join(cmd, " ")),
				}
			} else if depFile == "pyproject.toml" || depFile == "setup.py" {
				containerConfig.Cmd = []string{
					"/bin/sh", "-c", fmt.Sprintf("export DEBIAN_FRONTEND=noninteractive && set -e && apt-get update && apt-get install -y apt-utils &&apt-get install -y build-essential && uv venv && . .venv/bin/activate && uv pip install . && %s", strings.Join(cmd, " ")),
				}
			}
		case deps.Go:
			// Combine the install command with the run command
			containerConfig.Cmd = append(deps.SupportedLanguages[language].InstallCommand, cmd...)
		case deps.NodeJS:
			// Bun automatically installs dependencies when running the project, so just combine "bun" with the command after index 1
			containerConfig.Cmd = append([]string{"bun"}, cmd[1:]...)
		}
	}

	if progressToken != "" {
		server.SendNotificationToClient(
			"notifications/progress",
			map[string]interface{}{
				"progress":      50,
				"progressToken": progressToken,
			},
		)
	}

	// Mount the project directory to /app in the container
	hostConfig := &container.HostConfig{
		Binds: []string{
			fmt.Sprintf("%s:/app", projectDir),
		},
	}

	resp, err := cli.ContainerCreate(ctx, containerConfig, hostConfig, nil, nil, "")
	if err != nil {
		return "", fmt.Errorf("failed to create container: %w", err)
	}

	if progressToken != "" {
		server.SendNotificationToClient(
			"notifications/progress",
			map[string]interface{}{
				"progress":      75,
				"progressToken": progressToken,
			},
		)
	}

	if err := cli.ContainerStart(ctx, resp.ID, container.StartOptions{}); err != nil {
		return "", fmt.Errorf("failed to start container: %w", err)
	}

	if progressToken != "" {
		server.SendNotificationToClient(
			"notifications/progress",
			map[string]interface{}{
				"progress":      100,
				"progressToken": progressToken,
			},
		)
	}

	return resp.ID, nil
}
