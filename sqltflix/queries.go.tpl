{{ define "filter_list" }}
    WHERE (
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
{{ end }}

{{ define "query_list" }}
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
    {{ template "filter_list" . }}
{{ end }}

{{ define "query_list_preload" }}
    SELECT
        m.id                    {{ Scan.Int.To "ID" }}
        , m.title               {{ Scan.String.To "Title" }}
        , m.added_at            {{ Scan.Time.To "AddedAt" }}
        , m.rating              {{ Scan.Float.To "Rating" }}
    FROM movies m
    {{ template "filter_list" . }}
{{ end }}

{{ define "query_directors" }}
    SELECT
        md.movie_id			{{ Scan.Int.To "MovieID" }}
        , ARRAY_AGG(people.name ORDER BY people.name) 
            AS directors 	{{ Scan.StringSlice.To "Directors" }}
    FROM movie_directors md
    JOIN people ON people.id = md.person_id
    WHERE md.movie_id = ANY ({{ . }})
    GROUP BY md.movie_id;
{{ end }}

{{ define "filter_dashboard" }}
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
    {{ if .Desc }} DESC{{ else }} ASC {{ end }}
    {{ if and (gt .Limit 0) (lt .Limit 1000) }} LIMIT {{ .Limit }}{{ else }} LIMIT 1000{{ end }}
{{ end }}

{{ define "query_dashboard" }}
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
    {{ template "filter_dashboard" . }}
{{ end }}

{{ define "query_dashboard_preload" }}
    SELECT
        m.id                    {{ Scan.Int.To "ID" }}
        , m.title               {{ Scan.String.To "Title" }}
        , m.added_at            {{ Scan.Time.To "AddedAt" }}
        , m.rating              {{ Scan.Float.To "Rating" }}
    FROM movies m
    {{ template "filter_dashboard" . }}
{{ end }}