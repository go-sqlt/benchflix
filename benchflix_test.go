package benchflix_test

import (
	"context"
	"encoding/json"
	"io"
	"os"
	"runtime"
	"testing"
	"time"

	"github.com/go-sqlt/benchflix"
	"github.com/go-sqlt/benchflix/gormflix"
	"github.com/go-sqlt/benchflix/pgxflix"
	"github.com/go-sqlt/benchflix/sqlcflix"
	"github.com/go-sqlt/benchflix/sqlflix"
	"github.com/go-sqlt/benchflix/sqltflix"
	"github.com/go-sqlt/benchflix/sqlxflix"
	"github.com/go-sqlt/benchflix/squirrelflix"
	"github.com/go-sqlt/sqlt"
	_ "github.com/jackc/pgx/v5/stdlib"
	"golang.org/x/sync/errgroup"
)

var (
	MaxConns        = 6
	MinConns        = 3
	IdleTimeout     = 2 * time.Minute
	ListParams      []benchflix.ListParams
	DashboardParams []benchflix.DashboardParams
)

type NamedRepository struct {
	Name       string
	Repository func(conn string, min, max int, idle time.Duration) benchflix.Repository
}

var repositories = []NamedRepository{
	{
		Name:       "SQL",
		Repository: sqlflix.NewRepository,
	},
	{
		Name:       "PGX",
		Repository: pgxflix.NewRepository,
	},
	{
		Name:       "SQUIRREL",
		Repository: squirrelflix.NewRepository,
	},
	{
		Name:       "SQLX",
		Repository: sqlxflix.NewRepository,
	},
	{
		Name:       "GORM",
		Repository: gormflix.NewRepository,
	},
	{
		Name:       "SQLC",
		Repository: sqlcflix.NewRepository,
	},
	{
		Name: "SQLT",
		Repository: func(conn string, min, max int, idle time.Duration) benchflix.Repository {
			return sqltflix.NewRepository(conn, min, max, idle, sqlt.Config{})
		},
	},
	{
		Name: "SQLT-Cache",
		Repository: func(conn string, min, max int, idle time.Duration) benchflix.Repository {
			return sqltflix.NewRepository(conn, min, max, idle, sqlt.ExpressionSize(10_000))
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

	size := len(params)

	for i := range 12_500 {
		group.Go(func() error {
			_, err := exec(context.Background(), params[i%size])
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
			_, err := exec(context.Background(), params[i%size])
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

	if err = json.Unmarshal(data, &ListParams); err != nil {
		panic(err)
	}

	for _, r := range repositories {
		b.Run(r.Name, func(b *testing.B) {
			conn, resource := benchflix.InitializePostgres(r.Name)

			defer resource.Close()

			repo := r.Repository(conn, MinConns, MaxConns, IdleTimeout)

			b.Run("List", func(b *testing.B) {
				b.Run("100", func(b *testing.B) {
					ExecBenchmark(repo.QueryList, ListParams[:100], b)
				})

				b.Run("1000", func(b *testing.B) {
					ExecBenchmark(repo.QueryList, ListParams[:1000], b)
				})
			})

			b.Run("ListPreload", func(b *testing.B) {
				b.Run("100", func(b *testing.B) {
					ExecBenchmark(repo.QueryListPreload, ListParams[:100], b)
				})

				b.Run("1000", func(b *testing.B) {
					ExecBenchmark(repo.QueryListPreload, ListParams[:1000], b)
				})
			})

			b.Run("Dashboard", func(b *testing.B) {
				b.Run("100", func(b *testing.B) {
					ExecBenchmark(repo.QueryDashboard, DashboardParams[:100], b)
				})

				b.Run("1000", func(b *testing.B) {
					ExecBenchmark(repo.QueryDashboard, DashboardParams[:1000], b)
				})
			})

			b.Run("DashboardPreload", func(b *testing.B) {
				b.Run("100", func(b *testing.B) {
					ExecBenchmark(repo.QueryDashboardPreload, DashboardParams[:100], b)
				})

				b.Run("1000", func(b *testing.B) {
					ExecBenchmark(repo.QueryDashboardPreload, DashboardParams[:1000], b)
				})
			})

			_ = resource.Close()
		})
	}
}
