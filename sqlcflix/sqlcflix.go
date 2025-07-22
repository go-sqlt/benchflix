package sqlcflix

import (
	"context"

	"github.com/go-sqlt/benchflix"
)

type Repository struct {
	Queries *Queries
}

func (r Repository) QueryList(ctx context.Context, params benchflix.ListParams) ([]benchflix.Movie, error) {
	rows, err := r.Queries.Query(ctx, QueryParams(params))
	if err != nil {
		return nil, err
	}

	movies := make([]benchflix.Movie, len(rows))

	for i, row := range rows {
		movies[i] = benchflix.Movie(row)
	}

	return movies, nil
}

func (r Repository) QueryListPreload(ctx context.Context, params benchflix.ListParams) ([]benchflix.Movie, error) {
	rows, err := r.Queries.QueryPreload(ctx, QueryPreloadParams(params))
	if err != nil {
		return nil, err
	}

	var (
		movies = make([]benchflix.Movie, len(rows))
		ids    = make([]int64, len(rows))
		idMap  = make(map[int64]int, len(rows))
	)

	for i, row := range rows {
		movies[i] = benchflix.Movie{
			ID:      row.ID,
			Title:   row.Title,
			AddedAt: row.AddedAt,
			Rating:  row.Rating,
		}

		ids[i] = row.ID
		idMap[row.ID] = i
	}

	if len(movies) == 0 {
		return movies, nil
	}

	movieDirectors, err := r.Queries.QueryDirectors(ctx, ids)
	if err != nil {
		return nil, err
	}

	for _, md := range movieDirectors {
		movies[idMap[md.MovieID]].Directors = md.Directors
	}

	return movies, nil
}

func (r Repository) QueryDashboard(ctx context.Context, params benchflix.DashboardParams) ([]benchflix.Movie, error) {
	return nil, benchflix.ErrSkip
}

func (r Repository) QueryDashboardPreload(ctx context.Context, params benchflix.DashboardParams) ([]benchflix.Movie, error) {
	return nil, benchflix.ErrSkip
}
