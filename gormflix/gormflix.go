package gormflix

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/go-sqlt/benchflix"
	"gorm.io/gorm"
)

type Movie struct {
	ID        int64 `gorm:"primaryKey"`
	Title     string
	AddedAt   time.Time
	Rating    float64
	Directors []*Person `gorm:"many2many:movie_directors"`
}

type Person struct {
	ID   int64  `gorm:"primaryKey"`
	Name string `gorm:"unique;not null;index"`
}

type Repository struct {
	DB *gorm.DB
}

func (r Repository) QueryList(ctx context.Context, params benchflix.ListParams) ([]benchflix.Movie, error) {
	return nil, benchflix.ErrSkip
}

const queryListPreloadStatement = `
	SELECT
		m.id
		, m.title
		, m.added_at
		, m.rating
	FROM movies m
	WHERE
		(
			@search = ''
			OR to_tsvector('simple', m.title) @@ plainto_tsquery('simple', @search)
			OR EXISTS (
				SELECT 1
				FROM movie_directors md
				JOIN people p ON p.id = md.person_id
				WHERE md.movie_id = m.id
				AND to_tsvector('simple', p.name) @@ plainto_tsquery('simple', @search)
			)
		)
		AND (@year_added = 0 OR EXTRACT(YEAR FROM m.added_at) = @year_added)
		AND (@min_rating = 0 OR m.rating >= @min_rating)
	ORDER BY m.rating DESC
	LIMIT CASE WHEN @limit BETWEEN 1 AND 1000 THEN @limit ELSE 1000 END;
`

func (r Repository) QueryListPreload(ctx context.Context, params benchflix.ListParams) ([]benchflix.Movie, error) {
	var rows = make([]Movie, 0, params.Limit)

	if err := r.DB.Preload("Directors", func(db *gorm.DB) *gorm.DB {
		return db.Order("people.name DESC")
	}).Raw(queryListPreloadStatement,
		sql.Named("search", params.Search),
		sql.Named("year_added", params.YearAdded),
		sql.Named("min_rating", params.MinRating),
		sql.Named("limit", params.Limit)).
		Find(&rows).Error; err != nil {
		return nil, err
	}

	if len(rows) == 0 {
		return nil, nil
	}

	movies := make([]benchflix.Movie, len(rows))

	for i, m := range rows {
		movies[i] = benchflix.Movie{
			ID:      m.ID,
			Title:   m.Title,
			AddedAt: m.AddedAt,
			Rating:  m.Rating,
		}

		if len(m.Directors) == 0 {
			continue
		}

		movies[i].Directors = make([]string, len(m.Directors))

		for j, d := range m.Directors {
			movies[i].Directors[j] = d.Name
		}
	}

	return movies, nil
}

func (r Repository) QueryDashboard(ctx context.Context, params benchflix.DashboardParams) ([]benchflix.Movie, error) {
	return nil, benchflix.ErrSkip
}

func (r Repository) QueryDashboardPreload(ctx context.Context, params benchflix.DashboardParams) ([]benchflix.Movie, error) {
	var rows = make([]Movie, 0, params.Limit)

	query := r.DB.Table("movies")

	if params.WithDirectors {
		query = query.Preload("Directors", func(db *gorm.DB) *gorm.DB {
			return db.Order("people.name DESC")
		})
	}

	if params.Search != "" {
		query = query.Where(`(
			to_tsvector('simple', movies.title) @@ plainto_tsquery('simple', @search)
			OR EXISTS (
				SELECT 1 FROM movie_directors md
				JOIN people p ON p.id = md.person_id
				WHERE md.movie_id = movies.id
				AND to_tsvector('simple', p.name) @@ plainto_tsquery('simple', @search)
			)
		)`, sql.Named("search", params.Search))
	}

	if params.YearAdded != 0 {
		query = query.Where("EXTRACT(YEAR FROM movies.added_at) = ?", params.YearAdded)
	}

	if params.MinRating != 0 {
		query = query.Where("movies.rating >= ?", params.MinRating)
	}

	switch params.Sort {
	case "rating":
		if params.Desc {
			query = query.Order("movies.rating DESC")
		} else {
			query = query.Order("movies.rating ASC")
		}
	case "title":
		if params.Desc {
			query = query.Order("movies.title DESC")
		} else {
			query = query.Order("movies.title ASC")
		}
	case "added_at":
		if params.Desc {
			query = query.Order("movies.added_at DESC")
		} else {
			query = query.Order("movies.added_at ASC")
		}
	default:
		return nil, fmt.Errorf("invalid sort")
	}

	if params.Limit < 1 || params.Limit > 1000 {
		query = query.Limit(1000)
	} else {
		query = query.Limit(int(params.Limit))
	}

	if err := query.Find(&rows).Error; err != nil {
		return nil, err
	}

	movies := make([]benchflix.Movie, len(rows))
	for i, m := range rows {
		movies[i] = benchflix.Movie{
			ID:      m.ID,
			Title:   m.Title,
			AddedAt: m.AddedAt,
			Rating:  m.Rating,
		}

		if len(m.Directors) == 0 {
			continue
		}

		movies[i].Directors = make([]string, len(m.Directors))

		for j, d := range m.Directors {
			movies[i].Directors[j] = d.Name
		}
	}

	return movies, nil
}
