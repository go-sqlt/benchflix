package benchflix_test

import (
	"context"
	"database/sql"
	"encoding/json"
	"flag"
	"io"
	"os"
	"runtime"
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
	ListParams      []benchflix.ListParams
	DashboardParams []benchflix.DashboardParams
)

type NamedRepository struct {
	Name       string
	Repository func(conn string) benchflix.Repository
}

var repositories = []NamedRepository{
	{
		Name: "SQL",
		Repository: func(conn string) benchflix.Repository {
			db := benchflix.Must(sql.Open("pgx", conn))

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
			cfg := benchflix.Must(pgxpool.ParseConfig(conn))

			cfg.MaxConns = int32(MaxConns)
			cfg.MinConns = int32(MinConns)
			cfg.MaxConnIdleTime = IdleTimeout

			pool := benchflix.Must(pgxpool.NewWithConfig(context.Background(), cfg))

			return pgxflix.Repository{
				Pool: pool,
			}
		},
	},
	{
		Name: "SQUIRREL",
		Repository: func(conn string) benchflix.Repository {
			db := benchflix.Must(sql.Open("pgx", conn))

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
			db := benchflix.Must(sqlx.Connect("postgres", conn))

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
			db := benchflix.Must(gorm.Open(postgres.Open(conn), &gorm.Config{
				Logger:                 logger.Default.LogMode(logger.Silent),
				SkipDefaultTransaction: true,
				PrepareStmt:            true,
			}))

			sqldb := benchflix.Must(db.DB())

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
			cfg := benchflix.Must(pgxpool.ParseConfig(conn))

			cfg.MaxConns = int32(MaxConns)
			cfg.MinConns = int32(MinConns)
			cfg.MaxConnIdleTime = IdleTimeout

			pool := benchflix.Must(pgxpool.NewWithConfig(context.Background(), cfg))

			return sqlcflix.Repository{
				Queries: sqlcflix.New(pool),
			}
		},
	},
	{
		Name: "SQLT-Cache",
		Repository: func(conn string) benchflix.Repository {
			cfg := benchflix.Must(pgxpool.ParseConfig(conn))

			cfg.MaxConns = int32(MaxConns)
			cfg.MinConns = int32(MinConns)
			cfg.MaxConnIdleTime = IdleTimeout

			pool := benchflix.Must(pgxpool.NewWithConfig(context.Background(), cfg))

			return sqltflix.New(pool, sqlt.ExpressionSize(10_000))
		},
	},
	{
		Name: "SQLT",
		Repository: func(conn string) benchflix.Repository {
			cfg := benchflix.Must(pgxpool.ParseConfig(conn))

			cfg.MaxConns = int32(MaxConns)
			cfg.MinConns = int32(MinConns)
			cfg.MaxConnIdleTime = IdleTimeout

			pool := benchflix.Must(pgxpool.NewWithConfig(context.Background(), cfg))

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
			conn, resource := benchflix.InitializePostgres(r.Name)

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
