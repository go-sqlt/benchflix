package main

import (
	"bufio"
	"flag"
	"os"
	"slices"
	"strconv"
	"strings"

	"github.com/go-echarts/go-echarts/v2/charts"
	"github.com/go-echarts/go-echarts/v2/opts"
	"github.com/go-echarts/snapshot-chromedp/render"
	"github.com/montanaflynn/stats"
	"golang.org/x/tools/benchmark/parse"
)

var (
	title      = flag.String("title", "", "Chart Title")
	cores      = flag.Int("cores", 12, "Number of cores")
	frameworks = flag.String("frameworks", "", "X-Axis Frameworks")
	FRAMEWORKS []string
)

type Data struct {
	Name                       string
	Framework                  string
	Szenario                   string
	NsPerOp                    []float64
	AllocedBytesPerOp          []float64
	AllocsPerOp                []float64
	NsPerOpQuartiles           [3]float64
	AllocedBytesPerOpQuartiles [3]float64
	AllocsPerOpQuartiles       [3]float64
}

func main() {
	flag.Parse()

	FRAMEWORKS = strings.Split(*frameworks, ",")

	scan := bufio.NewScanner(os.Stdin)

	var data []Data

	for scan.Scan() {
		line := scan.Text()

		if !strings.HasPrefix(line, "Benchmark") {
			continue
		}

		b, err := parse.ParseLine(line)
		if err != nil {
			panic(err)
		}

		index := slices.IndexFunc(data, func(d Data) bool { return d.Name == b.Name })
		if index < 0 {
			parts := strings.Split(b.Name, "/")

			data = append(data, Data{
				Name:              b.Name,
				Framework:         parts[1],
				Szenario:          strings.TrimSuffix(parts[2], "-"+strconv.Itoa(*cores)),
				NsPerOp:           []float64{b.NsPerOp},
				AllocedBytesPerOp: []float64{float64(b.AllocedBytesPerOp)},
				AllocsPerOp:       []float64{float64(b.AllocsPerOp)},
			})

			continue
		}

		data[index].NsPerOp = append(data[index].NsPerOp, b.NsPerOp)
		data[index].AllocedBytesPerOp = append(data[index].AllocedBytesPerOp, float64(b.AllocedBytesPerOp))
		data[index].AllocsPerOp = append(data[index].AllocsPerOp, float64(b.AllocsPerOp))
	}

	renderChart(data, "NsPerOp", func(d Data) []float64 { return d.NsPerOp })
	renderChart(data, "BytesPerOp", func(d Data) []float64 { return d.AllocedBytesPerOp })
	renderChart(data, "AllocsPerOp", func(d Data) []float64 { return d.AllocsPerOp })
}

func renderChart(data []Data, suffix string, fn func(Data) []float64) {
	list := make([]opts.BarData, len(FRAMEWORKS))
	listpreload := make([]opts.BarData, len(FRAMEWORKS))
	dashboard := make([]opts.BarData, len(FRAMEWORKS))
	dashboardpreload := make([]opts.BarData, len(FRAMEWORKS))
	xSet := map[string]bool{}

	for _, d := range data {
		xSet[d.Framework] = true

		q, err := stats.Quartile(fn(d))
		if err != nil {
			panic(err)
		}

		index := slices.Index(FRAMEWORKS, d.Framework)

		switch d.Szenario {
		case "List":
			list[index] = opts.BarData{
				Name:  d.Framework,
				Value: q.Q2,
			}
		case "ListPreload":
			listpreload[index] = opts.BarData{
				Name:  d.Framework,
				Value: q.Q2,
			}
		case "Dashboard":
			dashboard[index] = opts.BarData{
				Name:  d.Framework,
				Value: q.Q2,
			}
		case "DashboardPreload":
			dashboardpreload[index] = opts.BarData{
				Name:  d.Framework,
				Value: q.Q2,
			}
		}
	}

	chart := charts.NewBar()
	chart.SetGlobalOptions(
		charts.WithTitleOpts(opts.Title{
			Title: *title + " " + suffix,
		}),
		charts.WithAnimation(false),
		charts.WithInitializationOpts(opts.Initialization{
			BackgroundColor: "#FFFFFF",
		}),
	)

	if len(list) > 0 {
		chart.AddSeries("List", list)
	}
	if len(listpreload) > 0 {
		chart.AddSeries("ListPreload", listpreload)
	}
	if len(dashboard) > 0 {
		chart.AddSeries("Dashboard", dashboard)
	}
	if len(dashboardpreload) > 0 {
		chart.AddSeries("DashboardPreload", dashboardpreload)
	}

	chart.SetXAxis(FRAMEWORKS)

	output := "charts/" + strings.ReplaceAll(*title, " ", "_") + "_" + suffix + ".png"

	if err := render.MakeChartSnapshot(chart.RenderContent(), output); err != nil {
		panic(err)
	}
}
