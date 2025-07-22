package pgxflix

import (
	"context"
	"fmt"
	"strings"

	"github.com/go-sqlt/benchflix"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository struct {
	Pool *pgxpool.Pool
}

var QueryList = `
	SELECT
		m.id
		, m.title
		, m.added_at
		, m.rating
		, d.directors
	FROM movies m
	LEFT JOIN LATERAL (
		SELECT ARRAY_AGG(p.name ORDER BY p.name) AS directors
		FROM movie_directors md
		JOIN people p ON p.id = md.person_id
		WHERE md.movie_id = m.id
	) d ON true
	WHERE
		(
			$1 = ''
			OR to_tsvector('simple', m.title) @@ plainto_tsquery('simple', $1)
			OR EXISTS (
				SELECT 1
				FROM movie_directors md
				JOIN people p ON p.id = md.person_id
				WHERE md.movie_id = m.id
				AND to_tsvector('simple', p.name) @@ plainto_tsquery('simple', $1)
			)
		)
		AND ($2 = 0 OR EXTRACT(YEAR FROM m.added_at) = $2)
		AND ($3 = 0 OR m.rating >= $3)
	ORDER BY m.rating DESC
	LIMIT CASE WHEN $4 BETWEEN 1 AND 1000 THEN $4 ELSE 1000 END;
`

func (r Repository) QueryList(ctx context.Context, params benchflix.ListParams) ([]benchflix.Movie, error) {
	rows, err := r.Pool.Query(ctx, QueryList, params.Search, params.YearAdded, params.MinRating, params.Limit)
	if err != nil {
		return nil, err
	}

	return pgx.CollectRows(rows, func(row pgx.CollectableRow) (benchflix.Movie, error) {
		var m benchflix.Movie

		if err := row.Scan(&m.ID, &m.Title, &m.AddedAt, &m.Rating, &m.Directors); err != nil {
			return m, err
		}

		return m, nil
	})
}

var (
	QueryListPreload = `
		SELECT
			m.id
			, m.title
			, m.added_at
			, m.rating
		FROM movies m
		WHERE
			(
				$1 = ''
				OR to_tsvector('simple', m.title) @@ plainto_tsquery('simple', $1)
				OR EXISTS (
					SELECT 1
					FROM movie_directors md
					JOIN people p ON p.id = md.person_id
					WHERE md.movie_id = m.id
					AND to_tsvector('simple', p.name) @@ plainto_tsquery('simple', $1)
				)
			)
			AND ($2 = 0 OR EXTRACT(YEAR FROM m.added_at) = $2)
			AND ($3 = 0 OR m.rating >= $3)
		ORDER BY m.rating DESC
		LIMIT CASE WHEN $4 BETWEEN 1 AND 1000 THEN $4 ELSE 1000 END;
	`

	QueryDirectors = `
		SELECT
			md.movie_id
			, ARRAY_AGG(people.name ORDER BY people.name) AS directors
		FROM movie_directors md
		JOIN people ON people.id = md.person_id
		WHERE md.movie_id = ANY ($1)
		GROUP BY md.movie_id;
	`
)

func (r Repository) QueryListPreload(ctx context.Context, params benchflix.ListParams) ([]benchflix.Movie, error) {
	var (
		index int
		ids   = make([]int64, 0, params.Limit)
		idMap = make(map[int64]int, params.Limit)
	)

	rows, err := r.Pool.Query(ctx, QueryListPreload,
		params.Search,
		params.YearAdded,
		params.MinRating,
		params.Limit,
	)
	if err != nil {
		return nil, err
	}

	movies, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (benchflix.Movie, error) {
		var movie benchflix.Movie

		if err := row.Scan(&movie.ID, &movie.Title, &movie.AddedAt, &movie.Rating); err != nil {
			return movie, err
		}

		idMap[movie.ID] = index
		index++
		ids = append(ids, movie.ID)

		return movie, nil
	})
	if err != nil {
		return nil, err
	}

	if len(movies) == 0 {
		return movies, nil
	}

	dirRows, err := r.Pool.Query(ctx, QueryDirectors, ids)
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
	sb := &strings.Builder{}

	sb.WriteString("SELECT m.id, m.title, m.added_at, m.rating")

	if params.WithDirectors {
		sb.WriteString(", d.directors")
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

	return pgx.CollectRows(rows, func(row pgx.CollectableRow) (benchflix.Movie, error) {
		var m benchflix.Movie

		if params.WithDirectors {
			if err := row.Scan(&m.ID, &m.Title, &m.AddedAt, &m.Rating, &m.Directors); err != nil {
				return m, err
			}
		} else {
			if err := row.Scan(&m.ID, &m.Title, &m.AddedAt, &m.Rating); err != nil {
				return m, err
			}
		}

		return m, nil
	})
}

func (r Repository) QueryDashboardPreload(ctx context.Context, params benchflix.DashboardParams) ([]benchflix.Movie, error) {
	var (
		index = 0
		ids   = make([]int64, 0, params.Limit)
		idMap = make(map[int64]int, params.Limit)
		sb    = &strings.Builder{}
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

	movies, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (benchflix.Movie, error) {
		var movie benchflix.Movie

		if err := row.Scan(&movie.ID, &movie.Title, &movie.AddedAt, &movie.Rating); err != nil {
			return movie, err
		}

		idMap[movie.ID] = index
		index++
		ids = append(ids, movie.ID)

		return movie, nil
	})
	if err != nil {
		return nil, err
	}

	if !params.WithDirectors || len(movies) == 0 {
		return movies, nil
	}

	dirRows, err := r.Pool.Query(ctx, QueryDirectors, ids)
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
