package main

import (
	"os"
	"strings"

	"github.com/go-echarts/go-echarts/v2/charts"
	"github.com/go-echarts/go-echarts/v2/opts"
	"github.com/go-echarts/snapshot-chromedp/render"
	"github.com/go-sqlt/benchflix"
	"github.com/montanaflynn/stats"
)

func main() {
	b := benchflix.Must(benchflix.ReadAll(os.Stdin))

	renderChart(b, "100 Params NsPerOp", func(s benchflix.Szenario) opts.BarData {
		return opts.BarData{Value: IgnoreErr(stats.Quartile(s.Hundred.NsPerOp)).Q2}
	})

	renderChart(b, "1000 Params NsPerOp", func(s benchflix.Szenario) opts.BarData {
		return opts.BarData{Value: IgnoreErr(stats.Quartile(s.Thousand.NsPerOp)).Q2}
	})

	renderChart(b, "100 Params BytesPerOp", func(s benchflix.Szenario) opts.BarData {
		return opts.BarData{Value: IgnoreErr(stats.Quartile(s.Hundred.BytesPerOp)).Q2}
	})

	renderChart(b, "1000 Params BytesPerOp", func(s benchflix.Szenario) opts.BarData {
		return opts.BarData{Value: IgnoreErr(stats.Quartile(s.Thousand.BytesPerOp)).Q2}
	})

	renderChart(b, "100 Params AllocsPerOp", func(s benchflix.Szenario) opts.BarData {
		return opts.BarData{Value: IgnoreErr(stats.Quartile(s.Hundred.AllocsPerOp)).Q2}
	})

	renderChart(b, "1000 Params AllocsPerOp", func(s benchflix.Szenario) opts.BarData {
		return opts.BarData{Value: IgnoreErr(stats.Quartile(s.Thousand.AllocsPerOp)).Q2}
	})
}

func renderChart(b benchflix.Benchmark, title string, fn func(benchflix.Szenario) opts.BarData) {
	chart := charts.NewBar()
	chart.SetGlobalOptions(
		charts.WithTitleOpts(opts.Title{
			Title: title,
		}),
		charts.WithAnimation(false),
		charts.WithInitializationOpts(opts.Initialization{
			BackgroundColor: "#FFFFFF",
		}),
	)

	chart.SetXAxis([]string{"SQL", "PGX", "SQUIRREL", "SQLX", "GORM", "SQLC", "SQLT", "SQLT-Cache"})

	chart.AddSeries("List", []opts.BarData{
		fn(b.SQL.List),
		fn(b.PGX.List),
		fn(b.SQUIRREL.List),
		fn(b.SQLX.List),
		fn(b.GORM.List),
		fn(b.SQLC.List),
		fn(b.SQLT.List),
		fn(b.SQLTCACHE.List),
	})

	chart.AddSeries("ListPreload", []opts.BarData{
		fn(b.SQL.ListPreload),
		fn(b.PGX.ListPreload),
		fn(b.SQUIRREL.ListPreload),
		fn(b.SQLX.ListPreload),
		fn(b.GORM.ListPreload),
		fn(b.SQLC.ListPreload),
		fn(b.SQLT.ListPreload),
		fn(b.SQLTCACHE.ListPreload),
	})

	chart.AddSeries("Dashboard", []opts.BarData{
		fn(b.SQL.Dashboard),
		fn(b.PGX.Dashboard),
		fn(b.SQUIRREL.Dashboard),
		fn(b.SQLX.Dashboard),
		fn(b.GORM.Dashboard),
		fn(b.SQLC.Dashboard),
		fn(b.SQLT.Dashboard),
		fn(b.SQLTCACHE.Dashboard),
	})

	chart.AddSeries("DashboardPreload", []opts.BarData{
		fn(b.SQL.DashboardPreload),
		fn(b.PGX.DashboardPreload),
		fn(b.SQUIRREL.DashboardPreload),
		fn(b.SQLX.DashboardPreload),
		fn(b.GORM.DashboardPreload),
		fn(b.SQLC.DashboardPreload),
		fn(b.SQLT.DashboardPreload),
		fn(b.SQLTCACHE.DashboardPreload),
	})

	output := "data/" + strings.ReplaceAll(title, " ", "_") + ".png"

	if err := render.MakeChartSnapshot(chart.RenderContent(), output); err != nil {
		panic(err)
	}
}

func IgnoreErr(q stats.Quartiles, err error) stats.Quartiles {
	return q
}
