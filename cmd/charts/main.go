package main

import (
	"fmt"
	"math"
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

	renderSzenario(b, "data/relative_latenz_list.png", "Relative Median-Latenz zu SQL - Szenario List", "List", benchflix.NsPerOp{}, 60, stats.Median)
	renderSzenario(b, "data/relativer_speicherbedarf_list.png", "Relativer Speicherbedarf – Szenario List", "List", benchflix.BytesPerOp{}, 40, stats.Median)

	renderSzenario(b, "data/relative_latenz_listpreload.png", "Relative Median-Latenz zu SQL - Szenario ListPreload", "ListPreload", benchflix.NsPerOp{}, 60, stats.Median)
	renderSzenario(b, "data/relativer_speicherbedarf_listpreload.png", "Relativer Speicherbedarf – Szenario ListPreload", "ListPreload", benchflix.BytesPerOp{}, 40, stats.Median)

	renderSzenario(b, "data/relative_latenz_dashboard.png", "Relative Median-Latenz zu SQL - Szenario Dashboard", "Dashboard", benchflix.NsPerOp{}, 60, stats.Median)
	renderSzenario(b, "data/relativer_speicherbedarf_dashboard.png", "Relativer Speicherbedarf – Szenario Dashboard", "Dashboard", benchflix.BytesPerOp{}, 40, stats.Median)

	renderSzenario(b, "data/relative_latenz_dashboardpreload.png", "Relative Median-Latenz zu SQL - Szenario DashboardPreload", "DashboardPreload", benchflix.NsPerOp{}, 60, stats.Median)
	renderSzenario(b, "data/relativer_speicherbedarf_dashboardpreload.png", "Relativer Speicherbedarf – Szenario DashboardPreload", "DashboardPreload", benchflix.BytesPerOp{}, 40, stats.Median)

	renderAll(b, "data/relative_latenz.png", "Relative Median-Latenz zu SQL", benchflix.NsPerOp{}, 60, stats.Median)
	renderAll(b, "data/relativer_speicherbedarf.png", "Relativer Speicherbedarf zu SQL", benchflix.BytesPerOp{}, 40, stats.Median)
}

func renderSzenario(b benchflix.Benchmark, output, title, szenario string, unit benchflix.Unit, minimum int, statFn func(input stats.Float64Data) (float64, error)) {
	frameworks := slices.DeleteFunc([]string{"SQLT-Cache", "SQLT", "SQLC", "GORM", "SQLX", "SQUIRREL", "PGX"}, func(f string) bool {
		_, ok := b["500"][f][szenario]

		return !ok
	})

	chart := charts.NewBar()
	chart.SetGlobalOptions(
		charts.WithTitleOpts(opts.Title{
			Title: title,
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

	for _, size := range []string{"500", "5000"} {
		minBar := []opts.BarData{}
		diffBar := []opts.BarData{}

		for _, f := range frameworks {
			var minVal float64 = 1000
			var maxVal float64 = 0

			for _, c := range []string{"1", "2", "3", "4", "5"} {
				val := benchflix.Must(statFn(unit.Unit(b[size][f][szenario][c]))) / benchflix.Must(statFn(unit.Unit(b[size]["SQL"][szenario][c]))) * 100
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

		chart.AddSeries(benchflix.Thousand(size), diffBar,
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

	if err := render.MakeChartSnapshot(chart.RenderContent(), output); err != nil {
		fmt.Println(chart)
		panic(err)
	}
}

func renderAll(
	b benchflix.Benchmark,
	output, title string,
	unit benchflix.Unit,
	minimum int,
	statFn func(input stats.Float64Data) (float64, error),
) {

	frameworks := []string{"SQLT-Cache", "SQLT", "SQLC", "GORM", "SQLX", "SQUIRREL", "PGX"}
	szenarien := []string{"List", "ListPreload", "Dashboard", "DashboardPreload"}
	sizes := []string{"500", "5000"}

	chart := charts.NewBar()
	chart.SetGlobalOptions(
		charts.WithTitleOpts(opts.Title{Title: title}),
		charts.WithAnimation(false),
		charts.WithInitializationOpts(opts.Initialization{BackgroundColor: "#FFFFFF"}),
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

	for _, szenario := range szenarien {
		minBar := make([]opts.BarData, 0, len(frameworks))
		diffBar := make([]opts.BarData, 0, len(frameworks))

		for _, f := range frameworks {
			minVal := math.Inf(1)
			maxVal := math.Inf(-1)

			for _, sz := range sizes {
				for _, chunk := range []string{"1", "2", "3", "4", "5"} {
					fv, okF := b[sz][f][szenario][chunk]
					sv, okS := b[sz]["SQL"][szenario][chunk]
					if !okF || !okS {
						continue
					}
					val := benchflix.Must(statFn(unit.Unit(fv))) /
						benchflix.Must(statFn(unit.Unit(sv))) * 100.0
					if val < minVal {
						minVal = val
					}
					if val > maxVal {
						maxVal = val
					}
				}
			}

			if math.IsInf(minVal, 1) || math.IsInf(maxVal, -1) {
				minVal = 0
				maxVal = 0
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
			s.Stack = "Stack-" + szenario
			s.ItemStyle = &opts.ItemStyle{Color: "transparent"}
			s.Emphasis = &opts.Emphasis{ItemStyle: &opts.ItemStyle{Color: "transparent"}}
			s.Name = ""
		}))

		chart.AddSeries(szenario, diffBar,
			charts.WithSeriesOpts(func(s *charts.SingleSeries) {
				s.Stack = "Stack-" + szenario
			}),
			charts.WithMarkLineStyleOpts(opts.MarkLineStyle{
				Symbol: []string{"none"},
				LineStyle: &opts.LineStyle{
					Color: "black",
					Type:  "dotted",
					Width: 1,
				},
				Label: &opts.Label{Formatter: "SQL"},
			}),
			charts.WithMarkLineNameXAxisItemOpts(opts.MarkLineNameXAxisItem{
				Name:  "SQL",
				XAxis: 100,
			}),
		)
	}

	if err := render.MakeChartSnapshot(chart.RenderContent(), output); err != nil {
		fmt.Println(chart)
		panic(err)
	}
}
