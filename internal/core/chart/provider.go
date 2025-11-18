package chart

import (
	"github.com/google/wire"

	"composepack/internal/util/fileloader"
)

// ProviderSet exposes the chart loader wiring for DI.
var ProviderSet = wire.NewSet(
	fileloader.NewFileSystemLoader,
	NewFileSystemChartLoader,
	NewCompositeLoader,
	wire.Bind(new(Loader), new(*CompositeLoader)),
)
