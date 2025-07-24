package main

import (
	"encoding/json"
	"flag"
	"math/rand/v2"
	"os"

	"github.com/go-sqlt/benchflix"
)

func main() {
	size := flag.Int("size", 0, "number of params")
	flag.Parse()

	params := make([]benchflix.DashboardParams, *size)

	for i := range *size {
		params[i] = RandomDashboardParams()
	}

	if err := json.NewEncoder(os.Stdout).Encode(params); err != nil {
		panic(err)
	}
}

var (
	search        = []string{"", "the", "to", "of", "a", "little", "shark", "thing"}
	sort          = []string{"title", "added_at", "rating"}
	desc          = []bool{true, false}
	withDirectors = []bool{true, false}
)

func RandomDashboardParams() benchflix.DashboardParams {
	return benchflix.DashboardParams{
		Search:        search[rand.IntN(len(search))],
		YearAdded:     2000 + rand.Int64N(25),
		MinRating:     float64(rand.IntN(100)) / 10,
		Limit:         1 + rand.Uint64N(99),
		Sort:          sort[rand.IntN(len(sort))],
		Desc:          desc[rand.IntN(len(desc))],
		WithDirectors: withDirectors[rand.IntN(len(withDirectors))],
	}
}
