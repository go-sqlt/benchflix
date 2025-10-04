package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/go-sqlt/benchflix"
)

func main() {
	for _, size := range []int{500, 5_000} {
		dashboardParams := make([]benchflix.DashboardParams, size)

		for i := range size {
			dashboardParams[i] = benchflix.Random()
		}

		file := benchflix.Must(os.Create(fmt.Sprintf("data/params_%d.json", size)))

		if err := json.NewEncoder(file).Encode(dashboardParams); err != nil {
			panic(err)
		}

	}
}
