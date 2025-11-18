package chart

import (
	"context"

	"composepack/internal/util/fileloader"
)

// Well-known directories/files for chart layouts.
const (
	MetadataFile       = "Chart.yaml"
	ValuesFile         = "values.yaml"
	ValuesSchemaFile   = "values.schema.json"
	TemplatesCompose   = "templates/compose"
	TemplatesFiles     = "templates/files"
	TemplatesHelpers   = "templates/helpers"
	FilesDir           = "files"
	TemplateFileSuffix = ".tpl"
)

// Loader describes chart loading behavior regardless of source (dir, archive, registry).
type Loader interface {
	Load(ctx context.Context, source string) (*Chart, error)
}

// LoaderFunc allows simple function-based implementations of Loader.
type LoaderFunc func(ctx context.Context, source string) (*Chart, error)

// Load implements Loader.
func (f LoaderFunc) Load(ctx context.Context, source string) (*Chart, error) {
	return f(ctx, source)
}

// ChartMetadata mirrors Helm-style metadata fields.
type ChartMetadata struct {
	Name        string   `yaml:"name"`
	Version     string   `yaml:"version"`
	Description string   `yaml:"description,omitempty"`
	Maintainers []string `yaml:"maintainers,omitempty"`
}

// Chart captures a fully loaded chart from disk/archive.
type Chart struct {
	Metadata      ChartMetadata
	BaseDir       string
	Values        map[string]any
	ValuesSchema  []byte
	ComposeTpls   map[string]string // templates/compose/*.tpl.yaml (rendered to Compose YAML)
	FileTemplates map[string]string // templates/files/**/*.tpl (rendered to runtime files)
	HelperTpls    map[string]string // templates/helpers/**/*.tpl (include-only snippets)
	StaticFiles   map[string][]byte // files/**/* (non-templated assets copied verbatim)
}

// LoadFromDirectory is a convenience wrapper around the filesystem loader.
func LoadFromDirectory(ctx context.Context, path string) (*Chart, error) {
	return NewFileSystemChartLoader(fileloader.NewFileSystemLoader()).Load(ctx, path)
}
