package app

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"composepack/internal/core/chart"
	"composepack/internal/core/dockercompose"
	"composepack/internal/core/release"
	releaseruntime "composepack/internal/core/runtime"
	"composepack/internal/core/templating"
	"composepack/internal/core/values"
	"composepack/internal/infra/config"
	"composepack/internal/infra/logging"
	"composepack/internal/infra/process"
	"composepack/internal/util/fileloader"

	"sigs.k8s.io/yaml"
)

// ErrNotImplemented is a shared placeholder for unimplemented application flows.
var ErrNotImplemented = errors.New("not implemented")

// Runtime aggregates long-lived dependencies that commands rely on.
type Runtime struct {
	Config         config.Config
	Logger         logging.Logger
	ChartLoader    chart.Loader
	TemplateEngine *templating.Engine
	RuntimeWriter  *releaseruntime.Writer
	ProcessRunner  *process.Runner
	DockerRunner   *dockercompose.Runner
	ReleaseStore   *release.Store
}

// NewRuntime wires default implementations for the runtime container.
func NewRuntime(cfg config.Config, logger logging.Logger, loader chart.Loader) *Runtime {
	if logger == nil {
		logger = logging.Nop{}
	}
	if loader == nil {
		loader = NewDefaultChartLoader()
	}
	procRunner := process.NewRunner()

	return &Runtime{
		Config:         cfg,
		Logger:         logger,
		ChartLoader:    loader,
		TemplateEngine: templating.NewEngine(),
		RuntimeWriter:  &releaseruntime.Writer{},
		ProcessRunner:  procRunner,
		DockerRunner:   dockercompose.NewRunner(procRunner),
		ReleaseStore:   &release.Store{},
	}
}

// NewDefaultChartLoader constructs the default filesystem chart loader.
func NewDefaultChartLoader() chart.Loader {
	fs := chart.NewFileSystemChartLoader(fileloader.NewFileSystemLoader())
	return chart.NewCompositeLoader(fs)
}

// Application provides methods that implement workflows such as install/up/down.
type Application struct {
	Runtime *Runtime
}

// NewApplication binds a runtime container to a high-level application service.
func NewApplication(rt *Runtime) *Application {
	return &Application{Runtime: rt}
}

// RenderOptions capture the shared knobs across install/template/up workflows.
type RenderOptions struct {
	ReleaseName    string
	ChartSource    string
	ValueFiles     []string
	SetValues      map[string]string
	RuntimeBaseDir string
}

// InstallOptions drives chart installation into a runtime directory.
type InstallOptions struct {
	RenderOptions
	AutoStart bool
}

// TemplateOptions render templates without invoking Docker Compose.
type TemplateOptions struct {
	RenderOptions
}

// UpOptions render and run docker compose up.
type UpOptions struct {
	RenderOptions
	Detach bool
}

// DownOptions control docker compose down behavior.
type DownOptions struct {
	ReleaseName    string
	RuntimeBaseDir string
	RemoveVolumes  bool
}

// LogsOptions control docker compose logs streaming.
type LogsOptions struct {
	ReleaseName    string
	RuntimeBaseDir string
	Follow         bool
	Tail           int
}

// PSOptions control docker compose ps display.
type PSOptions struct {
	ReleaseName    string
	RuntimeBaseDir string
}

// InstallRelease implements the install workflow described in the PRD.
func (a *Application) InstallRelease(ctx context.Context, opts InstallOptions) error {
	runtimeDir, _, err := a.renderRelease(ctx, opts.RenderOptions)
	if err != nil {
		return err
	}
	if !opts.AutoStart {
		return nil
	}
	args := []string{"up", "-d"}
	return a.Runtime.DockerRunner.Run(ctx, dockercompose.CommandOptions{
		WorkingDir: runtimeDir,
		Args:       args,
	})
}

// TemplateRelease renders templates and writes runtime files without running containers.
func (a *Application) TemplateRelease(ctx context.Context, opts TemplateOptions) error {
	_, _, err := a.renderRelease(ctx, opts.RenderOptions)
	return err
}

// UpRelease re-renders templates and invokes docker compose up.
func (a *Application) UpRelease(ctx context.Context, opts UpOptions) error {
	runtimeDir, _, err := a.renderRelease(ctx, opts.RenderOptions)
	if err != nil {
		return err
	}
	args := []string{"up"}
	if opts.Detach {
		args = append(args, "-d")
	}
	return a.Runtime.DockerRunner.Run(ctx, dockercompose.CommandOptions{
		WorkingDir: runtimeDir,
		Args:       args,
	})
}

