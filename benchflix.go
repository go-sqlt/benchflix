package benchflix

import (
	"bufio"
	"context"
	"database/sql"
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"os"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/ory/dockertest"
	"github.com/ory/dockertest/docker"
	"golang.org/x/tools/benchmark/parse"
)

var (
	ErrSkip = errors.New("skip")
	Pool    = Must(dockertest.NewPool(""))
	Movies  []Movie
)

type Movie struct {
	ID        int64
	Title     string
	AddedAt   time.Time `db:"added_at"`
	Rating    float64
	Directors []string
}

type ListParams struct {
	Search    string
	YearAdded int64   `db:"year_added" json:"year_added"`
	MinRating float64 `db:"min_rating" json:"min_rating"`
	Limit     uint64
}

type DashboardParams struct {
	Search        string
	YearAdded     int64   `db:"year_added" json:"year_added"`
	MinRating     float64 `db:"min_rating" json:"min_rating"`
	Limit         uint64
	Sort          string
	Desc          bool
	WithDirectors bool `json:"with_directors"`
}

type Repository interface {
	QueryList(ctx context.Context, params ListParams) ([]Movie, error)
	QueryListPreload(ctx context.Context, params ListParams) ([]Movie, error)
	QueryDashboard(ctx context.Context, params DashboardParams) ([]Movie, error)
	QueryDashboardPreload(ctx context.Context, params DashboardParams) ([]Movie, error)
}

func Must[T any](t T, err error) T {
	if err != nil {
		panic(err)
	}

	return t
}

func init() {
	file, err := os.Open("./movies.csv")
	if err != nil {
		panic(err)
	}

	records, err := csv.NewReader(file).ReadAll()
	if err != nil {
		panic(err)
	}

	Movies = make([]Movie, len(records)-1)

	for i, record := range records[1:] {
		Movies[i] = Movie{
			ID:        Must(strconv.ParseInt(record[0], 10, 64)),
			Title:     record[2],
			AddedAt:   Must(time.Parse(time.DateOnly, record[6])),
			Rating:    Must(strconv.ParseFloat(record[8], 64)),
			Directors: strings.Split(record[3], ", "),
		}
	}
}

func InitializePostgres(name string) (string, *dockertest.Resource) {
	resource := dockerPostgres(name)

	conn := fmt.Sprintf("host=localhost port=%s user=user password=password dbname=db sslmode=disable timezone=UTC", resource.GetPort("5432/tcp"))

	cfg := Must(pgxpool.ParseConfig(conn))

	db := Must(pgxpool.NewWithConfig(context.Background(), cfg))

	defer db.Close()

	_ = Must(db.Exec(context.Background(), `
		CREATE TABLE IF NOT EXISTS movies (
			id INTEGER PRIMARY KEY
			, title TEXT NOT NULL
			, added_at DATE NOT NULL
			, rating NUMERIC NOT NULL
		);

		CREATE TABLE IF NOT EXISTS people (
			id SERIAL PRIMARY KEY
			, name TEXT NOT NULL UNIQUE
		);

		CREATE TABLE IF NOT EXISTS movie_directors (
			movie_id INTEGER REFERENCES movies (id) ON DELETE CASCADE
			, person_id INTEGER REFERENCES people (id) ON DELETE CASCADE
			, PRIMARY KEY (movie_id, person_id)
		);

		CREATE INDEX IF NOT EXISTS idx_movies_title_fts ON movies USING GIN (to_tsvector('simple', title));
		CREATE INDEX IF NOT EXISTS idx_people_name_fts ON people USING GIN (to_tsvector('simple', name));
		CREATE INDEX IF NOT EXISTS idx_movies_added_year ON movies (EXTRACT(YEAR FROM added_at));
		CREATE INDEX IF NOT EXISTS idx_movies_added_at ON movies (added_at);
		CREATE INDEX IF NOT EXISTS idx_movies_rating ON movies (rating);
		CREATE INDEX IF NOT EXISTS idx_movies_title ON movies (title);
		CREATE INDEX IF NOT EXISTS idx_md_movie_person ON movie_directors (movie_id, person_id);
	`))

	for _, movie := range Movies {
		insertPostgres(context.Background(), db, movie)
	}

	return conn, resource
}

func insertPostgres(ctx context.Context, pool *pgxpool.Pool, movie Movie) {
	_ = Must(pool.Exec(ctx,
		`INSERT INTO movies (id, title, added_at, rating) VALUES ($1, $2, $3, $4) ON CONFLICT DO NOTHING;`,
		movie.ID, movie.Title, movie.AddedAt, movie.Rating,
	))

	if len(movie.Directors) == 0 {
		return
	}

	var (
		sb   strings.Builder
		args []any
	)

	for i, d := range movie.Directors {
		if i > 0 {
			sb.WriteString(",")
		}

		args = append(args, d)
		sb.WriteString(fmt.Sprintf("($%d)", len(args)))
	}

	rows := Must(pool.Query(ctx,
		`INSERT INTO people (name) VALUES `+sb.String()+` ON CONFLICT (name) DO UPDATE SET name = EXCLUDED.name RETURNING id;`,
		args...,
	))

	defer rows.Close()

	sb.Reset()
	args = []any{movie.ID}

	for rows.Next() {
		var id int64

		if err := rows.Scan(&id); err != nil {
			panic(err)
		}

		args = append(args, id)
		sb.WriteString(fmt.Sprintf(",($1, $%d)", len(args)))
	}

	_ = Must(pool.Exec(ctx,
		`INSERT INTO movie_directors (movie_id, person_id) VALUES `+sb.String()[1:]+" ON CONFLICT DO NOTHING;",
		args...,
	))
}

