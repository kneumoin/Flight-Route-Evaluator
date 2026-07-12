package provider

import (
	"context"

	"github.com/kneumoin/nepal/internal/model"
)

// CoverageAnalyzer reports cached-price coverage for a route/month.
type CoverageAnalyzer interface {
	AnalyzeLegCoverage(ctx context.Context, q model.Query) (model.RouteCoverageRow, error)
}
