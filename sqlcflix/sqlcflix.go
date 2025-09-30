package sqlcflix

import (
	"context"
	"time"

	"github.com/go-sqlt/benchflix"
	"github.com/jackc/pgx/v5/pgxpool"
)

func NewRepository(conn string, min, max int, idle time.Duration) benchflix.Repository {
	cfg := benchflix.Must(pgxpool.ParseConfig(conn))

	cfg.MaxConns = int32(max)
	cfg.MinConns = int32(min)
	cfg.MaxConnIdleTime = idle

	pool := benchflix.Must(pgxpool.NewWithConfig(context.Background(), cfg))

	return Repository{
		Queries: New(pool),
	}
}

type Repository struct {
	Queries *Queries
}

func (r Repository) QueryList(ctx context.Context, params benchflix.ListParams) ([]benchflix.Movie, error) {
	rows, err := r.Queries.queryList(ctx, queryListParams(params))
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
	rows, err := r.Queries.queryListPreload(ctx, queryListPreloadParams(params))
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

	movieDirectors, err := r.Queries.queryDirectors(ctx, ids)
	if err != nil {
		return nil, err
	}

	for _, md := range movieDirectors {
		movies[idMap[md.MovieID]].Directors = md.Directors
	}

	return movies, nil
}

func (r Repository) QueryDashboard(ctx context.Context, params benchflix.DashboardParams) ([]benchflix.Movie, error) {
	if params.WithDirectors {
		rows, err := r.Queries.queryDashboardWithDirectors(ctx, queryDashboardWithDirectorsParams{
			Search:    params.Search,
			YearAdded: params.YearAdded,
			MinRating: params.MinRating,
			Limit:     params.Limit,
			Sort:      params.Sort,
			Desc:      params.Desc,
		})
		if err != nil {
			return nil, err
		}

		movies := make([]benchflix.Movie, len(rows))

		for i, row := range rows {
			movies[i] = benchflix.Movie(row)
		}

		return movies, nil
	}

	rows, err := r.Queries.queryDashboardWithoutDirectors(ctx, queryDashboardWithoutDirectorsParams{
		Search:    params.Search,
		YearAdded: params.YearAdded,
		MinRating: params.MinRating,
		Limit:     params.Limit,
		Sort:      params.Sort,
		Desc:      params.Desc,
	})
	if err != nil {
		return nil, err
	}

	movies := make([]benchflix.Movie, len(rows))

	for i, row := range rows {
		movies[i] = benchflix.Movie{
			ID:      row.ID,
			Title:   row.Title,
			AddedAt: row.AddedAt,
			Rating:  row.Rating,
		}
	}

	return movies, nil
}

func (r Repository) QueryDashboardPreload(ctx context.Context, params benchflix.DashboardParams) ([]benchflix.Movie, error) {
	rows, err := r.Queries.queryDashboardPreload(ctx, queryDashboardPreloadParams{
		Search:    params.Search,
		YearAdded: params.YearAdded,
		MinRating: params.MinRating,
		Limit:     params.Limit,
		Sort:      params.Sort,
		Desc:      params.Desc,
	})
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

	if len(movies) == 0 || !params.WithDirectors {
		return movies, nil
	}

	movieDirectors, err := r.Queries.queryDirectors(ctx, ids)
	if err != nil {
		return nil, err
	}

	for _, md := range movieDirectors {
		movies[idMap[md.MovieID]].Directors = md.Directors
	}

	return movies, nil
}