// DownRelease shells out to docker compose down for the given release.
func (a *Application) DownRelease(ctx context.Context, opts DownOptions) error {
	runtimeDir, err := a.runtimeDir(opts.RuntimeBaseDir, opts.ReleaseName)
	if err != nil {
		return err
	}

	args := []string{"down"}
	if opts.RemoveVolumes {
		args = append(args, "--volumes")
	}

	return a.Runtime.DockerRunner.Run(ctx, dockercompose.CommandOptions{
		WorkingDir: runtimeDir,
		Args:       args,
	})
}

// StreamLogs tails docker compose logs for the release.
func (a *Application) StreamLogs(ctx context.Context, opts LogsOptions) error {
	runtimeDir, err := a.runtimeDir(opts.RuntimeBaseDir, opts.ReleaseName)
	if err != nil {
		return err
	}

	args := []string{"logs"}
	if opts.Follow {
		args = append(args, "--follow")
	}
	if opts.Tail >= 0 {
		args = append(args, "--tail", fmt.Sprintf("%d", opts.Tail))
	}

	return a.Runtime.DockerRunner.Run(ctx, dockercompose.CommandOptions{
		WorkingDir: runtimeDir,
		Args:       args,
	})
}

// ShowStatus surfaces docker compose ps data.
func (a *Application) ShowStatus(ctx context.Context, opts PSOptions) error {
	runtimeDir, err := a.runtimeDir(opts.RuntimeBaseDir, opts.ReleaseName)
	if err != nil {
		return err
	}

	args := []string{"ps"}
	return a.Runtime.DockerRunner.Run(ctx, dockercompose.CommandOptions{
		WorkingDir: runtimeDir,
		Args:       args,
	})
}

func (a *Application) renderRelease(ctx context.Context, opts RenderOptions) (string, *release.Metadata, error) {
	if opts.ReleaseName == "" {
		return "", nil, errors.New("release name is required")
	}
	if opts.ChartSource == "" {
		return "", nil, errors.New("chart source must be provided")
	}

	baseDir, err := a.resolveBaseDir(opts.RuntimeBaseDir)
	if err != nil {
		return "", nil, err
	}

	ch, err := a.Runtime.ChartLoader.Load(ctx, opts.ChartSource)
	if err != nil {
		return "", nil, fmt.Errorf("load chart: %w", err)
	}

	mergedValues, valueSources, err := a.buildValues(ch, opts)
	if err != nil {
		return "", nil, err
	}

	rc := templating.RenderContext{
		Values: mergedValues,
		Env:    captureEnv(),
		Release: templating.ReleaseInfo{
			Name: opts.ReleaseName,
		},
		Chart: templating.ChartInfo{
			Name:    ch.Metadata.Name,
			Version: ch.Metadata.Version,
		},
		Files: templating.NewFilesAccessor(ch.StaticFiles),
	}

	composeFragments, err := a.Runtime.TemplateEngine.RenderComposeFragments(ctx, ch, rc)
	if err != nil {
		return "", nil, fmt.Errorf("render compose templates: %w", err)
	}
	if len(composeFragments) == 0 {
		return "", nil, errors.New("chart produced no compose templates")
	}

	fileAssets, err := a.Runtime.TemplateEngine.RenderFiles(ctx, ch, rc)
	if err != nil {
		return "", nil, fmt.Errorf("render file templates: %w", err)
	}

	mergedCompose, orderedFragments, err := a.mergeFragments(ctx, composeFragments, fileAssets, opts.ReleaseName)
	if err != nil {
		return "", nil, err
	}

	runtimeDir, err := a.Runtime.RuntimeWriter.Write(ctx, releaseruntime.WriteOptions{
		ReleaseName: opts.ReleaseName,
		BaseDir:     baseDir,
		ComposeYAML: mergedCompose,
		Files:       fileAssets,
	})
	if err != nil {
		return "", nil, fmt.Errorf("write runtime directory: %w", err)
	}

	meta := &release.Metadata{
		ReleaseName:   opts.ReleaseName,
		ChartName:     ch.Metadata.Name,
		ChartVersion:  ch.Metadata.Version,
		Values:        deepCopyMap(mergedValues),
		ValuesSources: valueSources,
		ComposeFiles:  orderedFragments,
	}

	if err := a.Runtime.ReleaseStore.Save(ctx, runtimeDir, meta); err != nil {
		return "", nil, fmt.Errorf("save release metadata: %w", err)
	}

	return runtimeDir, meta, nil
}

func (a *Application) resolveBaseDir(override string) (string, error) {
	if override != "" {
		return override, nil
	}
	if a.Runtime == nil {
		return "", errors.New("runtime is not configured")
	}
	if a.Runtime.Config.ReleasesBaseDir == "" {
		return "", errors.New("releases base directory is not configured")
	}
	return a.Runtime.Config.ReleasesBaseDir, nil
}

func (a *Application) runtimeDir(baseOverride, release string) (string, error) {
	if release == "" {
		return "", errors.New("release name is required")
	}
	base, err := a.resolveBaseDir(baseOverride)
	if err != nil {
		return "", err
	}
	return filepath.Join(base, release), nil
}

