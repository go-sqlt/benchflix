package sqltflix

import (
	"context"
	"time"

	"github.com/go-sqlt/benchflix"
	"github.com/go-sqlt/sqlt"
	"github.com/jackc/pgx/v5/pgxpool"
)

type MovieDirectors struct {
	MovieID   int64
	Directors []string
}

func NewRepository(conn string, min, max int, idle time.Duration, config sqlt.Config) Repository {
	cfg := benchflix.Must(pgxpool.ParseConfig(conn))

	cfg.MaxConns = int32(max)
	cfg.MinConns = int32(min)
	cfg.MaxConnIdleTime = idle

	pool := benchflix.Must(pgxpool.NewWithConfig(context.Background(), cfg))

	config = config.With(sqlt.ParseFiles("sqltflix/queries.go.tpl"))

	return Repository{
		Pool:                           pool,
		QueryListStatement:             sqlt.AllPgx[benchflix.ListParams, benchflix.Movie](config, sqlt.Lookup("query_list")),
		QueryListPreloadStatement:      sqlt.AllPgx[benchflix.ListParams, benchflix.Movie](config, sqlt.Lookup("query_list_preload")),
		QueryDirectorsStatement:        sqlt.AllPgx[[]int64, MovieDirectors](config, sqlt.Lookup("query_directors")),
		QueryDashboardStatement:        sqlt.AllPgx[benchflix.DashboardParams, benchflix.Movie](config, sqlt.Lookup("query_dashboard")),
		QueryDashboardPreloadStatement: sqlt.AllPgx[benchflix.DashboardParams, benchflix.Movie](config, sqlt.Lookup("query_dashboard_preload")),
	}
}

type Repository struct {
	Pool                           *pgxpool.Pool
	QueryListStatement             sqlt.PgxStatement[benchflix.ListParams, []benchflix.Movie]
	QueryListPreloadStatement      sqlt.PgxStatement[benchflix.ListParams, []benchflix.Movie]
	QueryDirectorsStatement        sqlt.PgxStatement[[]int64, []MovieDirectors]
	QueryDashboardStatement        sqlt.PgxStatement[benchflix.DashboardParams, []benchflix.Movie]
	QueryDashboardPreloadStatement sqlt.PgxStatement[benchflix.DashboardParams, []benchflix.Movie]
}

func (r Repository) QueryList(ctx context.Context, params benchflix.ListParams) ([]benchflix.Movie, error) {
	return r.QueryListStatement.Exec(ctx, r.Pool, params)
}

func (r Repository) QueryListPreload(ctx context.Context, params benchflix.ListParams) ([]benchflix.Movie, error) {
	movies, err := r.QueryListPreloadStatement.Exec(ctx, r.Pool, params)
	if err != nil {
		return nil, err
	}

	if len(movies) == 0 {
		return movies, nil
	}

	var (
		ids   = make([]int64, len(movies))
		idMap = make(map[int64]int, len(movies))
	)

	for i, m := range movies {
		ids[i] = m.ID
		idMap[m.ID] = i
	}

	movieDirectors, err := r.QueryDirectorsStatement.Exec(ctx, r.Pool, ids)
	if err != nil {
		return nil, err
	}

	for _, md := range movieDirectors {
		movies[idMap[md.MovieID]].Directors = md.Directors
	}

	return movies, nil
}

func (r Repository) QueryDashboard(ctx context.Context, params benchflix.DashboardParams) ([]benchflix.Movie, error) {
	return r.QueryDashboardStatement.Exec(ctx, r.Pool, params)
}

func (r Repository) QueryDashboardPreload(ctx context.Context, params benchflix.DashboardParams) ([]benchflix.Movie, error) {
	movies, err := r.QueryDashboardPreloadStatement.Exec(ctx, r.Pool, params)
	if err != nil {
		return nil, err
	}

	if len(movies) == 0 || !params.WithDirectors {
		return movies, nil
	}

	var (
		ids   = make([]int64, len(movies))
		idMap = make(map[int64]int, len(movies))
	)

	for i, m := range movies {
		ids[i] = m.ID
		idMap[m.ID] = i
	}

	movieDirectors, err := r.QueryDirectorsStatement.Exec(ctx, r.Pool, ids)
	if err != nil {
		return nil, err
	}

	for _, md := range movieDirectors {
		movies[idMap[md.MovieID]].Directors = md.Directors
	}

	return movies, nil
}
