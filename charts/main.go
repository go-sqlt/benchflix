package main

import (
	"bufio"
	"flag"
	"fmt"
	"math"
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
	title = flag.String("title", "", "Chart Title")
)

type Data struct {
	Name                       string
	NsPerOp                    []float64
	AllocedBytesPerOp          []float64
	AllocsPerOp                []float64
	NsPerOpQuartiles           [3]float64
	AllocedBytesPerOpQuartiles [3]float64
	AllocsPerOpQuartiles       [3]float64
}

func main() {
	flag.Parse()

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

		parts := strings.Split(b.Name, "/")
		name := parts[1]

		index := slices.IndexFunc(data, func(d Data) bool { return d.Name == name })
		if index < 0 {
			data = append(data, Data{
				Name:              name,
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
	renderChart(data, "AllocedBytesPerOp", func(d Data) []float64 { return d.AllocedBytesPerOp })
	renderChart(data, "AllocsPerOp", func(d Data) []float64 { return d.AllocsPerOp })
}

func renderChart(data []Data, suffix string, fn func(Data) []float64) {
	chart := charts.NewBar()
	chart.SetGlobalOptions(
		charts.WithTitleOpts(opts.Title{
			Title: *title + " " + suffix,
		}),
		charts.WithAnimation(false),
		charts.WithInitializationOpts(opts.Initialization{
			BackgroundColor: "#FFFFFF",
		}),
		charts.WithXAxisOpts(opts.XAxis{Position: "bottom"}),
	)

	q2 := []opts.BarData{}
	xaxis := []string{}

	for _, d := range data {
		q, err := stats.Quartile(fn(d))
		if err != nil {
			panic(err)
		}

		q2 = append(q2, opts.BarData{
			Name:  d.Name,
			Value: strconv.FormatFloat(q.Q2, 'f', 1, 64),
		})

		xaxis = append(xaxis, d.Name+fmt.Sprintf(`
%d
%d
%d
%d
				`, int(math.Round(q.Q1)), int(math.Round(q.Q2)), int(math.Round(q.Q3)), int(math.Round(q.Q3-q.Q1))))
	}

	chart.AddSeries("2. Quartil", q2)
	chart.SetXAxis(xaxis)

	output := "charts/" + strings.ReplaceAll(*title, " ", "_") + "_" + suffix + ".png"

	if err := render.MakeChartSnapshot(chart.RenderContent(), output); err != nil {
		panic(err)
	}
}
