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
	RuntimePath    string
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
	RuntimePath    string
	RemoveVolumes  bool
}

// LogsOptions control docker compose logs streaming.
type LogsOptions struct {
	ReleaseName    string
	RuntimeBaseDir string
	RuntimePath    string
	Follow         bool
	Tail           int
}

// PSOptions control docker compose ps display.
type PSOptions struct {
	ReleaseName    string
	RuntimeBaseDir string
	RuntimePath    string
}

// DiffOptions control drift detection between current and proposed state.
type DiffOptions struct {
	RenderOptions
	ShowFiles    bool
	ContextLines int
}

// InstallRelease implements the install workflow described in the PRD.
// It only works for new releases and will error if the release already exists.
func (a *Application) InstallRelease(ctx context.Context, opts InstallOptions) error {
	// Check if release already exists
	_, runtimeDir, err := a.resolveRuntimeLocation(opts.ReleaseName, opts.RuntimeBaseDir, opts.RuntimePath)
	if err != nil {
		return err
	}

	existingMeta, err := a.Runtime.ReleaseStore.Load(ctx, runtimeDir)
	if err != nil {
		return fmt.Errorf("check existing release: %w", err)
	}
	if existingMeta != nil {
		return fmt.Errorf("release %s already exists (use 'composepack apply' to update an existing release)", opts.ReleaseName)
	}

	// Render and install the release
	runtimeDir, _, err = a.renderRelease(ctx, opts.RenderOptions)
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

// ApplyOptions drives chart application/update into an existing or new runtime directory.
type ApplyOptions struct {
	RenderOptions
	AutoStart bool
}

// ApplyRelease implements the apply workflow (like Helm upgrade).
// It runs docker compose down first if the release exists, then installs/updates.
func (a *Application) ApplyRelease(ctx context.Context, opts ApplyOptions) error {
	// Check if release exists
	_, runtimeDir, err := a.resolveRuntimeLocation(opts.ReleaseName, opts.RuntimeBaseDir, opts.RuntimePath)
	if err != nil {
		return err
	}

	existingMeta, err := a.Runtime.ReleaseStore.Load(ctx, runtimeDir)
	if err != nil {
		return fmt.Errorf("check existing release: %w", err)
	}

	// If release exists, run docker compose down first
	if existingMeta != nil {
		// Check if docker-compose.yaml exists (release might be partially created)
		composePath := filepath.Join(runtimeDir, "docker-compose.yaml")
		if _, err := os.Stat(composePath); err == nil {
			// Run docker compose down (ignore errors if containers aren't running)
			_ = a.Runtime.DockerRunner.Run(ctx, dockercompose.CommandOptions{
				WorkingDir: runtimeDir,
				Args:       []string{"down"},
			})
		}
	}

	// Render and install the release
	runtimeDir, _, err = a.renderRelease(ctx, opts.RenderOptions)
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
	_, runtimeDir, err := a.resolveRuntimeLocation(opts.ReleaseName, opts.RuntimeBaseDir, opts.RuntimePath)
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
	_, runtimeDir, err := a.resolveRuntimeLocation(opts.ReleaseName, opts.RuntimeBaseDir, opts.RuntimePath)
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

	if opts.Follow {
		// For follow mode, stream directly without capturing
		return a.Runtime.DockerRunner.Run(ctx, dockercompose.CommandOptions{
			WorkingDir: runtimeDir,
			Args:       args,
		})
	}

	// For non-follow mode, capture and print output
	output, err := a.Runtime.DockerRunner.RunWithOutput(ctx, dockercompose.CommandOptions{
		WorkingDir: runtimeDir,
		Args:       args,
	})
	if err != nil {
		return err
	}
	if len(output) > 0 {
		fmt.Print(string(output))
	}
	return nil
}

// ShowStatus surfaces docker compose ps data.
func (a *Application) ShowStatus(ctx context.Context, opts PSOptions) error {
	_, runtimeDir, err := a.resolveRuntimeLocation(opts.ReleaseName, opts.RuntimeBaseDir, opts.RuntimePath)
	if err != nil {
		return err
	}

	args := []string{"ps"}
	output, err := a.Runtime.DockerRunner.RunWithOutput(ctx, dockercompose.CommandOptions{
		WorkingDir: runtimeDir,
		Args:       args,
	})
	if err != nil {
		return err
	}
	if len(output) > 0 {
		fmt.Print(string(output))
	}
	return nil
}

// DiffRelease compares the current release with what would be deployed.
// If no release exists, it shows what would be created.
func (a *Application) DiffRelease(ctx context.Context, opts DiffOptions) error {
	// Determine chart source - use provided one, or try to infer from existing release
	chartSource := opts.ChartSource

	// Load the existing release (if it exists)
	_, currentRuntimeDir, err := a.resolveRuntimeLocation(opts.ReleaseName, opts.RuntimeBaseDir, opts.RuntimePath)
	if err != nil {
		return err
	}

	// Load current release metadata (may be nil if release doesn't exist)
	currentMeta, err := a.Runtime.ReleaseStore.Load(ctx, currentRuntimeDir)
	if err != nil {
		return fmt.Errorf("load current release metadata: %w", err)
	}

	// Auto-resolve chart source: use provided one, or infer from existing release
	if chartSource == "" {
		if currentMeta == nil {
			return fmt.Errorf("--chart is required (release %s doesn't exist yet; specify chart to see what would be created)", opts.ReleaseName)
		}
		if currentMeta.ChartSource == "" {
			return fmt.Errorf("release %s exists but chart source is unknown (provide --chart to compare)", opts.ReleaseName)
		}
		// Auto-resolve from existing release metadata
		chartSource = currentMeta.ChartSource
	}

	// Render the proposed new release in memory (don't write to disk)
	ch, err := a.Runtime.ChartLoader.Load(ctx, chartSource)
	if err != nil {
		return fmt.Errorf("load chart: %w", err)
	}

	mergedValues, _, err := a.buildValues(ch, opts.RenderOptions)
	if err != nil {
		return err
	}

	rc := templating.RenderContext{
		Values: mergedValues,
		Env:    captureEnv(),
		Release: templating.ReleaseInfo{
			Name: opts.ReleaseName,
		},
		Chart: ch.Metadata,
		Files: templating.NewFilesAccessor(ch.StaticFiles),
	}

	newComposeFragments, err := a.Runtime.TemplateEngine.RenderComposeFragments(ctx, ch, rc)
	if err != nil {
		return fmt.Errorf("render new compose templates: %w", err)
	}

	newFileAssets, err := a.Runtime.TemplateEngine.RenderFiles(ctx, ch, rc)
	if err != nil {
		return fmt.Errorf("render new file templates: %w", err)
	}

	newMergedCompose, _, err := a.mergeFragments(ctx, newComposeFragments, newFileAssets, opts.ReleaseName)
	if err != nil {
		return err
	}

	// If no existing release, show what would be created
	if currentMeta == nil {
		return a.showDiff(nil, newMergedCompose, nil, newFileAssets, opts)
	}

	// Load current compose file
	currentComposePath := filepath.Join(currentRuntimeDir, "docker-compose.yaml")
	currentCompose, err := os.ReadFile(currentComposePath)
	if err != nil {
		return fmt.Errorf("read current compose file: %w", err)
	}

	// Load current files
	currentFiles, err := a.loadCurrentFiles(currentRuntimeDir)
	if err != nil {
		return fmt.Errorf("load current files: %w", err)
	}

	// Perform the diff
	return a.showDiff(currentCompose, newMergedCompose, currentFiles, newFileAssets, opts)
}

func (a *Application) renderRelease(ctx context.Context, opts RenderOptions) (string, *release.Metadata, error) {
	if opts.ReleaseName == "" {
		return "", nil, errors.New("release name is required")
	}
	if opts.ChartSource == "" {
		return "", nil, errors.New("chart source must be provided")
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
		Chart: ch.Metadata,
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

	baseDir, _, err := a.resolveRuntimeLocation(opts.ReleaseName, opts.RuntimeBaseDir, opts.RuntimePath)
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
		ChartMetadata: ch.Metadata,
		ChartSource:   opts.ChartSource,
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

func (a *Application) resolveRuntimeLocation(release, baseOverride, runtimePath string) (string, string, error) {
	if release == "" {
		return "", "", errors.New("release name is required")
	}
	if runtimePath != "" {
		abs, err := filepath.Abs(runtimePath)
		if err != nil {
			return "", "", fmt.Errorf("resolve runtime path: %w", err)
		}
		if filepath.Base(abs) != release {
			return "", "", fmt.Errorf("runtime directory %s does not match release %s", abs, release)
		}
		return filepath.Dir(abs), abs, nil
	}

	base, err := a.resolveBaseDir(baseOverride)
	if err != nil {
		return "", "", err
	}
	return base, filepath.Join(base, release), nil
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

func (a *Application) loadCurrentFiles(runtimeDir string) (map[string][]byte, error) {
	filesDir := filepath.Join(runtimeDir, "files")
	files := make(map[string][]byte)

	// Check if files directory exists
	if _, err := os.Stat(filesDir); os.IsNotExist(err) {
		return files, nil
	}

	err := filepath.Walk(filesDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}

		relPath, err := filepath.Rel(filesDir, path)
		if err != nil {
			return err
		}

		data, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("read file %s: %w", relPath, err)
		}

		files[relPath] = data
		return nil
	})

	if err != nil {
		return nil, err
	}

	return files, nil
}

func (a *Application) showDiff(currentCompose, newCompose []byte, currentFiles, newFiles map[string][]byte, opts DiffOptions) error {
	// Import the differ package inline to use it
	return showDiffOutput(currentCompose, newCompose, currentFiles, newFiles, opts.ShowFiles, opts.ContextLines)
}

func showDiffOutput(currentCompose, newCompose []byte, currentFiles, newFiles map[string][]byte, showFiles bool, contextLines int) error {
	// Handle case where no existing release (everything is new)
	if currentCompose == nil {
		fmt.Println("📝 New Release - What would be created:")
		fmt.Println()
		fmt.Println("Docker Compose Configuration:")
		fmt.Println(string(newCompose))
		fmt.Println()

		// Extract services that would be created
		affected := extractAffectedServices(nil, newCompose)
		if len(affected) > 0 {
			fmt.Println("🚀 Services that would be created:")
			for _, svc := range affected {
				fmt.Printf("  • %s\n", svc)
			}
			fmt.Println()
		}

		// Show files that would be created
		if len(newFiles) > 0 {
			fmt.Println("📁 Files that would be created:")
			for path := range newFiles {
				fmt.Printf("  + %s\n", path)
			}
			fmt.Println()
		}

		if showFiles && len(newFiles) > 0 {
			fmt.Println("📄 File Contents:")
			for path, data := range newFiles {
				fmt.Printf("\n--- %s\n", path)
				fmt.Println(string(data))
			}
		}

		return nil
	}

	// Compare compose files using string diff
	currentStr := string(currentCompose)
	newStr := string(newCompose)

	hasComposeChanges := currentStr != newStr

	if !hasComposeChanges {
		fmt.Println("✓ No changes detected in docker-compose.yaml")
	} else {
		fmt.Println("📝 Docker Compose Changes:")
		fmt.Println()

		// Show a simple unified diff
		currentLines := strings.Split(currentStr, "\n")
		newLines := strings.Split(newStr, "\n")

		// Simple line-by-line comparison
		maxLen := len(currentLines)
		if len(newLines) > maxLen {
			maxLen = len(newLines)
		}

		for i := 0; i < maxLen; i++ {
			var currentLine, newLine string
			if i < len(currentLines) {
				currentLine = currentLines[i]
			}
			if i < len(newLines) {
				newLine = newLines[i]
			}

			if currentLine != newLine {
				if currentLine != "" {
					fmt.Printf("- %s\n", currentLine)
				}
				if newLine != "" {
					fmt.Printf("+ %s\n", newLine)
				}
			}
		}
		fmt.Println()

		// Extract affected services
		affected := extractAffectedServices(currentCompose, newCompose)
		if len(affected) > 0 {
			fmt.Println("⚠️  Affected Services:")
			for _, svc := range affected {
				fmt.Printf("  • %s\n", svc)
			}
			fmt.Println()
		}
	}

	// Compare files
	added := []string{}
	modified := []string{}
	removed := []string{}

	if currentFiles == nil {
		// No existing files - everything is new
		for path := range newFiles {
			added = append(added, path)
		}
	} else {
		for path, newData := range newFiles {
			oldData, exists := currentFiles[path]
			if !exists {
				added = append(added, path)
			} else if string(oldData) != string(newData) {
				modified = append(modified, path)
			}
		}

		for path := range currentFiles {
			if _, exists := newFiles[path]; !exists {
				removed = append(removed, path)
			}
		}
	}

	hasFileChanges := len(added) > 0 || len(modified) > 0 || len(removed) > 0

	if !hasFileChanges {
		if showFiles {
			fmt.Println("✓ No changes detected in files/")
		}
		return nil
	}

	fmt.Println("📁 File Changes:")
	if len(added) > 0 {
		fmt.Println("  Added:")
		for _, f := range added {
			fmt.Printf("    + %s\n", f)
		}
	}
	if len(removed) > 0 {
		fmt.Println("  Removed:")
		for _, f := range removed {
			fmt.Printf("    - %s\n", f)
		}
	}
	if len(modified) > 0 {
		fmt.Println("  Modified:")
		for _, f := range modified {
			fmt.Printf("    ~ %s\n", f)
		}
	}
	fmt.Println()

	if showFiles && len(modified) > 0 {
		fmt.Println("📄 Detailed File Diffs:")
		for _, filename := range modified {
			fmt.Printf("\n--- a/%s\n", filename)
			fmt.Printf("+++ b/%s\n", filename)

			oldLines := strings.Split(string(currentFiles[filename]), "\n")
			newLines := strings.Split(string(newFiles[filename]), "\n")

			maxLen := len(oldLines)
			if len(newLines) > maxLen {
				maxLen = len(newLines)
			}

			for i := 0; i < maxLen; i++ {
				var oldLine, newLine string
				if i < len(oldLines) {
					oldLine = oldLines[i]
				}
				if i < len(newLines) {
					newLine = newLines[i]
				}

				if oldLine != newLine {
					if oldLine != "" {
						fmt.Printf("- %s\n", oldLine)
					}
					if newLine != "" {
						fmt.Printf("+ %s\n", newLine)
					}
				}
			}
		}
	}

	return nil
}

func extractAffectedServices(oldYAML, newYAML []byte) []string {
	var affected []string

	var oldServices map[string]bool
	if oldYAML == nil {
		oldServices = make(map[string]bool)
	} else {
		oldServices = extractServiceNames(oldYAML)
	}

	newServices := extractServiceNames(newYAML)

	// Services that exist in new but not old (added)
	for svc := range newServices {
		if !oldServices[svc] {
			affected = append(affected, svc)
		}
	}

	// Services that exist in old but not new (removed)
	for svc := range oldServices {
		if !newServices[svc] {
			affected = append(affected, fmt.Sprintf("%s (removed)", svc))
		}
	}

	// Services that exist in both - check if they changed
	for svc := range newServices {
		if oldServices[svc] {
			oldSvcData := extractServiceData(oldYAML, svc)
			newSvcData := extractServiceData(newYAML, svc)
			if oldSvcData != newSvcData {
				affected = append(affected, fmt.Sprintf("%s (modified)", svc))
			}
		}
	}

	return affected
}

func extractServiceNames(composeYAML []byte) map[string]bool {
	services := make(map[string]bool)

	if composeYAML == nil {
		return services
	}

	var doc map[string]any
	if err := yaml.Unmarshal(composeYAML, &doc); err != nil {
		return services
	}

	if svcs, ok := doc["services"].(map[string]any); ok {
		for name := range svcs {
			services[name] = true
		}
	}

	return services
}

func extractServiceData(composeYAML []byte, serviceName string) string {
	if composeYAML == nil {
		return ""
	}

	var doc map[string]any
	if err := yaml.Unmarshal(composeYAML, &doc); err != nil {
		return ""
	}

	svcs, ok := doc["services"].(map[string]any)
	if !ok {
		return ""
	}

	svcData, ok := svcs[serviceName]
	if !ok {
		return ""
	}

	data, err := yaml.Marshal(svcData)
	if err != nil {
		return ""
	}

	return strings.TrimSpace(string(data))
}
