package tools

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/Automata-Labs-team/code-sandbox-mcp/languages"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/image"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/moby/moby/client"
	"github.com/moby/moby/pkg/stdcopy"
)

func RunCodeSandbox(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	arguments := request.Params.Arguments
	steps, _ := arguments["steps"].(float64)
	if steps == 0 {
		steps = 100
	}
	server := server.ServerFromContext(ctx)
	var progressToken mcp.ProgressToken
	if request.Params.Meta != nil && request.Params.Meta.ProgressToken != nil {
		progressToken = request.Params.Meta.ProgressToken
	}

	language, ok := request.Params.Arguments["language"].(string)
	if !ok {
		return mcp.NewToolResultError(fmt.Sprintf("Language not supported: %s", request.Params.Arguments["language"])), nil
	}
	code, ok := request.Params.Arguments["code"].(string)
	if !ok {
		return mcp.NewToolResultError("language must be a string"), nil
	}
	parsed := languages.Language(language)
	config := languages.SupportedLanguages[languages.Language(language)]

	if progressToken != "" {
		if err := server.SendNotificationToClient(
			"notifications/progress",
			map[string]interface{}{
				"progress":      int(10),
				"total":         int(steps),
				"progressToken": progressToken,
			},
		); err != nil {
			return &mcp.CallToolResult{
				Content: []interface{}{
					mcp.NewTextContent("Could not send progress to client"),
				},
				IsError: false,
			}, nil
		}
	}

	cmd := config.RunCommand
	escapedCode := strings.ToValidUTF8(code, "")

	// Create a channel to receive the result from runInDocker
	resultCh := make(chan struct {
		logs string
		err  error
	}, 1)

	// Run the Docker container in a goroutine
	go func() {
		logs, err := runInDocker(ctx, cmd, config.Image, escapedCode, parsed)
		resultCh <- struct {
			logs string
			err  error
		}{logs, err}
	}()

	progress := 20
	for {
		select {
		case result := <-resultCh:
			if progressToken != "" {
				// Send final progress update
				_ = server.SendNotificationToClient(
					"notifications/progress",
					map[string]interface{}{
						"progress":      100,
						"total":         int(steps),
						"progressToken": progressToken,
					},
				)
			}
			if result.err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("Error: %v", result.err)), nil
			}
			return mcp.NewToolResultText(fmt.Sprintf("Logs: %s", result.logs)), nil
		default:
			time.Sleep(2 * time.Second)
			if progressToken != "" {
				if progress >= 90 && progress < 100 {
					progress = progress + 1
				} else {
					progress = progress + 5
				}
				if err := server.SendNotificationToClient(
					"notifications/progress",
					map[string]interface{}{
						"progress":      progress,
						"total":         int(steps),
						"progressToken": progressToken,
					},
				); err != nil {
					server.SendNotificationToClient("notifications/error", map[string]interface{}{
						"message": fmt.Sprintf("Failed to send progress: %v", err),
					})
				}
			}
		}
	}
}

func runInDocker(ctx context.Context, cmd []string, dockerImage string, code string, language languages.Language) (string, error) {
	cli, err := client.NewClientWithOpts(
		client.FromEnv,
		client.WithAPIVersionNegotiation(),
	)
	if err != nil {
		return "", fmt.Errorf("failed to create Docker client: %w", err)
	}
	defer cli.Close()

	// Pull the Docker image
	reader, err := cli.ImagePull(ctx, dockerImage, image.PullOptions{})
	if err != nil {
		return "", fmt.Errorf("failed to pull Docker image %s: %w", dockerImage, err)
	}
	defer reader.Close()

	_, err = io.Copy(io.Discard, reader)
	if err != nil {
		return "", fmt.Errorf("failed to copy Docker image pull output: %w", err)
	}

	// Create container config
	config := &container.Config{
		Image: dockerImage,
		Cmd:   cmd,
		Tty:   false,
	}

	// Create a temporary directory for the Go file
	tmpDir, err := os.MkdirTemp("", "docker-sandbox-*")
	if err != nil {
		return "", fmt.Errorf("failed to create temporary directory: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	// Write the code to a file in the temporary directory
	tmpFile := filepath.Join(tmpDir, "main."+languages.SupportedLanguages[language].FileExtension)
	err = os.WriteFile(tmpFile, []byte(code), 0644)
	if err != nil {
		return "", fmt.Errorf("failed to write code to temporary file: %w", err)
	}

	// Mount the temporary directory to /app and set it as working directory
	hostConfig := &container.HostConfig{
		Binds: []string{
			fmt.Sprintf("%s:/app", tmpDir),
		},
	}

	// Update container config to work in the mounted directory
	config.WorkingDir = "/app"

	sandboxContainer, err := cli.ContainerCreate(ctx, config, hostConfig, nil, nil, "")
	if err != nil {
		return "", fmt.Errorf("failed to create container: %w", err)
	}

	if err := cli.ContainerStart(ctx, sandboxContainer.ID, container.StartOptions{}); err != nil {
		return "", fmt.Errorf("failed to start container: %w", err)
	}

	// Wait for container to finish
	statusCh, errCh := cli.ContainerWait(ctx, sandboxContainer.ID, container.WaitConditionNotRunning)

	select {
	case err := <-errCh:
		if err != nil {
			panic(err)
		}
	case <-statusCh:
	}

	out, err := cli.ContainerLogs(ctx, sandboxContainer.ID, container.LogsOptions{ShowStdout: true, ShowStderr: true})
	if err != nil {
		return "", fmt.Errorf("failed to get container logs: %w", err)
	}
	defer out.Close()

	var b strings.Builder
	_, err = stdcopy.StdCopy(&b, &b, out)
	if err != nil {
		return "", fmt.Errorf("failed to copy container output: %w", err)
	}

	return b.String(), nil
}
