package pgxflix

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/go-sqlt/benchflix"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	queryList = `
		SELECT m.id, m.title, m.added_at, m.rating, d.directors::TEXT[] AS directors
		FROM movies m
		LEFT JOIN LATERAL (
			SELECT ARRAY_AGG(p.name ORDER BY p.name) AS directors
			FROM movie_directors md
			JOIN people p ON p.id = md.person_id
			WHERE md.movie_id = m.id
		) d ON true
		WHERE
			(
				$1::TEXT = ''
				OR to_tsvector('simple', m.title) @@ plainto_tsquery('simple', $1)
				OR EXISTS (
					SELECT 1
					FROM movie_directors md
					JOIN people p ON p.id = md.person_id
					WHERE md.movie_id = m.id
					AND to_tsvector('simple', p.name) @@ plainto_tsquery('simple', $1)
				)
			)
			AND ($2::INT8 = 0 OR EXTRACT(YEAR FROM m.added_at) = $2)
			AND ($3::FLOAT8 = 0 OR m.rating >= $3)
		ORDER BY m.rating DESC
		LIMIT CASE WHEN $4::INT4 BETWEEN 1 AND 1000 THEN $4 ELSE 1000 END
	`

	queryListPreload = `
		SELECT id, title, added_at, rating
		FROM movies m
		WHERE
			(
				$1::TEXT = ''
				OR to_tsvector('simple', title) @@ plainto_tsquery('simple', $1)
				OR EXISTS (
					SELECT 1
					FROM movie_directors md
					JOIN people p ON p.id = md.person_id
					WHERE md.movie_id = id
					AND to_tsvector('simple', p.name) @@ plainto_tsquery('simple', $1)
				)
			)
			AND ($2::INT8 = 0 OR EXTRACT(YEAR FROM added_at) = $2)
			AND ($3::FLOAT8 = 0 OR rating >= $3)
		ORDER BY rating DESC
		LIMIT CASE WHEN $4::INT4 BETWEEN 1 AND 1000 THEN $4 ELSE 1000 END
	`

	queryDirectors = `
		SELECT md.movie_id, ARRAY_AGG(people.name ORDER BY people.name)::TEXT[] AS directors
		FROM movie_directors md
		JOIN people ON people.id = md.person_id
		WHERE md.movie_id = ANY ($1::INT8[])
		GROUP BY md.movie_id
	`
)

func NewRepository(conn string, min, max int, idle time.Duration) benchflix.Repository {
	cfg := benchflix.Must(pgxpool.ParseConfig(conn))

	cfg.MaxConns = int32(max)
	cfg.MinConns = int32(min)
	cfg.MaxConnIdleTime = idle

	return Repository{
		Pool: benchflix.Must(pgxpool.NewWithConfig(context.Background(), cfg)),
	}
}

type Repository struct {
	Pool *pgxpool.Pool
}

func (r Repository) QueryList(ctx context.Context, params benchflix.ListParams) ([]benchflix.Movie, error) {
	rows, err := r.Pool.Query(ctx, queryList, params.Search, params.YearAdded, params.MinRating, params.Limit)
	if err != nil {
		return nil, err
	}

	defer rows.Close()

	var movies = make([]benchflix.Movie, 0, params.Limit)

	for rows.Next() {
		var m benchflix.Movie

		if err := rows.Scan(&m.ID, &m.Title, &m.AddedAt, &m.Rating, &m.Directors); err != nil {
			return nil, err
		}

		movies = append(movies, m)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return movies, nil
}

func (r Repository) QueryListPreload(ctx context.Context, params benchflix.ListParams) ([]benchflix.Movie, error) {
	var (
		movies = make([]benchflix.Movie, 0, params.Limit)
		ids    = make([]int64, 0, params.Limit)
		idMap  = make(map[int64]int, params.Limit)
	)

	rows, err := r.Pool.Query(ctx, queryListPreload, params.Search, params.YearAdded, params.MinRating, params.Limit)
	if err != nil {
		return nil, err
	}

	defer rows.Close()

	for rows.Next() {
		var movie benchflix.Movie

		if err := rows.Scan(&movie.ID, &movie.Title, &movie.AddedAt, &movie.Rating); err != nil {
			return nil, err
		}

		idMap[movie.ID] = len(ids)
		ids = append(ids, movie.ID)
		movies = append(movies, movie)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	if len(movies) == 0 {
		return movies, nil
	}

	rows.Close()

	dirRows, err := r.Pool.Query(ctx, queryDirectors, ids)
	if err != nil {
		return nil, err
	}

	defer dirRows.Close()

	for dirRows.Next() {
		var (
			movieID   int64
			directors []string
		)

		if err := dirRows.Scan(&movieID, &directors); err != nil {
			return nil, err
		}

		movies[idMap[movieID]].Directors = directors
	}

	if err = dirRows.Err(); err != nil {
		return nil, err
	}

	return movies, nil
}

func (r Repository) QueryDashboard(ctx context.Context, params benchflix.DashboardParams) ([]benchflix.Movie, error) {
	var (
		movies = make([]benchflix.Movie, 0, params.Limit)
		sb     = &strings.Builder{}
	)

	sb.WriteString("SELECT m.id, m.title, m.added_at, m.rating")

	if params.WithDirectors {
		sb.WriteString(", d.directors::TEXT[] AS directors")
	}

	sb.WriteString(" FROM movies m")

	if params.WithDirectors {
		sb.WriteString(` LEFT JOIN LATERAL (
			SELECT ARRAY_AGG(p.name ORDER BY p.name) AS directors
			FROM movie_directors md
			JOIN people p ON p.id = md.person_id
			WHERE md.movie_id = m.id
		) d ON true`)
	}

	sb.WriteString(" WHERE 1=1")

	if params.Search != "" {
		sb.WriteString(` AND (
			to_tsvector('simple', m.title) @@ plainto_tsquery('simple', @search)
			OR EXISTS (
				SELECT 1 FROM movie_directors md
				JOIN people p ON p.id = md.person_id
				WHERE md.movie_id = m.id
				AND to_tsvector('simple', p.name) @@ plainto_tsquery('simple', @search)
			)
		)`)
	}

	if params.YearAdded != 0 {
		sb.WriteString(" AND EXTRACT(YEAR FROM m.added_at) = @year_added")
	}
	
	if params.MinRating != 0 {
		sb.WriteString(" AND m.rating >= @min_rating")
	}

	order := "ASC"

	if params.Desc {
		order = "DESC"
	}

	switch params.Sort {
	case "rating":
		fmt.Fprintf(sb, " ORDER BY m.rating %s", order)
	case "title":
		fmt.Fprintf(sb, " ORDER BY m.title %s", order)
	case "added_at":
		fmt.Fprintf(sb, " ORDER BY m.added_at %s", order)
	default:
		return nil, fmt.Errorf("invalid sort")
	}

	if params.Limit < 1 || params.Limit > 1000 {
		sb.WriteString(" LIMIT 1000")
	} else {
		fmt.Fprintf(sb, " LIMIT %d", params.Limit)
	}

	rows, err := r.Pool.Query(ctx, sb.String(), pgx.NamedArgs{
		"search":     params.Search,
		"year_added": params.YearAdded,
		"min_rating": params.MinRating,
	})
	if err != nil {
		return nil, err
	}

	defer rows.Close()

	for rows.Next() {
		var m benchflix.Movie

		if params.WithDirectors {
			if err := rows.Scan(&m.ID, &m.Title, &m.AddedAt, &m.Rating, &m.Directors); err != nil {
				return nil, err
			}
		} else {
			if err := rows.Scan(&m.ID, &m.Title, &m.AddedAt, &m.Rating); err != nil {
				return nil, err
			}
		}

		movies = append(movies, m)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return movies, nil
}

func (r Repository) QueryDashboardPreload(ctx context.Context, params benchflix.DashboardParams) ([]benchflix.Movie, error) {
	var (
		movies = make([]benchflix.Movie, 0, params.Limit)
		ids    = make([]int64, 0, params.Limit)
		idMap  = make(map[int64]int, params.Limit)
		sb     = &strings.Builder{}
	)

	sb.WriteString(`
		SELECT m.id, m.title, m.added_at, m.rating
		FROM movies m
		WHERE 1=1`)

	if params.Search != "" {
		sb.WriteString(` AND (
			to_tsvector('simple', m.title) @@ plainto_tsquery('simple', @search)
			OR EXISTS (
				SELECT 1 FROM movie_directors md
				JOIN people p ON p.id = md.person_id
				WHERE md.movie_id = m.id
				AND to_tsvector('simple', p.name) @@ plainto_tsquery('simple', @search)
			)
		)`)
	}
	if params.YearAdded != 0 {
		sb.WriteString(" AND EXTRACT(YEAR FROM m.added_at) = @year_added")
	}
	if params.MinRating != 0 {
		sb.WriteString(" AND m.rating >= @min_rating")
	}

	order := "ASC"

	if params.Desc {
		order = "DESC"
	}

	switch params.Sort {
	case "rating":
		fmt.Fprintf(sb, " ORDER BY m.rating %s", order)
	case "title":
		fmt.Fprintf(sb, " ORDER BY m.title %s", order)
	case "added_at":
		fmt.Fprintf(sb, " ORDER BY m.added_at %s", order)
	default:
		return nil, fmt.Errorf("invalid sort")
	}

	if params.Limit < 1 || params.Limit > 1000 {
		sb.WriteString(" LIMIT 1000")
	} else {
		fmt.Fprintf(sb, " LIMIT %d", params.Limit)
	}

	rows, err := r.Pool.Query(ctx, sb.String(), pgx.NamedArgs{
		"search":     params.Search,
		"year_added": params.YearAdded,
		"min_rating": params.MinRating,
	})
	if err != nil {
		return nil, err
	}

	defer rows.Close()

	for rows.Next() {
		var movie benchflix.Movie

		if err := rows.Scan(&movie.ID, &movie.Title, &movie.AddedAt, &movie.Rating); err != nil {
			return nil, err
		}

		if params.WithDirectors {
			idMap[movie.ID] = len(ids)
			ids = append(ids, movie.ID)
		}

		movies = append(movies, movie)
	}

	if !params.WithDirectors {
		return movies, nil
	}

	rows.Close()

	dirRows, err := r.Pool.Query(ctx, queryDirectors, ids)
	if err != nil {
		return nil, err
	}

	defer dirRows.Close()

	for dirRows.Next() {
		var (
			movieID   int64
			directors []string
		)

		if err := dirRows.Scan(&movieID, &directors); err != nil {
			return nil, err
		}

		movies[idMap[movieID]].Directors = directors
	}

	if err = dirRows.Err(); err != nil {
		return nil, err
	}

	return movies, nil
}
