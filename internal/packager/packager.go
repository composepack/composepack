package packager

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
	"time"

	"composepack/internal/core/chart"
)

// Options controls packaging behavior.
type Options struct {
	ChartPath   string
	Destination string
	OutputName  string
	Force       bool
}

// PackageChart produces a .cpack.tgz archive containing the chart source.
func PackageChart(ctx context.Context, loader chart.Loader, opts Options) (string, error) {
	if loader == nil {
		return nilPathErr("loader is required")
	}
	if opts.ChartPath == "" {
		return nilPathErr("chart path is required")
	}
	chartPath, err := filepath.Abs(opts.ChartPath)
	if err != nil {
		return "", fmt.Errorf("resolve chart path: %w", err)
	}

	ch, err := loader.Load(ctx, chartPath)
	if err != nil {
		return "", fmt.Errorf("load chart: %w", err)
	}

	destDir := opts.Destination
	if destDir == "" {
		destDir = "."
	}
	if err := os.MkdirAll(destDir, 0o755); err != nil {
		return "", fmt.Errorf("ensure destination: %w", err)
	}

	filename := opts.OutputName
	if filename == "" {
		filename = fmt.Sprintf("%s-%s.cpack.tgz", ch.Metadata.Name, ch.Metadata.Version)
	}

	outputPath := filepath.Join(destDir, filename)
	if !opts.Force {
		if _, err := os.Stat(outputPath); err == nil {
			return "", fmt.Errorf("output file %s already exists (use --force to overwrite)", outputPath)
		}
	}

	file, err := os.Create(outputPath)
	if err != nil {
		return "", fmt.Errorf("create output file: %w", err)
	}
	defer file.Close()

	gz := gzip.NewWriter(file)
	gz.Name = filename
	gz.ModTime = time.Now()
	defer gz.Close()

	tw := tar.NewWriter(gz)
	defer tw.Close()

	if err := filepath.WalkDir(chartPath, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(chartPath, path)
		if err != nil {
			return err
		}
		if rel == "." {
			return nil
		}
		if shouldSkip(rel) {
			if d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		info, err := d.Info()
		if err != nil {
			return err
		}
		header, err := tar.FileInfoHeader(info, "")
		if err != nil {
			return err
		}
		header.Name = filepath.ToSlash(rel)
		if err := tw.WriteHeader(header); err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		f, err := os.Open(path)
		if err != nil {
			return err
		}
		if _, err := io.Copy(tw, f); err != nil {
			f.Close()
			return err
		}
		return f.Close()
	}); err != nil {
		return "", fmt.Errorf("archive chart: %w", err)
	}

	if err := tw.Close(); err != nil {
		return "", err
	}
	if err := gz.Close(); err != nil {
		return "", err
	}
	if err := file.Close(); err != nil {
		return "", err
	}

	return outputPath, nil
}

func shouldSkip(rel string) bool {
	parts := strings.Split(rel, string(os.PathSeparator))
	for _, part := range parts {
		if part == ".git" || part == ".DS_Store" || strings.HasPrefix(part, "._") || strings.HasPrefix(part, "__MACOSX") {
			return true
		}
	}
	return false
}

func nilPathErr(msg string) (string, error) {
	return "", errors.New(msg)
}
