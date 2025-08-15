package benchflix

import (
	"bufio"
	"context"
	"database/sql"
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/rand/v2"
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
	ID        int64     `db:"id" json:"id"`
	Title     string    `db:"title" json:"title"`
	AddedAt   time.Time `db:"added_at" json:"added_at"`
	Rating    float64   `db:"rating" json:"rating"`
	Directors []string  `db:"directors" json:"directors"`
}

type ListParams struct {
	Search    string  `db:"search" json:"search"`
	YearAdded int64   `db:"year_added" json:"year_added"`
	MinRating float64 `db:"min_rating" json:"min_rating"`
	Limit     uint64  `db:"limit" json:"limit"`
}

type DashboardParams struct {
	Search        string  `db:"search" json:"search"`
	YearAdded     int64   `db:"year_added" json:"year_added"`
	MinRating     float64 `db:"min_rating" json:"min_rating"`
	Limit         uint64  `db:"limit" json:"limit"`
	Sort          string  `db:"sort" json:"sort"`
	Desc          bool    `db:"desc" json:"desc"`
	WithDirectors bool    `db:"with_directors" json:"with_directors"`
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

type Benchmark map[string]Size

type Size map[string]Framework

type Framework map[string]Szenario

type Szenario map[string]Chunk

type Chunk struct {
	NsPerOp, BytesPerOp, AllocsPerOp []float64
}

type Unit interface {
	Name() string
	Unit(c Chunk) []float64
}

type NsPerOp struct{}

func (NsPerOp) Name() string {
	return "Ns"
}

func (NsPerOp) Unit(c Chunk) []float64 {
	return c.NsPerOp
}

type BytesPerOp struct{}

func (BytesPerOp) Name() string {
	return "Bytes"
}

func (BytesPerOp) Unit(c Chunk) []float64 {
	return c.BytesPerOp
}

type AllocsPerOp struct{}

func (AllocsPerOp) Name() string {
	return "Allocs"
}

func (AllocsPerOp) Unit(c Chunk) []float64 {
	return c.AllocsPerOp
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

		size := strings.TrimPrefix(parts[1], "Size-")
		framework := parts[2]
		szenario := parts[3]
		chunk := strings.TrimPrefix(strings.TrimSuffix(parts[4], "-12"), "Chunk-")

		if _, ok := bench[size]; !ok {
			bench[size] = Size{}
		}

		if _, ok := bench[size][framework]; !ok {
			bench[size][framework] = Framework{}
		}

		if _, ok := bench[size][framework][szenario]; !ok {
			bench[size][framework][szenario] = Szenario{}
		}

		c, ok := bench[size][framework][szenario][chunk]
		if !ok {
			c = Chunk{}
		}

		c.NsPerOp = append(c.NsPerOp, b.NsPerOp)
		c.AllocsPerOp = append(c.AllocsPerOp, float64(b.AllocsPerOp))
		c.BytesPerOp = append(c.BytesPerOp, float64(b.AllocedBytesPerOp))

		bench[size][framework][szenario][chunk] = c
	}

	return bench, nil
}

var (
	Search        = []string{"", "the", "of", "a", "s", "in", "and", "to", "love", "my", "man", "girl", "one", "for"}
	Sort          = []string{"title", "added_at", "rating"}
	Desc          = []bool{true, false}
	WithDirectors = []bool{true, false}
	MinRating     = []float64{0, 0.5, 1, 1.5, 2, 2.5, 3, 3.5, 4, 4.5, 5, 5.5, 6, 6.5, 7, 7.5, 8, 8.5, 9}
	Limit         = []uint64{5, 10, 15, 20, 25, 30, 35, 40, 45, 50, 55, 60, 65, 70, 75, 80, 85, 90, 95, 100}
	YearAdded     = []int64{
		0,
		2001, 2002, 2003, 2004, 2005, 2006, 2007, 2008, 2009, 2010,
		2011, 2012, 2013, 2014, 2015, 2016, 2017, 2018, 2019, 2020,
		2021, 2022, 2023, 2024,
	}
)

type Stats struct {
	Search        map[string]int `json:"search"`
	YearAdded     map[string]int `json:"year_added"`
	MinRating     map[string]int `json:"min_rating"`
	Limit         map[string]int `json:"limit"`
	Sort          map[string]int `json:"sort"`
	Desc          map[string]int `json:"desc"`
	WithDirectors map[string]int `json:"with_directors"`
}

func (s *Stats) Reset() {
	*s = Stats{
		Search:        map[string]int{},
		Sort:          map[string]int{},
		Desc:          map[string]int{},
		WithDirectors: map[string]int{},
		Limit:         map[string]int{},
		MinRating:     map[string]int{},
		YearAdded:     map[string]int{},
	}
}

func (s Stats) Print(size, i int) {
	file := Must(os.Create(fmt.Sprintf("data/stats_%d_%d.json", size, i)))

	if err := json.NewEncoder(file).Encode(s); err != nil {
		panic(err)
	}
}

func (stats *Stats) RandomParams() (ListParams, DashboardParams) {
	searchValue := Search[rand.IntN(len(Search))]
	yearAddedValue := YearAdded[rand.IntN(len(YearAdded))]
	minRatingValue := MinRating[rand.IntN(len(MinRating))]
	limitValue := Limit[rand.IntN(len(Limit))]
	sortValue := Sort[rand.IntN(len(Sort))]
	descValue := Desc[rand.IntN(len(Desc))]
	withDirectorsValue := WithDirectors[rand.IntN(len(WithDirectors))]

	stats.Search[searchValue]++
	stats.YearAdded[strconv.FormatInt(yearAddedValue, 10)]++
	stats.MinRating[strconv.FormatFloat(minRatingValue, 'f', 1, 64)]++
	stats.Limit[strconv.FormatUint(limitValue, 10)]++
	stats.Sort[sortValue]++
	stats.Desc[strconv.FormatBool(descValue)]++
	stats.WithDirectors[strconv.FormatBool(withDirectorsValue)]++

	return ListParams{
			Search:    searchValue,
			YearAdded: yearAddedValue,
			MinRating: minRatingValue,
			Limit:     limitValue,
		}, DashboardParams{
			Search:        searchValue,
			YearAdded:     yearAddedValue,
			MinRating:     minRatingValue,
			Limit:         limitValue,
			Sort:          sortValue,
			Desc:          descValue,
			WithDirectors: withDirectorsValue,
		}
}