func dockerPostgres(name string) *dockertest.Resource {
	if err := Pool.Client.Ping(); err != nil {
		panic(fmt.Errorf("could not connect to Docker: %s", err))
	}

	if err := removePostgresContainer(Pool, name); err != nil {
		panic(fmt.Errorf("removing old container: %w", err))
	}

	resource := Must(Pool.RunWithOptions(&dockertest.RunOptions{
		Name:       name,
		Repository: "postgres",
		Tag:        "17",
		Env: []string{
			"POSTGRES_USER=user",
			"POSTGRES_PASSWORD=password",
			"POSTGRES_DB=db",
		},
	}, func(config *docker.HostConfig) {
		config.AutoRemove = true
		config.RestartPolicy = docker.RestartPolicy{Name: "no"}
	}))

	if err := Pool.Retry(func() error {
		db, err := sql.Open("pgx", fmt.Sprintf(
			"host=localhost port=%s user=user password=password dbname=db sslmode=disable",
			resource.GetPort("5432/tcp"),
		))
		if err != nil {
			return err
		}
		defer db.Close()

		return db.Ping()
	}); err != nil {
		_ = resource.Close()

		panic(fmt.Errorf("postgres never became ready: %w", err))
	}

	return resource
}

func removePostgresContainer(pool *dockertest.Pool, name string) error {
	containers := Must(pool.Client.ListContainers(docker.ListContainersOptions{All: true}))

	for _, c := range containers {
		if slices.Contains(c.Names, "/"+name) {
			return pool.Client.RemoveContainer(docker.RemoveContainerOptions{
				ID:    c.ID,
				Force: true,
			})
		}
	}

	return nil
}

type Benchmark struct {
	SQL, PGX, SQUIRREL, SQLX, GORM, SQLC, SQLT, SQLTCACHE Framework
}

type Framework struct {
	List, ListPreload, Dashboard, DashboardPreload Szenario
}

type Szenario struct {
	Hundred, Thousand Params
}

type Params struct {
	NsPerOp, BytesPerOp, AllocsPerOp []float64
}

func ReadAll(reader io.Reader) (Benchmark, error) {
	bench := Benchmark{}

	scan := bufio.NewScanner(reader)

	for scan.Scan() {
		line := scan.Text()

		if !strings.HasPrefix(line, "Benchmark") {
			continue
		}

		b, err := parse.ParseLine(line)
		if err != nil {
			panic(err)
		}

		parts := strings.Split(b.Name, "/")

		var framework *Framework

		switch parts[1] {
		case "SQL":
			framework = &bench.SQL
		case "PGX":
			framework = &bench.PGX
		case "SQUIRREL":
			framework = &bench.SQUIRREL
		case "SQLX":
			framework = &bench.SQLX
		case "GORM":
			framework = &bench.GORM
		case "SQLC":
			framework = &bench.SQLC
		case "SQLT":
			framework = &bench.SQLT
		case "SQLT-Cache":
			framework = &bench.SQLTCACHE
		default:
			return bench, fmt.Errorf("invalid framework: %s", parts[1])
		}

		var szenario *Szenario

		switch parts[2] {
		case "List":
			szenario = &framework.List
		case "ListPreload":
			szenario = &framework.ListPreload
		case "Dashboard":
			szenario = &framework.Dashboard
		case "DashboardPreload":
			szenario = &framework.DashboardPreload
		default:
			return bench, fmt.Errorf("invalid szenario: %s", parts[2])
		}

		switch {
		case strings.HasPrefix(parts[3], "100-"):
			szenario.Hundred.NsPerOp = append(szenario.Hundred.NsPerOp, b.NsPerOp)
			szenario.Hundred.BytesPerOp = append(szenario.Hundred.NsPerOp, float64(b.AllocedBytesPerOp))
			szenario.Hundred.AllocsPerOp = append(szenario.Hundred.NsPerOp, float64(b.AllocedBytesPerOp))
		case strings.HasPrefix(parts[3], "1000-"):
			szenario.Thousand.NsPerOp = append(szenario.Thousand.NsPerOp, b.NsPerOp)
			szenario.Thousand.BytesPerOp = append(szenario.Thousand.NsPerOp, float64(b.AllocedBytesPerOp))
			szenario.Thousand.AllocsPerOp = append(szenario.Thousand.NsPerOp, float64(b.AllocedBytesPerOp))
		default:
			return bench, fmt.Errorf("invalid params: %s", parts[3])
		}
	}

	return bench, nil
}
