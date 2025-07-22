package benchflix

import (
	"context"
	"errors"
	"time"
)

var ErrSkip = errors.New("skip")

type Movie struct {
	ID        int64
	Title     string
	AddedAt   time.Time `db:"added_at"`
	Rating    float64
	Directors []string
}

type ListParams struct {
	Search    string
	YearAdded int64   `db:"year_added"`
	MinRating float64 `db:"min_rating"`
	Limit     uint64
}

type DashboardParams struct {
	Search        string
	YearAdded     int64   `db:"year_added"`
	MinRating     float64 `db:"min_rating"`
	Limit         uint64
	Sort          string
	Desc          bool
	WithDirectors bool
}

type Repository interface {
	QueryList(ctx context.Context, params ListParams) ([]Movie, error)
	QueryListPreload(ctx context.Context, params ListParams) ([]Movie, error)
	QueryDashboard(ctx context.Context, params DashboardParams) ([]Movie, error)
	QueryDashboardPreload(ctx context.Context, params DashboardParams) ([]Movie, error)
}
