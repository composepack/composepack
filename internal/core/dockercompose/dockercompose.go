package dockercompose

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"composepack/internal/infra/process"
)

var (
	defaultComposeCommand = []string{"docker", "compose"}
	legacyComposeCommand  = []string{"docker-compose"}
)

// Runner wraps exec invocations to `docker compose` / `docker-compose`.
type Runner struct {
	exec     *process.Runner
	primary  []string
	fallback []string
}

// NewRunner constructs a docker compose runner using the provided process runner.
func NewRunner(execRunner *process.Runner) *Runner {
	if execRunner == nil {
		execRunner = process.NewRunner()
	}
	return &Runner{
		exec:     execRunner,
		primary:  defaultComposeCommand,
		fallback: legacyComposeCommand,
	}
}

// MergeOptions tunes how compose fragments should be merged via `docker compose config`.
type MergeOptions struct {
	WorkingDir    string
	FragmentPaths []string
	ProjectName   string
}

// CommandOptions describe docker compose command invocations from runtime directories.
type CommandOptions struct {
	WorkingDir string
	Args       []string
}

// MergeFragments shells out to `docker compose config` to get a merged YAML.
func (r *Runner) MergeFragments(ctx context.Context, opts MergeOptions) ([]byte, error) {
	if len(opts.FragmentPaths) == 0 {
		return nil, errors.New("at least one compose fragment is required")
	}

	args := make([]string, 0, len(opts.FragmentPaths)*2+1)
	for _, path := range opts.FragmentPaths {
		args = append(args, "-f", path)
	}
	args = append(args, "config")

	stdout, stderr, err := r.run(ctx, opts.WorkingDir, args, opts.ProjectName)
	if err != nil {
		return nil, composeError("docker compose config", err, stderr)
	}

	return stdout, nil
}

// Run executes docker compose commands (up/down/logs/etc) in the runtime directory.
func (r *Runner) Run(ctx context.Context, opts CommandOptions) error {
	if opts.WorkingDir == "" {
		return errors.New("working directory is required")
	}
	if len(opts.Args) == 0 {
		return errors.New("docker compose arguments are required")
	}

	_, stderr, err := r.run(ctx, opts.WorkingDir, opts.Args, "")
	if err != nil {
		return composeError("docker compose", err, stderr)
	}
	return nil
}

// RunWithOutput executes docker compose commands and returns stdout.
func (r *Runner) RunWithOutput(ctx context.Context, opts CommandOptions) ([]byte, error) {
	if opts.WorkingDir == "" {
		return nil, errors.New("working directory is required")
	}
	if len(opts.Args) == 0 {
		return nil, errors.New("docker compose arguments are required")
	}

	stdout, stderr, err := r.run(ctx, opts.WorkingDir, opts.Args, "")
	if err != nil {
		return nil, composeError("docker compose", err, stderr)
	}
	return stdout, nil
}

func (r *Runner) run(ctx context.Context, dir string, args []string, project string) ([]byte, []byte, error) {
	stdout, stderr, err := r.exec.Run(ctx, process.Command{
		Name: r.primary[0],
		Args: append(append([]string{}, r.primary[1:]...), args...),
		Dir:  dir,
		Env:  composeEnv(project),
	})
	if err == nil {
		return stdout, stderr, nil
	}
	if !process.IsNotFound(err) || len(r.fallback) == 0 {
		return stdout, stderr, err
	}

	return r.exec.Run(ctx, process.Command{
		Name: r.fallback[0],
		Args: append(append([]string{}, r.fallback[1:]...), args...),
		Dir:  dir,
		Env:  composeEnv(project),
	})
}

func composeEnv(project string) []string {
	if project == "" {
		return nil
	}
	return []string{fmt.Sprintf("COMPOSE_PROJECT_NAME=%s", project)}
}
func composeError(action string, err error, stderr []byte) error {
	msg := strings.TrimSpace(string(stderr))
	if msg != "" {
		return fmt.Errorf("%s failed: %w: %s", action, err, msg)
	}
	return fmt.Errorf("%s failed: %w", action, err)
}
