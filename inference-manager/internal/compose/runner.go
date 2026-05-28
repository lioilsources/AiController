package compose

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
)

// Runner controls docker compose stacks.
type Runner interface {
	Up(ctx context.Context, composeFile string) error
	Down(ctx context.Context, composeFile string) error
}

// DockerRunner invokes the docker compose CLI.
type DockerRunner struct{}

func New() *DockerRunner { return &DockerRunner{} }

func (r *DockerRunner) Up(ctx context.Context, composeFile string) error {
	return run(ctx, composeFile, "up", "-d", "--remove-orphans")
}

func (r *DockerRunner) Down(ctx context.Context, composeFile string) error {
	return run(ctx, composeFile, "down")
}

func run(ctx context.Context, composeFile string, args ...string) error {
	cmdArgs := append([]string{"compose", "-f", composeFile}, args...)
	cmd := exec.CommandContext(ctx, "docker", cmdArgs...)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("docker %v: %w: %s", cmdArgs, err, stderr.String())
	}
	return nil
}
