package main

import (
	"fmt"
	"os"
	"slices"

	"github.com/go-echarts/go-echarts/v2/charts"
	"github.com/go-echarts/go-echarts/v2/opts"
	"github.com/go-echarts/snapshot-chromedp/render"
	"github.com/go-sqlt/benchflix"
	"github.com/montanaflynn/stats"
)

func main() {
	b := benchflix.Must(benchflix.ReadAll(os.Stdin))

	renderChart(b, "List", benchflix.NsPerOp{}, 60)
	renderChart(b, "List", benchflix.BytesPerOp{}, 40)
	renderChart(b, "List", benchflix.AllocsPerOp{}, 40)

	renderChart(b, "ListPreload", benchflix.NsPerOp{}, 60)
	renderChart(b, "ListPreload", benchflix.BytesPerOp{}, 40)
	renderChart(b, "ListPreload", benchflix.AllocsPerOp{}, 40)

	renderChart(b, "Dashboard", benchflix.NsPerOp{}, 60)
	renderChart(b, "Dashboard", benchflix.BytesPerOp{}, 40)
	renderChart(b, "Dashboard", benchflix.AllocsPerOp{}, 40)

	renderChart(b, "DashboardPreload", benchflix.NsPerOp{}, 60)
	renderChart(b, "DashboardPreload", benchflix.BytesPerOp{}, 40)
	renderChart(b, "DashboardPreload", benchflix.AllocsPerOp{}, 40)
}

func renderChart(b benchflix.Benchmark, szenario string, unit benchflix.Unit, minimum int) {
	frameworks := slices.DeleteFunc([]string{"PGX", "SQUIRREL", "SQLX", "GORM", "SQLC", "SQLT", "SQLT-Cache"}, func(f string) bool {
		_, ok := b["100"][f][szenario]

		return !ok
	})

	chart := charts.NewBar()
	chart.SetGlobalOptions(
		charts.WithTitleOpts(opts.Title{
			Title: fmt.Sprintf("Szenario %s: Vergleich (%s) in %%", szenario, unit.Name()),
		}),
		charts.WithAnimation(false),
		charts.WithInitializationOpts(opts.Initialization{
			BackgroundColor: "#FFFFFF",
		}),
		charts.WithYAxisOpts(opts.YAxis{
			Type: "category",
			Data: frameworks,
		}),
		charts.WithXAxisOpts(opts.XAxis{
			Type: "value",
			Min:  minimum,
		}),
		charts.WithLegendOpts(opts.Legend{
			Right:  "0",
			Orient: "vertical",
		}),
	)

	for _, size := range []string{"100", "1000", "10000"} {
		minBar := []opts.BarData{}
		diffBar := []opts.BarData{}

		for _, f := range frameworks {
			var minVal float64 = 1000
			var maxVal float64 = 0

			for _, c := range []string{"1", "2", "3", "4", "5"} {
				val := benchflix.Must(stats.Mean(unit.Unit(b[size][f][szenario][c]))) / benchflix.Must(stats.Mean(unit.Unit(b[size]["SQL"][szenario][c]))) * 100
				minVal = min(val, minVal)
				maxVal = max(val, maxVal)
			}

			minBar = append(minBar, opts.BarData{
				Name:  f,
				Value: minVal,
			})

			diffBar = append(diffBar, opts.BarData{
				Name:  f,
				Value: maxVal - minVal,
			})
		}

		chart.AddSeries("", minBar, charts.WithSeriesOpts(func(s *charts.SingleSeries) {
			s.Stack = fmt.Sprintf("Stack-%s", size)
			s.ItemStyle = &opts.ItemStyle{
				Color: "transparent",
			}
			s.Emphasis = &opts.Emphasis{
				ItemStyle: &opts.ItemStyle{
					Color: "transparent",
				},
			}
		}))

		chart.AddSeries(size, diffBar,
			charts.WithSeriesOpts(func(s *charts.SingleSeries) {
				s.Stack = fmt.Sprintf("Stack-%s", size)
			}),
			charts.WithMarkLineStyleOpts(opts.MarkLineStyle{
				Symbol: []string{"none"},
				LineStyle: &opts.LineStyle{
					Color: "black",
					Type:  "dotted", // dotted style
					Width: 1,
				},
				Label: &opts.Label{
					Formatter: "SQL",
				},
			}),
			charts.WithMarkLineNameXAxisItemOpts(
				opts.MarkLineNameXAxisItem{
					Name:  "SQL",
					XAxis: 100,
				},
			))
	}

	output := fmt.Sprintf("data/chart_%s_%s.png", szenario, unit.Name())

	if err := render.MakeChartSnapshot(chart.RenderContent(), output); err != nil {
		fmt.Println(chart)
		panic(err)
	}
}
