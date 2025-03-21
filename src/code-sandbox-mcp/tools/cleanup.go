package tools

import (
	"context"
	"fmt"
	"time"
	"log"

	"github.com/docker/docker/api/types/container"
	"github.com/moby/moby/client"
	"github.com/mark3labs/mcp-go/mcp"
)

var CleanupEnabled bool

// CleanupContainer removes a container after it has finished executing
func CleanupContainer(ctx context.Context, containerID string, waitForExit bool, force bool, timeoutSeconds int) error {

    // Create Docker client
    cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
    if err != nil {
        return fmt.Errorf("Failed to create Docker client for cleanup: %w", err)
    }
    defer cli.Close()

    if waitForExit {
        // Wait for container to exit with timeout
        statusCh, errCh := cli.ContainerWait(ctx, containerID, container.WaitConditionNotRunning)
        
        select {
        case err := <-errCh:
            if err != nil {
                fmt.Errorf("Error waiting for container %s: %v\n", containerID, err)
                // Continue with removal even if there's an error waiting
            }
        case <-statusCh:
            // Container exited normally
            log.Printf("Container %s exited, proceeding with cleanup\n", containerID)
        case <-time.After(time.Duration(timeoutSeconds) * time.Second):
            log.Printf("Timeout waiting for container %s to exit\n", containerID)
            // Continue with removal
        }
    }

    // Remove the container
    removeOptions := container.RemoveOptions{
        Force: force,
    }
    
    if err := cli.ContainerRemove(ctx, containerID, removeOptions); err != nil {
        return fmt.Errorf("Failed to remove container %s: %w", containerID, err)
    }
    
    log.Printf("Successfully removed container %s\n", containerID)
    return nil
}

// CleanupContainerTool is the handler for the cleanup_container tool
func CleanupContainerTool(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
    containerID, ok := request.Params.Arguments["containerId"].(string)
    if !ok || containerID == "" {
        return mcp.NewToolResultError("Container ID is required"), nil
    }

    forceStr, _ := request.Params.Arguments["force"].(string)
    force := forceStr == "true"

    // Create Docker client
    cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
    if err != nil {
        return mcp.NewToolResultError(fmt.Sprintf("Failed to create Docker client: %v", err)), nil
    }
    defer cli.Close()

    // Verify container exists
    _, err = cli.ContainerInspect(ctx, containerID)
    if err != nil {
        return mcp.NewToolResultError(fmt.Sprintf("Container %s not found: %v", containerID, err)), nil
    }

    // Call the existing cleanup function
    err = CleanupContainer(ctx, containerID, false, force, 30)
    if err != nil {
        return mcp.NewToolResultError(fmt.Sprintf("Failed to clean up container: %v", err)), nil
    }

    return mcp.NewToolResultText(fmt.Sprintf("Successfully cleaned up container %s", containerID)), nil
}