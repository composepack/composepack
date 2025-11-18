package chart

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// CompositeLoader delegates to filesystem or archive loader based on source path.
type CompositeLoader struct {
	fs *FileSystemChartLoader
}

// NewCompositeLoader builds a loader that supports directories and archives.
func NewCompositeLoader(fsLoader *FileSystemChartLoader) *CompositeLoader {
	return &CompositeLoader{fs: fsLoader}
}

// Load inspects the source and loads from tar/tgz archives or directories.
func (l *CompositeLoader) Load(ctx context.Context, source string) (*Chart, error) {
	if source == "" {
		return nil, fmt.Errorf("chart source must be provided")
	}
	resolved := source
	cleanup := func() {}
	if isURL(source) {
		var err error
		resolved, cleanup, err = l.downloadIfURL(ctx, source)
		if err != nil {
			return nil, err
		}
		defer cleanup()
		source = resolved
	}

	if info, err := os.Stat(source); err == nil {
		if info.IsDir() {
			return l.fs.Load(ctx, source)
		}
		if looksLikeArchive(source) {
			return l.loadArchive(ctx, source)
		}
		return nil, fmt.Errorf("chart source %q is not a directory", source)
	} else if !os.IsNotExist(err) {
		return nil, err
	}

	if looksLikeArchive(source) {
		return l.loadArchive(ctx, source)
	}

	return nil, fmt.Errorf("chart source %q not found", source)
}

func (l *CompositeLoader) loadArchive(ctx context.Context, source string) (*Chart, error) {
	tmpDir, err := os.MkdirTemp("", "composepack-chart-*")
	if err != nil {
		return nil, fmt.Errorf("create temp dir: %w", err)
	}
	defer os.RemoveAll(tmpDir)
	if err := extractArchive(source, tmpDir); err != nil {
		return nil, err
	}
	root, err := findChartRoot(tmpDir)
	if err != nil {
		return nil, err
	}
	return l.fs.Load(ctx, root)
}

func looksLikeArchive(path string) bool {
	lower := strings.ToLower(path)
	return strings.HasSuffix(lower, ".tar") ||
		strings.HasSuffix(lower, ".tar.gz") ||
		strings.HasSuffix(lower, ".tgz") ||
		strings.HasSuffix(lower, ".cpack") ||
		strings.HasSuffix(lower, ".cpack.tgz")
}

func extractArchive(path, dest string) error {
	file, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("open archive: %w", err)
	}
	defer file.Close()

	var reader io.Reader = file
	if strings.HasSuffix(strings.ToLower(path), ".gz") || strings.HasSuffix(strings.ToLower(path), ".tgz") {
		gz, err := gzip.NewReader(file)
		if err != nil {
			return fmt.Errorf("create gzip reader: %w", err)
		}
		defer gz.Close()
		reader = gz
	}

	tr := tar.NewReader(reader)
	for {
		header, err := tr.Next()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return fmt.Errorf("read archive: %w", err)
		}

		name := header.Name
		if shouldSkipArchiveEntry(name) {
			continue
		}
		target := filepath.Join(dest, name)
		switch header.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(target, os.FileMode(header.Mode)); err != nil {
				return fmt.Errorf("create dir %s: %w", target, err)
			}
		case tar.TypeReg:
			if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
				return fmt.Errorf("create file dir: %w", err)
			}
			out, err := os.OpenFile(target, os.O_CREATE|os.O_RDWR|os.O_TRUNC, os.FileMode(header.Mode))
			if err != nil {
				return fmt.Errorf("create file %s: %w", target, err)
			}
			if _, err := io.Copy(out, tr); err != nil {
				out.Close()
				return fmt.Errorf("write file %s: %w", target, err)
			}
			out.Close()
		}
	}

	return nil
}

func shouldSkipArchiveEntry(name string) bool {
	base := filepath.Base(name)
	if strings.HasPrefix(base, "._") || base == ".DS_Store" {
		return true
	}
	if strings.HasPrefix(name, "__MACOSX/") {
		return true
	}
	return false
}

func findChartRoot(base string) (string, error) {
	var chartDir string
	err := filepath.WalkDir(base, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, relErr := filepath.Rel(base, path)
		if relErr != nil {
			return relErr
		}
		if rel == "." {
			rel = ""
		}
		if d.IsDir() && shouldSkipArchiveEntry(rel) {
			return filepath.SkipDir
		}
		if !d.IsDir() && strings.EqualFold(d.Name(), MetadataFile) {
			chartDir = filepath.Dir(path)
			return io.EOF
		}
		return nil
	})
	if err != nil && err != io.EOF {
		return "", err
	}
	if chartDir == "" {
		return "", fmt.Errorf("chart archive missing %s", MetadataFile)
	}
	return chartDir, nil
}
