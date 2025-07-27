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

	return Repository{
		Pool: pool,
		QueryListStatement: sqlt.AllPgx[benchflix.ListParams, benchflix.Movie](
			config,
			sqlt.Parse(`
				SELECT
					m.id                    {{ Scan.Int.To "ID" }}
					, m.title               {{ Scan.String.To "Title" }}
					, m.added_at            {{ Scan.Time.To "AddedAt" }}
					, m.rating              {{ Scan.Float.To "Rating" }}
					, d.directors           {{ Scan.StringSlice.To "Directors" }}
				FROM movies m
				LEFT JOIN LATERAL (
					SELECT ARRAY_AGG(p.name ORDER BY p.name) AS directors
					FROM movie_directors md
					JOIN people p ON p.id = md.person_id
					WHERE md.movie_id = m.id
				) d ON true
				WHERE
					(
						{{ .Search }} = ''
						OR to_tsvector('simple', m.title) @@ plainto_tsquery('simple', {{ .Search }})
						OR EXISTS (
							SELECT 1
							FROM movie_directors md
							JOIN people p ON p.id = md.person_id
							WHERE md.movie_id = m.id
							AND to_tsvector('simple', p.name) @@ plainto_tsquery('simple', {{ .Search }})
						)
					)
					AND ({{ .YearAdded }} = 0 OR EXTRACT(YEAR FROM m.added_at) = {{ .YearAdded }})
					AND ({{ .MinRating }} = 0 OR m.rating >= {{ .MinRating }})
				ORDER BY m.rating DESC
				LIMIT CASE WHEN {{ .Limit }} BETWEEN 1 AND 1000 THEN {{ .Limit }} ELSE 1000 END;
			`),
		),
		QueryListPreloadStatement: sqlt.AllPgx[benchflix.ListParams, benchflix.Movie](
			config,
			sqlt.Parse(`
				SELECT
					m.id                    {{ Scan.Int.To "ID" }}
					, m.title               {{ Scan.String.To "Title" }}
					, m.added_at            {{ Scan.Time.To "AddedAt" }}
					, m.rating              {{ Scan.Float.To "Rating" }}
				FROM movies m
				WHERE
					(
						{{ .Search }} = ''
						OR to_tsvector('simple', m.title) @@ plainto_tsquery('simple', {{ .Search }})
						OR EXISTS (
							SELECT 1
							FROM movie_directors md
							JOIN people p ON p.id = md.person_id
							WHERE md.movie_id = m.id
							AND to_tsvector('simple', p.name) @@ plainto_tsquery('simple', {{ .Search }})
						)
					)
					AND ({{ .YearAdded }} = 0 OR EXTRACT(YEAR FROM m.added_at) = {{ .YearAdded }})
					AND ({{ .MinRating }} = 0 OR m.rating >= {{ .MinRating }})
				ORDER BY m.rating DESC
				LIMIT CASE WHEN {{ .Limit }} BETWEEN 1 AND 1000 THEN {{ .Limit }} ELSE 1000 END;
			`),
		),
		QueryDirectorsStatement: sqlt.AllPgx[[]int64, MovieDirectors](
			config,
			sqlt.Parse(`
				SELECT
					md.movie_id			{{ Scan.Int.To "MovieID" }}
					, ARRAY_AGG(people.name ORDER BY people.name) 
						AS directors 	{{ Scan.StringSlice.To "Directors" }}
				FROM movie_directors md
				JOIN people ON people.id = md.person_id
				WHERE md.movie_id = ANY ({{ . }})
				GROUP BY md.movie_id;
			`),
		),
		QueryDashboardStatement: sqlt.AllPgx[benchflix.DashboardParams, benchflix.Movie](
			config,
			sqlt.Parse(`
				SELECT
					m.id                    {{ Scan.Int.To "ID" }}
					, m.title               {{ Scan.String.To "Title" }}
					, m.added_at            {{ Scan.Time.To "AddedAt" }}
					, m.rating              {{ Scan.Float.To "Rating" }}
					{{ if .WithDirectors }}
						, d.directors       {{ Scan.StringSlice.To "Directors" }}
					{{ end }}
				FROM movies m
				{{ if .WithDirectors }}
					LEFT JOIN LATERAL (
						SELECT ARRAY_AGG(p.name ORDER BY p.name) AS directors
						FROM movie_directors md
						JOIN people p ON p.id = md.person_id
						WHERE md.movie_id = m.id
					) d ON true
				{{ end }}
				WHERE 1=1
				{{ if .Search }}
					AND (
						to_tsvector('simple', m.title) @@ plainto_tsquery('simple', {{ .Search }})
						OR EXISTS (
							SELECT 1
							FROM movie_directors md
							JOIN people p ON p.id = md.person_id
							WHERE md.movie_id = m.id
							AND to_tsvector('simple', p.name) @@ plainto_tsquery('simple', {{ .Search }})
						)
					)
				{{ end }}
				{{ if .YearAdded }} AND EXTRACT(YEAR FROM m.added_at) = {{ .YearAdded }}{{ end }}
				{{ if .MinRating }} AND m.rating >= {{ .MinRating }} {{ end }}
				ORDER BY
				{{ if eq .Sort "title" }} m.title
					{{ else if eq .Sort "added_at" }} m.added_at
					{{ else }} m.rating
				{{ end }} 
				{{ if .Desc }} DESC{{ else }} ASC{{ end }}
				{{ if and (gt .Limit 0) (lt .Limit 1000) }} LIMIT {{ .Limit }}{{ else }} LIMIT 1000{{ end }}
			`),
		),
		QueryDashboardPreloadStatement: sqlt.AllPgx[benchflix.DashboardParams, benchflix.Movie](
			config,
			sqlt.Parse(`
				SELECT
					m.id                    {{ Scan.Int.To "ID" }}
					, m.title               {{ Scan.String.To "Title" }}
					, m.added_at            {{ Scan.Time.To "AddedAt" }}
					, m.rating              {{ Scan.Float.To "Rating" }}
				FROM movies m
				WHERE 1=1
				{{ if .Search }}
					AND (
						to_tsvector('simple', m.title) @@ plainto_tsquery('simple', {{ .Search }})
						OR EXISTS (
							SELECT 1
							FROM movie_directors md
							JOIN people p ON p.id = md.person_id
							WHERE md.movie_id = m.id
							AND to_tsvector('simple', p.name) @@ plainto_tsquery('simple', {{ .Search }})
						)
					)
				{{ end }}
				{{ if .YearAdded }} AND EXTRACT(YEAR FROM m.added_at) = {{ .YearAdded }}{{ end }}
				{{ if .MinRating }} AND m.rating >= {{ .MinRating }}{{ end }}
				ORDER BY
				{{ if eq .Sort "title" }} m.title
					{{ else if eq .Sort "added_at" }} m.added_at
					{{ else }} m.rating
				{{ end }}  	 
				{{ if .Desc }} DESC{{ else }} ASC {{ end }}
				{{ if and (gt .Limit 0) (lt .Limit 1000) }} LIMIT {{ .Limit }}{{ else }} LIMIT 1000{{ end }}
			`),
		),
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
