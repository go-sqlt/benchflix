package benchflix_test

import (
	"context"
	"database/sql"
	"encoding/csv"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"runtime"
	"slices"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/Masterminds/squirrel"
	"github.com/go-sqlt/benchflix"
	"github.com/go-sqlt/benchflix/gormflix"
	"github.com/go-sqlt/benchflix/pgxflix"
	"github.com/go-sqlt/benchflix/sqlcflix"
	"github.com/go-sqlt/benchflix/sqlflix"
	"github.com/go-sqlt/benchflix/sqltflix"
	"github.com/go-sqlt/benchflix/sqlxflix"
	"github.com/go-sqlt/benchflix/squirrelflix"
	"github.com/go-sqlt/sqlt"
	"github.com/jackc/pgx/v5/pgxpool"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/jmoiron/sqlx"
	"github.com/ory/dockertest"
	"github.com/ory/dockertest/docker"
	"golang.org/x/sync/errgroup"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var (
	ParamSize       = flag.Int("size", 0, "number of params")
	MaxConns        = 6
	MinConns        = 3
	IdleTimeout     = 2 * time.Minute
	Pool            = Must(dockertest.NewPool(""))
	Movies          []benchflix.Movie
	ListParams      []benchflix.ListParams
	DashboardParams []benchflix.DashboardParams
)

type PostgresRepository struct {
	Name       string
	Repository func(conn string) benchflix.Repository
}

var repositories = []PostgresRepository{
	{
		Name: "SQL",
		Repository: func(conn string) benchflix.Repository {
			db := Must(sql.Open("pgx", conn))

			db.SetMaxOpenConns(MaxConns)
			db.SetMaxIdleConns(MinConns)
			db.SetConnMaxIdleTime(IdleTimeout)

			return sqlflix.Repository{
				DB: db,
			}
		},
	},
	{
		Name: "PGX",
		Repository: func(conn string) benchflix.Repository {
			cfg := Must(pgxpool.ParseConfig(conn))

			cfg.MaxConns = int32(MaxConns)
			cfg.MinConns = int32(MinConns)
			cfg.MaxConnIdleTime = IdleTimeout

			pool := Must(pgxpool.NewWithConfig(context.Background(), cfg))

			return pgxflix.Repository{
				Pool: pool,
			}
		},
	},
	{
		Name: "SQUIRREL",
		Repository: func(conn string) benchflix.Repository {
			db := Must(sql.Open("pgx", conn))

			db.SetMaxOpenConns(MaxConns)
			db.SetMaxIdleConns(MinConns)
			db.SetConnMaxIdleTime(IdleTimeout)

			return squirrelflix.Repository{
				DB:     db,
				Select: squirrel.Select().PlaceholderFormat(squirrel.Dollar),
			}
		},
	},
	{
		Name: "SQLX",
		Repository: func(conn string) benchflix.Repository {
			db := Must(sqlx.Connect("postgres", conn))

			db.SetMaxOpenConns(MaxConns)
			db.SetMaxIdleConns(MinConns)
			db.SetConnMaxIdleTime(IdleTimeout)

			return sqlxflix.Repository{
				DB: db,
			}
		},
	},
	{
		Name: "GORM",
		Repository: func(conn string) benchflix.Repository {
			db := Must(gorm.Open(postgres.Open(conn), &gorm.Config{
				Logger:                 logger.Default.LogMode(logger.Silent),
				SkipDefaultTransaction: true,
				PrepareStmt:            true,
			}))

			sqldb := Must(db.DB())

			sqldb.SetMaxOpenConns(MaxConns)
			sqldb.SetMaxIdleConns(MinConns)
			sqldb.SetConnMaxIdleTime(IdleTimeout)

			return gormflix.Repository{
				DB: db,
			}
		},
	},
	{
		Name: "SQLC",
		Repository: func(conn string) benchflix.Repository {
			cfg := Must(pgxpool.ParseConfig(conn))

			cfg.MaxConns = int32(MaxConns)
			cfg.MinConns = int32(MinConns)
			cfg.MaxConnIdleTime = IdleTimeout

			pool := Must(pgxpool.NewWithConfig(context.Background(), cfg))

			return sqlcflix.Repository{
				Queries: sqlcflix.New(pool),
			}
		},
	},
	{
		Name: "SQLT-Cache",
		Repository: func(conn string) benchflix.Repository {
			cfg := Must(pgxpool.ParseConfig(conn))

			cfg.MaxConns = int32(MaxConns)
			cfg.MinConns = int32(MinConns)
			cfg.MaxConnIdleTime = IdleTimeout

			pool := Must(pgxpool.NewWithConfig(context.Background(), cfg))

			return sqltflix.New(pool, sqlt.ExpressionSize(10_000))
		},
	},
	{
		Name: "SQLT",
		Repository: func(conn string) benchflix.Repository {
			cfg := Must(pgxpool.ParseConfig(conn))

			cfg.MaxConns = int32(MaxConns)
			cfg.MinConns = int32(MinConns)
			cfg.MaxConnIdleTime = IdleTimeout

			pool := Must(pgxpool.NewWithConfig(context.Background(), cfg))

			return sqltflix.New(pool, sqlt.Config{})
		},
	},
}

func ExecBenchmark[P any](exec func(context.Context, P) ([]benchflix.Movie, error), params []P, b *testing.B) {
	_, err := exec(context.Background(), params[0])
	if err == benchflix.ErrSkip {
		b.SkipNow()

		return
	}

	var group errgroup.Group

	group.SetLimit(MaxConns)

	for i := range 12_500 {
		group.Go(func() error {
			_, err := exec(context.Background(), params[i%*ParamSize])
			if err != nil {
				return err
			}

			return nil
		})
	}

	if err = group.Wait(); err != nil {
		b.Fatal(err)

		return
	}

	runtime.GC()
	time.Sleep(500 * time.Millisecond)

	b.ResetTimer()

	b.RunParallel(func(pb *testing.PB) {
		i := 0

		for pb.Next() {
			_, err := exec(context.Background(), params[i%*ParamSize])
			if err != nil {
				b.Fatal(err)

				return
			}

			i++
		}
	})
}

func Benchmark(b *testing.B) {
	file, err := os.Open("./movies.csv")
	if err != nil {
		panic(err)
	}

	records, err := csv.NewReader(file).ReadAll()
	if err != nil {
		panic(err)
	}

	Movies = make([]benchflix.Movie, len(records)-1)

	for i, record := range records[1:] {
		Movies[i] = benchflix.Movie{
			ID:        Must(strconv.ParseInt(record[0], 10, 64)),
			Title:     record[2],
			AddedAt:   Must(time.Parse(time.DateOnly, record[6])),
			Rating:    Must(strconv.ParseFloat(record[8], 64)),
			Directors: strings.Split(record[3], ", "),
		}
	}

	params, err := os.Open("./params.json")
	if err != nil {
		panic(err)
	}

	data, err := io.ReadAll(params)
	if err != nil {
		panic(err)
	}

	if err = json.Unmarshal(data, &DashboardParams); err != nil {
		panic(err)
	}

	DashboardParams = DashboardParams[:*ParamSize]

	if err = json.Unmarshal(data, &ListParams); err != nil {
		panic(err)
	}

	ListParams = ListParams[:*ParamSize]

	for _, r := range repositories {
		b.Run(r.Name, func(b *testing.B) {
			conn, resource := initializePostgres(Pool, r.Name)

			defer resource.Close()
			repo := r.Repository(conn)

			b.Run("List", func(b *testing.B) {
				ExecBenchmark(repo.QueryList, ListParams, b)
			})

			b.Run("ListPreload", func(b *testing.B) {
				ExecBenchmark(repo.QueryListPreload, ListParams, b)
			})

			b.Run("Dashboard", func(b *testing.B) {
				ExecBenchmark(repo.QueryDashboard, DashboardParams, b)
			})

			b.Run("DashboardPreload", func(b *testing.B) {
				ExecBenchmark(repo.QueryDashboardPreload, DashboardParams, b)
			})

			_ = resource.Close()
		})
	}
}

func initializePostgres(pool *dockertest.Pool, name string) (string, *dockertest.Resource) {
	resource := dockerPostgres(pool, name)

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

func Must[T any](t T, err error) T {
	if err != nil {
		panic(err)
	}

	return t
}

func insertPostgres(ctx context.Context, pool *pgxpool.Pool, movie benchflix.Movie) {
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

func dockerPostgres(pool *dockertest.Pool, name string) *dockertest.Resource {
	if err := pool.Client.Ping(); err != nil {
		panic(fmt.Errorf("could not connect to Docker: %s", err))
	}

	if err := removePostgresContainer(pool, name); err != nil {
		panic(fmt.Errorf("removing old container: %w", err))
	}

	resource := Must(pool.RunWithOptions(&dockertest.RunOptions{
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

	if err := pool.Retry(func() error {
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
