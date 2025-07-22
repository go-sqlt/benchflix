-- name: Query :many
SELECT
    m.id
    , m.title
    , m.added_at
    , m.rating
    , d.directors::TEXT[] AS directors
FROM movies m
LEFT JOIN LATERAL (
    SELECT ARRAY_AGG(p.name ORDER BY p.name) AS directors
    FROM movie_directors md
    JOIN people p ON p.id = md.person_id
    WHERE md.movie_id = m.id
) d ON true
WHERE
    (
        sqlc.narg(search)::TEXT = ''
        OR to_tsvector('simple', m.title) @@ plainto_tsquery('simple', sqlc.narg(search))
        OR EXISTS (
            SELECT 1
            FROM movie_directors md
            JOIN people p ON p.id = md.person_id
            WHERE md.movie_id = m.id
            AND to_tsvector('simple', p.name) @@ plainto_tsquery('simple', sqlc.narg(search))
        )
    )
    AND (sqlc.narg(year_added)::INT8 = 0 OR EXTRACT(YEAR FROM m.added_at) = sqlc.narg(year_added))
    AND (sqlc.narg(min_rating)::FLOAT8 = 0 OR m.rating >= sqlc.narg(min_rating))
ORDER BY m.rating DESC
LIMIT CASE WHEN sqlc.narg('limit')::INT4 BETWEEN 1 AND 1000 THEN sqlc.narg('limit') ELSE 1000 END;

-- name: QueryPreload :many
SELECT
    id
    , title
    , added_at
    , rating
FROM movies m
WHERE
    (
        sqlc.narg(search)::TEXT = ''
        OR to_tsvector('simple', title) @@ plainto_tsquery('simple', sqlc.narg(search))
        OR EXISTS (
            SELECT 1
            FROM movie_directors md
            JOIN people p ON p.id = md.person_id
            WHERE md.movie_id = id
            AND to_tsvector('simple', p.name) @@ plainto_tsquery('simple', sqlc.narg(search))
        )
    )
    AND (sqlc.narg(year_added)::INT8 = 0 OR EXTRACT(YEAR FROM added_at) = sqlc.narg(year_added))
    AND (sqlc.narg(min_rating)::FLOAT8 = 0 OR rating >= sqlc.narg(min_rating))
ORDER BY rating DESC
LIMIT CASE WHEN sqlc.narg('limit')::INT4 BETWEEN 1 AND 1000 THEN sqlc.narg('limit') ELSE 1000 END;

-- name: QueryDirectors :many
SELECT
    md.movie_id
    , ARRAY_AGG(people.name ORDER BY people.name)::TEXT[] AS directors
FROM movie_directors md
JOIN people ON people.id = md.person_id
WHERE md.movie_id = ANY ($1::INT8[])
GROUP BY md.movie_id;