package sqlflix

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/go-sqlt/benchflix"
	"github.com/lib/pq"
)

func NewRepository(conn string, min, max int, idle time.Duration) benchflix.Repository {
	db := benchflix.Must(sql.Open("pgx", conn))

	db.SetMaxOpenConns(max)
	db.SetMaxIdleConns(min)
	db.SetConnMaxIdleTime(idle)

	return Repository{
		DB: db,
	}
}

type Repository struct {
	DB *sql.DB
}

func (r Repository) QueryList(ctx context.Context, params benchflix.ListParams) ([]benchflix.Movie, error) {
	rows, err := r.DB.QueryContext(ctx, `
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
	`, params.Search, params.YearAdded, params.MinRating, params.Limit)
	if err != nil {
		return nil, err
	}

	defer rows.Close()

	var movies = make([]benchflix.Movie, 0, params.Limit)

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

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return movies, nil
}

func (r Repository) QueryListPreload(ctx context.Context, params benchflix.ListParams) ([]benchflix.Movie, error) {
	var (
		movies = make([]benchflix.Movie, 0, params.Limit)
		index  = 0
		ids    = make([]int64, 0, params.Limit)
		idMap  = make(map[int64]int, params.Limit)
	)

	rows, err := r.DB.QueryContext(ctx, `
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
	`, params.Search, params.YearAdded, params.MinRating, params.Limit)
	if err != nil {
		return nil, err
	}

	defer rows.Close()

	for rows.Next() {
		var movie benchflix.Movie

		if err := rows.Scan(&movie.ID, &movie.Title, &movie.AddedAt, &movie.Rating); err != nil {
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

	dirRows, err := r.DB.QueryContext(ctx, `
		SELECT md.movie_id, ARRAY_AGG(people.name ORDER BY people.name) AS directors
		FROM movie_directors md
		JOIN people ON people.id = md.person_id
		WHERE md.movie_id = ANY ($1)
		GROUP BY md.movie_id;
	`, ids)
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
		movies     = make([]benchflix.Movie, 0, params.Limit)
		sb         = &strings.Builder{}
		args       = make([]any, 0, 3)
		paramIndex = 1
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
		fmt.Fprintf(sb, ` AND (
			to_tsvector('simple', m.title) @@ plainto_tsquery('simple', $%d)
			OR EXISTS (
			SELECT 1 FROM movie_directors md
			JOIN people p ON p.id = md.person_id
			WHERE md.movie_id = m.id
				AND to_tsvector('simple', p.name) @@ plainto_tsquery('simple', $%d)
			)
		)`, paramIndex, paramIndex)
		args = append(args, params.Search)
		paramIndex++
	}

	if params.YearAdded != 0 {
		fmt.Fprintf(sb, " AND EXTRACT(YEAR FROM m.added_at) = $%d", paramIndex)
		args = append(args, params.YearAdded)
		paramIndex++
	}

	if params.MinRating != 0 {
		fmt.Fprintf(sb, " AND m.rating >= $%d", paramIndex)
		args = append(args, params.MinRating)
		paramIndex++
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

	rows, err := r.DB.QueryContext(ctx, sb.String(), args...)
	if err != nil {
		return nil, err
	}

	defer rows.Close()

	for rows.Next() {
		var (
			movie     benchflix.Movie
			directors pq.StringArray
		)

		if params.WithDirectors {
			if err := rows.Scan(&movie.ID, &movie.Title, &movie.AddedAt, &movie.Rating, &directors); err != nil {
				return nil, err
			}

			movie.Directors = directors
		} else {
			if err := rows.Scan(&movie.ID, &movie.Title, &movie.AddedAt, &movie.Rating); err != nil {
				return nil, err
			}
		}

		movies = append(movies, movie)
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	return movies, nil
}

func (r Repository) QueryDashboardPreload(ctx context.Context, params benchflix.DashboardParams) ([]benchflix.Movie, error) {
	var (
		movies     = make([]benchflix.Movie, 0, params.Limit)
		index      = 0
		ids        = make([]int64, 0, params.Limit)
		idMap      = make(map[int64]int, params.Limit)
		sb         = &strings.Builder{}
		args       = make([]any, 0, 3)
		paramIndex = 1
	)

	sb.WriteString(`
		SELECT m.id, m.title, m.added_at, m.rating
		FROM movies m
		WHERE 1=1`)

	if params.Search != "" {
		fmt.Fprintf(sb, ` AND (
			to_tsvector('simple', m.title) @@ plainto_tsquery('simple', $%d)
			OR EXISTS (
			SELECT 1 FROM movie_directors md
			JOIN people p ON p.id = md.person_id
			WHERE md.movie_id = m.id
				AND to_tsvector('simple', p.name) @@ plainto_tsquery('simple', $%d)
			)
		)`, paramIndex, paramIndex)
		args = append(args, params.Search)
		paramIndex++
	}

	if params.YearAdded != 0 {
		fmt.Fprintf(sb, " AND EXTRACT(YEAR FROM m.added_at) = $%d", paramIndex)
		args = append(args, params.YearAdded)
		paramIndex++
	}

	if params.MinRating != 0 {
		fmt.Fprintf(sb, " AND m.rating >= $%d", paramIndex)
		args = append(args, params.MinRating)
		paramIndex++
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

	rows, err := r.DB.QueryContext(ctx, sb.String(), args...)
	if err != nil {
		return nil, err
	}

	defer rows.Close()

	for rows.Next() {
		var movie benchflix.Movie

		if err := rows.Scan(&movie.ID, &movie.Title, &movie.AddedAt, &movie.Rating); err != nil {
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

	if !params.WithDirectors || len(movies) == 0 {
		return movies, nil
	}

	dirRows, err := r.DB.QueryContext(ctx, `
		SELECT md.movie_id, ARRAY_AGG(people.name ORDER BY people.name) AS directors
		FROM movie_directors md
		JOIN people ON people.id = md.person_id
		WHERE md.movie_id = ANY ($1)
		GROUP BY md.movie_id;
	`, ids)
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
