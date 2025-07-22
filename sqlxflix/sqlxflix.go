package sqlxflix

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/go-sqlt/benchflix"
	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
)

type Movie struct {
	ID        int64          `db:"id"`
	Title     string         `db:"title"`
	AddedAt   time.Time      `db:"added_at"`
	Rating    float64        `db:"rating"`
	Directors pq.StringArray `db:"directors"`
}

type Repository struct {
	DB *sqlx.DB
}

const queryStatement = `
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
		AND ($3::NUMERIC = 0 OR m.rating >= $3)
	ORDER BY m.rating DESC
	LIMIT CASE WHEN $4 BETWEEN 1 AND 1000 THEN $4 ELSE 1000 END;
`

func (r Repository) QueryList(ctx context.Context, params benchflix.ListParams) ([]benchflix.Movie, error) {
	var rows []Movie

	err := r.DB.SelectContext(ctx, &rows, queryStatement,
		params.Search, params.YearAdded, params.MinRating, params.Limit)
	if err != nil {
		return nil, err
	}

	var result = make([]benchflix.Movie, len(rows))

	for i, m := range rows {
		result[i] = benchflix.Movie{
			ID:        m.ID,
			Title:     m.Title,
			AddedAt:   m.AddedAt,
			Rating:    m.Rating,
			Directors: m.Directors,
		}
	}

	return result, nil
}

const (
	queryPreloadStatement = `
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
			AND ($3::NUMERIC = 0 OR m.rating >= $3)
		ORDER BY m.rating DESC
		LIMIT CASE WHEN $4 BETWEEN 1 AND 1000 THEN $4 ELSE 1000 END;
	`

	queryMovieDirectors = `
		SELECT md.movie_id, ARRAY_AGG(people.name ORDER BY people.name) AS directors
		FROM movie_directors md
		JOIN people ON people.id = md.person_id
		WHERE md.movie_id = ANY ($1)
		GROUP BY md.movie_id;
	`
)

func (r Repository) QueryListPreload(ctx context.Context, params benchflix.ListParams) ([]benchflix.Movie, error) {
	var (
		movies = make([]benchflix.Movie, 0, params.Limit)
		index  int
		ids    = make(pq.Int64Array, 0, params.Limit)
		idMap  = make(map[int64]int, params.Limit)
	)

	rows, err := r.DB.QueryxContext(ctx, queryPreloadStatement,
		params.Search, params.YearAdded, params.MinRating, params.Limit)
	if err != nil {
		return nil, err
	}

	defer rows.Close()

	for rows.Next() {
		var movie benchflix.Movie

		if err = rows.StructScan(&movie); err != nil {
			return nil, err
		}

		idMap[movie.ID] = index
		index++
		ids = append(ids, movie.ID)
		movies = append(movies, movie)
	}

	if err := rows.Close(); err != nil {
		return nil, err
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	if len(movies) == 0 {
		return movies, nil
	}

	dirRows, err := r.DB.QueryContext(ctx, queryMovieDirectors, ids)
	if err != nil {
		return nil, err
	}

	defer dirRows.Close()

	for dirRows.Next() {
		var (
			movieID   int64
			directors pq.StringArray
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
		sb     = &strings.Builder{}
		movies = make([]benchflix.Movie, 0, params.Limit)
	)

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
			to_tsvector('simple', m.title) @@ plainto_tsquery('simple', :search)
			OR EXISTS (
			SELECT 1 FROM movie_directors md
			JOIN people p ON p.id = md.person_id
			WHERE md.movie_id = m.id
				AND to_tsvector('simple', p.name) @@ plainto_tsquery('simple', :search)
			)
		)`)
	}
	if params.YearAdded != 0 {
		sb.WriteString(" AND EXTRACT(YEAR FROM m.added_at) = :year_added")
	}
	if params.MinRating != 0 {
		sb.WriteString(" AND m.rating >= :min_rating")
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
		sb.WriteString(" LIMIT :limit")
	}

	sql, args, err := r.DB.BindNamed(sb.String(), params)
	if err != nil {
		return nil, err
	}

	if params.WithDirectors {
		rows, err := r.DB.QueryContext(ctx, sql, args...)
		if err != nil {
			return nil, err
		}

		defer rows.Close()

		for rows.Next() {
			var (
				movie     benchflix.Movie
				directors pq.StringArray
			)

			if err := rows.Scan(&movie.ID, &movie.Title, &movie.AddedAt, &movie.Rating, &directors); err != nil {
				return nil, err
			}

			movie.Directors = directors

			movies = append(movies, movie)
		}

		if err = rows.Err(); err != nil {
			return nil, err
		}

		return movies, nil
	}

	err = r.DB.SelectContext(ctx, &movies, sql, args...)
	if err != nil {
		return nil, err
	}

	return movies, nil
}

func (r Repository) QueryDashboardPreload(ctx context.Context, params benchflix.DashboardParams) ([]benchflix.Movie, error) {
	var (
		movies = make([]benchflix.Movie, 0, params.Limit)
		index  = 0
		ids    = make(pq.Int64Array, 0, params.Limit)
		idMap  = make(map[int64]int, params.Limit)
		sb     = &strings.Builder{}
	)

	sb.WriteString(`SELECT m.id, m.title, m.added_at, m.rating FROM movies m WHERE 1=1`)

	if params.Search != "" {
		sb.WriteString(` AND (
			to_tsvector('simple', m.title) @@ plainto_tsquery('simple', :search)
			OR EXISTS (
			SELECT 1 FROM movie_directors md
			JOIN people p ON p.id = md.person_id
			WHERE md.movie_id = m.id
				AND to_tsvector('simple', p.name) @@ plainto_tsquery('simple', :search)
			)
		)`)
	}
	if params.YearAdded != 0 {
		sb.WriteString(" AND EXTRACT(YEAR FROM m.added_at) = :year_added")
	}
	if params.MinRating != 0 {
		sb.WriteString(" AND m.rating >= :min_rating")
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
		sb.WriteString(" LIMIT :limit")
	}

	sql, args, err := r.DB.BindNamed(sb.String(), params)
	if err != nil {
		return nil, err
	}

	rows, err := r.DB.QueryxContext(ctx, sql, args...)
	if err != nil {
		return nil, err
	}

	defer rows.Close()

	for rows.Next() {
		var movie benchflix.Movie

		if err := rows.StructScan(&movie); err != nil {
			return nil, err
		}

		idMap[movie.ID] = index
		index++
		ids = append(ids, movie.ID)

		movies = append(movies, movie)
	}

	if err := rows.Close(); err != nil {
		return nil, err
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	if len(movies) == 0 || !params.WithDirectors {
		return movies, nil
	}

	dirRows, err := r.DB.QueryContext(ctx, queryMovieDirectors, ids)
	if err != nil {
		return nil, err
	}

	defer dirRows.Close()

	for dirRows.Next() {
		var (
			movieID   int64
			directors pq.StringArray
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