func (a *Application) buildValues(ch *chart.Chart, opts RenderOptions) (map[string]any, []string, error) {
	var result map[string]any
	if ch.Values != nil {
		copied := deepCopyMap(ch.Values)
		result = copied
	} else {
		result = map[string]any{}
	}

	sources := []string{"chart:values.yaml"}

	for _, path := range opts.ValueFiles {
		contents, err := loadValuesFile(path)
		if err != nil {
			return nil, nil, fmt.Errorf("load values file %s: %w", path, err)
		}
		result, err = values.Merge(result, contents)
		if err != nil {
			return nil, nil, fmt.Errorf("merge values file %s: %w", path, err)
		}
		sources = append(sources, path)
	}

	if len(opts.SetValues) > 0 {
		setOverrides := buildSetOverrides(opts.SetValues)
		if len(setOverrides) > 0 {
			var err error
			result, err = values.Merge(result, setOverrides)
			if err != nil {
				return nil, nil, fmt.Errorf("apply --set overrides: %w", err)
			}
			sources = append(sources, "cli:set")
		}
	}

	if err := values.Validate(ch.ValuesSchema, result); err != nil {
		return nil, nil, fmt.Errorf("validate values: %w", err)
	}

	return result, sources, nil
}

func (a *Application) mergeFragments(ctx context.Context, fragments map[string][]byte, files map[string][]byte, releaseName string) ([]byte, []string, error) {
	tempDir, err := os.MkdirTemp("", "composepack-fragments-*")
	if err != nil {
		return nil, nil, fmt.Errorf("create temp directory: %w", err)
	}
	defer os.RemoveAll(tempDir)

	names := make([]string, 0, len(fragments))
	for name := range fragments {
		names = append(names, name)
	}
	sort.Strings(names)

	var fragmentPaths []string
	for _, name := range names {
		dest := filepath.Join(tempDir, name)
		if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
			return nil, nil, fmt.Errorf("prepare fragment directory: %w", err)
		}
		if err := os.WriteFile(dest, fragments[name], 0o644); err != nil {
			return nil, nil, fmt.Errorf("write fragment %s: %w", name, err)
		}
		fragmentPaths = append(fragmentPaths, dest)
	}

	if len(files) > 0 {
		filesRoot := filepath.Join(tempDir, "files")
		for path, data := range files {
			target := filepath.Join(filesRoot, path)
			if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
				return nil, nil, fmt.Errorf("prepare file asset directory: %w", err)
			}
			if err := os.WriteFile(target, data, 0o644); err != nil {
				return nil, nil, fmt.Errorf("write file asset %s: %w", path, err)
			}
		}
	}

	data, err := a.Runtime.DockerRunner.MergeFragments(ctx, dockercompose.MergeOptions{
		WorkingDir:    tempDir,
		FragmentPaths: fragmentPaths,
		ProjectName:   releaseName,
	})
	if err != nil {
		return nil, nil, err
	}

	rendered := strings.ReplaceAll(string(data), tempDir, ".")
	return []byte(rendered), names, nil
}

func loadValuesFile(path string) (map[string]any, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	if len(data) == 0 {
		return map[string]any{}, nil
	}

	var out map[string]any
	if err := yaml.Unmarshal(data, &out); err != nil {
		return nil, err
	}
	return out, nil
}

func buildSetOverrides(in map[string]string) map[string]any {
	out := make(map[string]any, len(in))
	for key, val := range in {
		assignSetValue(out, strings.Split(key, "."), val)
	}
	return out
}

func assignSetValue(dst map[string]any, path []string, value string) {
	if len(path) == 0 {
		return
	}
	key := path[0]
	if len(path) == 1 {
		dst[key] = value
		return
	}

	next, ok := dst[key].(map[string]any)
	if !ok {
		next = map[string]any{}
		dst[key] = next
	}
	assignSetValue(next, path[1:], value)
}

func deepCopyMap(src map[string]any) map[string]any {
	if src == nil {
		return nil
	}
	dst := make(map[string]any, len(src))
	for k, v := range src {
		dst[k] = deepCopyValue(v)
	}
	return dst
}

func deepCopyValue(val any) any {
	switch typed := val.(type) {
	case map[string]any:
		return deepCopyMap(typed)
	case []any:
		res := make([]any, len(typed))
		for i, v := range typed {
			res[i] = deepCopyValue(v)
		}
		return res
	default:
		return typed
	}
}

func captureEnv() map[string]string {
	env := os.Environ()
	out := make(map[string]string, len(env))
	for _, kv := range env {
		if kv == "" {
			continue
		}
		parts := strings.SplitN(kv, "=", 2)
		if len(parts) != 2 {
			continue
		}
		out[parts[0]] = parts[1]
	}
	return out
}
