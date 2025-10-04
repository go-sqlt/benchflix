package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/go-sqlt/benchflix"
)

func main() {
	for _, size := range []int{500, 5_000} {
		chunkSize := size / 5

		if chunkSize*5 != size {
			panic("here")
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

		dashboardParams := []benchflix.DashboardParams{}

		file, err := os.Open(fmt.Sprintf("data/params_%d.json", size))
		if err != nil {
			panic(err)
		}

		if err = json.NewDecoder(file).Decode(&dashboardParams); err != nil {
			panic(err)
		}

		chunk := 1

		for i := range size {
			stats.Add(dashboardParams[i])

			if i > 0 && i%chunkSize == chunkSize-1 {
				stats.Print(size, chunk)
				stats.Reset()
				chunk++
			}
		}
	}
}
