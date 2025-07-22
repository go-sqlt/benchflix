package squirrelflix

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/Masterminds/squirrel"
	"github.com/go-sqlt/benchflix"
	"github.com/lib/pq"
)

type Repository struct {
	DB     *sql.DB
	Select squirrel.SelectBuilder
}

func (r Repository) QueryList(ctx context.Context, params benchflix.ListParams) ([]benchflix.Movie, error) {
	return nil, benchflix.ErrSkip
}

func (r Repository) QueryListPreload(ctx context.Context, params benchflix.ListParams) ([]benchflix.Movie, error) {
	return nil, benchflix.ErrSkip
}

func (r Repository) QueryDashboard(ctx context.Context, params benchflix.DashboardParams) ([]benchflix.Movie, error) {
	sb := r.Select.Columns("m.id", "m.title", "m.added_at", "m.rating").From("movies AS m")

	if params.WithDirectors {
		sb = sb.Column("d.directors").LeftJoin(`LATERAL (
			SELECT ARRAY_AGG(p.name ORDER BY p.name) AS directors
			FROM movie_directors md
			JOIN people p ON p.id = md.person_id
			WHERE md.movie_id = m.id
		) d ON true`)
	}

	if params.Search != "" {
		sb = sb.Where(`
			to_tsvector('simple', m.title) @@ plainto_tsquery('simple', ?)
			OR EXISTS (
			SELECT 1 FROM movie_directors md
			JOIN people p ON p.id = md.person_id
			WHERE md.movie_id = m.id
				AND to_tsvector('simple', p.name) @@ plainto_tsquery('simple', ?)
			)
		`, params.Search, params.Search)
	}

	if params.YearAdded != 0 {
		sb = sb.Where("EXTRACT(YEAR FROM m.added_at) = ?", params.YearAdded)
	}

	if params.MinRating != 0 {
		sb = sb.Where("m.rating >= ?", params.MinRating)
	}

	order := "ASC"

	if params.Desc {
		order = "DESC"
	}

	switch params.Sort {
	case "rating":
		sb = sb.OrderBy(fmt.Sprintf("m.rating %s", order))
	case "title":
		sb = sb.OrderBy(fmt.Sprintf("m.title %s", order))
	case "added_at":
		sb = sb.OrderBy(fmt.Sprintf("m.added_at %s", order))
	default:
		return nil, errors.New("invalid sort")
	}

	if params.Limit < 1 || params.Limit > 1000 {
		sb = sb.Limit(1000)
	} else {
		sb = sb.Limit(params.Limit)
	}

	rows, err := sb.RunWith(r.DB).QueryContext(ctx)
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

const QueryDirectors = `
	SELECT
		md.movie_id
		, ARRAY_AGG(people.name ORDER BY people.name) AS directors
	FROM movie_directors md
	JOIN people ON people.id = md.person_id
	WHERE md.movie_id = ANY ($1)
	GROUP BY md.movie_id;
`

func (r Repository) QueryDashboardPreload(ctx context.Context, params benchflix.DashboardParams) ([]benchflix.Movie, error) {
	sb := r.Select.Columns("m.id", "m.title", "m.added_at", "m.rating").From("movies AS m")

	if params.Search != "" {
		sb = sb.Where(`
			to_tsvector('simple', m.title) @@ plainto_tsquery('simple', ?)
			OR EXISTS (
				SELECT 1 FROM movie_directors md
				JOIN people p ON p.id = md.person_id
				WHERE md.movie_id = m.id
				AND to_tsvector('simple', p.name) @@ plainto_tsquery('simple', ?)
			)
		`, params.Search, params.Search)
	}

	if params.YearAdded != 0 {
		sb = sb.Where("EXTRACT(YEAR FROM m.added_at) = ?", params.YearAdded)
	}

	if params.MinRating != 0 {
		sb = sb.Where("m.rating >= ?", params.MinRating)
	}

	order := "ASC"

	if params.Desc {
		order = "DESC"
	}

	switch params.Sort {
	case "rating":
		sb = sb.OrderBy(fmt.Sprintf("m.rating %s", order))
	case "title":
		sb = sb.OrderBy(fmt.Sprintf("m.title %s", order))
	case "added_at":
		sb = sb.OrderBy(fmt.Sprintf("m.added_at %s", order))
	default:
		return nil, errors.New("invalid sort")
	}

	if params.Limit < 1 || params.Limit > 1000 {
		sb = sb.Limit(1000)
	} else {
		sb = sb.Limit(params.Limit)
	}

	rows, err := sb.RunWith(r.DB).QueryContext(ctx)
	if err != nil {
		return nil, err
	}

	defer rows.Close()

	var (
		movies = make([]benchflix.Movie, 0, params.Limit)
		index  int
		ids    = make([]int64, 0, params.Limit)
		idMap  = make(map[int64]int, params.Limit)
	)

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

	sb = r.Select.
		Columns("md.movie_id", "ARRAY_AGG(p.name ORDER BY p.name) AS directors").
		From("movie_directors md").Join("people p ON p.id = md.person_id").
		Where("md.movie_id = ANY(?)", ids).GroupBy("md.movie_id")

	dirRows, err := sb.RunWith(r.DB).QueryContext(ctx)
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
