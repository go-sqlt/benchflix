package benchflix_test

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"slices"
	"strconv"
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
	WarmupSize  = 250_000
	MaxConns    = 6
	MinConns    = 3
	IdleTimeout = 2 * time.Minute
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
			return sqltflix.NewRepository(conn, min, max, idle, sqlt.ExpressionSize(5_000))
		},
	},
}

func Benchmark(b *testing.B) {
	for _, size := range []int{500, 5_000} {
		b.Run(fmt.Sprintf("Size-%d", size), func(b *testing.B) {
			chunkSize := size / 5

			if chunkSize*5 != size {
				b.Fail()

				return
			}

			stats := &benchflix.Stats{
				Search:        map[string]int{},
				Sort:          map[string]int{},
				Desc:          map[string]int{},
				WithDirectors: map[string]int{},
				Limit:         map[string]int{},
				MinRating:     map[string]int{},
				YearAdded:     map[string]int{},
			}

			listParams := make([]benchflix.ListParams, size)
			dashboardParams := make([]benchflix.DashboardParams, size)
			chunk := 1

			for i := range size {
				listParams[i], dashboardParams[i] = stats.RandomParams()

				if i > 0 && i%chunkSize == chunkSize-1 {
					stats.Print(size, chunk)
					stats.Reset()
					chunk++
				}
			}

			file := benchflix.Must(os.Create(fmt.Sprintf("data/params_%d.json", size)))

			if err := json.NewEncoder(file).Encode(dashboardParams); err != nil {
				b.Fatal(err)

				return
			}

			for _, nr := range repositories {
				b.Run(nr.Name, func(b *testing.B) {
					b.Run("List", func(b *testing.B) {
						execBench(b, nr, "List", listParams, chunkSize, func(r benchflix.Repository, params benchflix.ListParams) ([]benchflix.Movie, error) {
							return r.QueryList(context.Background(), params)
						})
					})

					b.Run("ListPreload", func(b *testing.B) {
						execBench(b, nr, "ListPreload", listParams, chunkSize, func(r benchflix.Repository, params benchflix.ListParams) ([]benchflix.Movie, error) {
							return r.QueryListPreload(context.Background(), params)
						})
					})

					b.Run("Dashboard", func(b *testing.B) {
						execBench(b, nr, "Dashboard", dashboardParams, chunkSize, func(r benchflix.Repository, params benchflix.DashboardParams) ([]benchflix.Movie, error) {
							return r.QueryDashboard(context.Background(), params)
						})
					})

					b.Run("DashboardPreload", func(b *testing.B) {
						execBench(b, nr, "DashboardPreload", dashboardParams, chunkSize, func(r benchflix.Repository, params benchflix.DashboardParams) ([]benchflix.Movie, error) {
							return r.QueryDashboardPreload(context.Background(), params)
						})
					})
				})
			}
		})
	}
}

func execBench[P any](b *testing.B, nr NamedRepository, szenario string, params []P, chunkSize int, fn func(r benchflix.Repository, params P) ([]benchflix.Movie, error)) {
	conn, resource := benchflix.InitializePostgres(nr.Name + "_" + szenario + "_" + strconv.Itoa(len(params)))

	defer resource.Close()

	r := nr.Repository(conn, MinConns, MaxConns, IdleTimeout)

	_, err := fn(r, params[0])
	if err != nil {
		if err == benchflix.ErrSkip {
			b.SkipNow()

			return
		}

		b.Fatal(err)

		return
	}

	var group errgroup.Group

	group.SetLimit(MaxConns * 2)

	for w := 0; w < MaxConns*2; w++ {
		group.Go(func() error {
			for i := 0; i < WarmupSize/(MaxConns*2); i++ {
				if _, err := fn(r, params[i%len(params)]); err != nil {
					return err
				}
			}
			return nil
		})
	}

	if err = group.Wait(); err != nil {
		b.Fatal(err)

		return
	}

	var chunkNr int

	for chunk := range slices.Chunk(params, chunkSize) {
		chunkNr++

		b.Run(fmt.Sprintf("Chunk-%d", chunkNr), func(b *testing.B) {
			b.RunParallel(func(pb *testing.PB) {
				i := 0

				for pb.Next() {
					_, err := fn(r, chunk[i%chunkSize])
					if err != nil {
						b.Fatal(err)

						return
					}

					i++
				}
			})
		})
	}

	_ = resource.Close()
}
