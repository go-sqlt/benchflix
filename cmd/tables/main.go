package main

import (
	"fmt"
	"math"
	"os"
	"strings"

	"github.com/go-sqlt/benchflix"
	"github.com/montanaflynn/stats"
)

func main() {
	b := benchflix.Must(benchflix.ReadAll(os.Stdin))

	Variationskoeffizient(b, "SQL")
	Variationskoeffizient(b, "PGX")
	Variationskoeffizient(b, "SQUIRREL")
	Variationskoeffizient(b, "SQLX")
	Variationskoeffizient(b, "GORM")
	Variationskoeffizient(b, "SQLC")
	Variationskoeffizient(b, "SQLT")
	Variationskoeffizient(b, "SQLT-Cache")

	Mittelwerte(b, "SQL", benchflix.NsPerOp{})
	Mittelwerte(b, "SQL", benchflix.BytesPerOp{})
	Mittelwerte(b, "SQL", benchflix.AllocsPerOp{})
	Mittelwerte(b, "PGX", benchflix.NsPerOp{})
	Mittelwerte(b, "PGX", benchflix.BytesPerOp{})
	Mittelwerte(b, "PGX", benchflix.AllocsPerOp{})
	Mittelwerte(b, "SQUIRREL", benchflix.NsPerOp{})
	Mittelwerte(b, "SQUIRREL", benchflix.BytesPerOp{})
	Mittelwerte(b, "SQUIRREL", benchflix.AllocsPerOp{})
	Mittelwerte(b, "SQLX", benchflix.NsPerOp{})
	Mittelwerte(b, "SQLX", benchflix.BytesPerOp{})
	Mittelwerte(b, "SQLX", benchflix.AllocsPerOp{})
	Mittelwerte(b, "GORM", benchflix.NsPerOp{})
	Mittelwerte(b, "GORM", benchflix.BytesPerOp{})
	Mittelwerte(b, "GORM", benchflix.AllocsPerOp{})
	Mittelwerte(b, "SQLC", benchflix.NsPerOp{})
	Mittelwerte(b, "SQLC", benchflix.BytesPerOp{})
	Mittelwerte(b, "SQLC", benchflix.AllocsPerOp{})
	Mittelwerte(b, "SQLT", benchflix.NsPerOp{})
	Mittelwerte(b, "SQLT", benchflix.BytesPerOp{})
	Mittelwerte(b, "SQLT", benchflix.AllocsPerOp{})
	Mittelwerte(b, "SQLT-Cache", benchflix.NsPerOp{})
	Mittelwerte(b, "SQLT-Cache", benchflix.BytesPerOp{})
	Mittelwerte(b, "SQLT-Cache", benchflix.AllocsPerOp{})

	Benchmarkdifferenz(b, "PGX", benchflix.NsPerOp{})
	Benchmarkdifferenz(b, "SQUIRREL", benchflix.NsPerOp{})
	Benchmarkdifferenz(b, "SQLX", benchflix.NsPerOp{})
	Benchmarkdifferenz(b, "GORM", benchflix.NsPerOp{})
	Benchmarkdifferenz(b, "SQLC", benchflix.NsPerOp{})
	Benchmarkdifferenz(b, "SQLT", benchflix.NsPerOp{})
	Benchmarkdifferenz(b, "SQLT-Cache", benchflix.NsPerOp{})

	Benchmarkdifferenz(b, "PGX", benchflix.BytesPerOp{})
	Benchmarkdifferenz(b, "SQUIRREL", benchflix.BytesPerOp{})
	Benchmarkdifferenz(b, "SQLX", benchflix.BytesPerOp{})
	Benchmarkdifferenz(b, "GORM", benchflix.BytesPerOp{})
	Benchmarkdifferenz(b, "SQLC", benchflix.BytesPerOp{})
	Benchmarkdifferenz(b, "SQLT", benchflix.BytesPerOp{})
	Benchmarkdifferenz(b, "SQLT-Cache", benchflix.BytesPerOp{})

	Benchmarkdifferenz(b, "PGX", benchflix.AllocsPerOp{})
	Benchmarkdifferenz(b, "SQUIRREL", benchflix.AllocsPerOp{})
	Benchmarkdifferenz(b, "SQLX", benchflix.AllocsPerOp{})
	Benchmarkdifferenz(b, "GORM", benchflix.AllocsPerOp{})
	Benchmarkdifferenz(b, "SQLC", benchflix.AllocsPerOp{})
	Benchmarkdifferenz(b, "SQLT", benchflix.AllocsPerOp{})
	Benchmarkdifferenz(b, "SQLT-Cache", benchflix.AllocsPerOp{})
}

func Variationskoeffizient(b benchflix.Benchmark, framework string) {
	file := benchflix.Must(os.Create(fmt.Sprintf("data/variationskoeffizienten_%s.tex", strings.ToLower(framework))))

	fmt.Fprintf(file, `
\begin{table}[ht]
\setlength{\extrarowheight}{-1pt}
\centering
\caption{%s: Variantionskoeffizienten in \%%}
\begin{tabular}{llrrrrrr}
\toprule
Szenario & Einheit & Parameter & ${CV_1}$ & ${CV_2}$ & ${CV_3}$& ${CV_4}$ & ${CV_5}$ \\
\midrule`, framework)

	for _, szenario := range []string{"List", "ListPreload", "Dashboard", "DashboardPreload"} {
		for _, size := range []string{"100", "1000", "10000"} {
			s, ok := b[size][framework][szenario]
			if !ok {
				continue
			}

			for _, unit := range []benchflix.Unit{benchflix.NsPerOp{}, benchflix.BytesPerOp{}, benchflix.AllocsPerOp{}} {
				fmt.Fprintf(file, `
	%s & %s & %s`,
					szenario, unit.Name(), size,
				)

				for _, chunk := range []string{"1", "2", "3", "4", "5"} {
					c, ok := s[chunk]
					if !ok {
						panic(fmt.Sprintf("%s %s %s %s", size, framework, szenario, chunk))
					}

					nsX := benchflix.Must(stats.Mean(unit.Unit(c)))
					nsSD := benchflix.Must(stats.StandardDeviation(unit.Unit(c)))

					cv := nsSD / nsX * 100

					fmt.Fprintf(file, ` & %s %.1f`,
						cellColor(cv+100), cv,
					)
				}

				fmt.Fprintf(file, ` \\`)
			}
		}
	}

	fmt.Fprintf(file, `
\bottomrule
\end{tabular}
\label{tab:variationskoeffizienten_%s}
\end{table}
	`, strings.ToLower(framework))
}

func Benchmarkdifferenz(b benchflix.Benchmark, framework string, unit benchflix.Unit) {
	file := benchflix.Must(os.Create(fmt.Sprintf("data/benchmarkdifferenz_%s_%s.tex", strings.ToLower(framework), strings.ToLower(unit.Name()))))

	fmt.Fprintf(file, `
\begin{table}[ht]
\setlength{\extrarowheight}{-1pt}
\centering
\caption{%s: Benchmarkdifferenz (%s) in \%%}
\begin{tabular}{llrrrrrr}
\toprule
Szenario & Parameter & ${\Delta \overline{X}_1}$ & ${\Delta \overline{X}_2}$ & ${\Delta \overline{X}_3}$ & ${\Delta \overline{X}_4}$ & ${\Delta \overline{X}_5}$ \\
\midrule`, framework, unit.Name())

	for _, szenario := range []string{"List", "ListPreload", "Dashboard", "DashboardPreload"} {
		for _, size := range []string{"100", "1000", "10000"} {
			s, ok := b[size][framework][szenario]
			if !ok {
				continue
			}

			fmt.Fprintf(file, `
	%s & %s`,
				szenario, size,
			)

			for _, chunk := range []string{"1", "2", "3", "4", "5"} {
				c, ok := s[chunk]
				if !ok {
					panic(fmt.Sprintf("%s %s %s %s", size, framework, szenario, chunk))
				}

				baseX := benchflix.Must(stats.Mean(unit.Unit(b[size]["SQL"][szenario][chunk])))

				x := benchflix.Must(stats.Mean(unit.Unit(c)))

				percent := x / baseX * 100

				fmt.Fprintf(file, ` & %s %.1f`, cellColor(percent), percent)
			}

			fmt.Fprintf(file, ` \\`)
		}
	}

	fmt.Fprintf(file, `
\bottomrule
\end{tabular}
\label{tab:benchmarkdifferenz_%s_%s}
\end{table}
	`, strings.ToLower(framework), strings.ToLower(unit.Name()))
}

func Mittelwerte(b benchflix.Benchmark, framework string, unit benchflix.Unit) {
	file := benchflix.Must(os.Create(fmt.Sprintf("data/mittelwerte_%s_%s.tex", strings.ToLower(framework), strings.ToLower(unit.Name()))))

	fmt.Fprintf(file, `
\begin{table}[ht]
\setlength{\extrarowheight}{-1pt}
\centering
\caption{%s: Mittelwerte (%s)}
\begin{tabular}{llrrrrrr}
\toprule
Szenario & Parameter & ${\Delta \overline{X}_1}$ & ${\Delta \overline{X}_2}$ & ${\Delta \overline{X}_3}$ & ${\Delta \overline{X}_4}$ & ${\Delta \overline{X}_5}$ \\
\midrule`, framework, unit.Name())

	for _, szenario := range []string{"List", "ListPreload", "Dashboard", "DashboardPreload"} {
		for _, size := range []string{"100", "1000", "10000"} {
			s, ok := b[size][framework][szenario]
			if !ok {
				continue
			}

			fmt.Fprintf(file, `
	%s & %s`,
				szenario, size,
			)

			for _, chunk := range []string{"1", "2", "3", "4", "5"} {
				c, ok := s[chunk]
				if !ok {
					panic(fmt.Sprintf("%s %s %s %s", size, framework, szenario, chunk))
				}

				x := benchflix.Must(stats.Mean(unit.Unit(c)))

				fmt.Fprintf(file, ` & %.f`, math.Round(x))
			}

			fmt.Fprintf(file, ` \\`)
		}
	}

	fmt.Fprintf(file, `
\bottomrule
\end{tabular}
\label{tab:mittelwerte_%s_%s}
\end{table}
	`, strings.ToLower(framework), strings.ToLower(unit.Name()))
}

func cellColor(percent float64) string {
	delta := percent - 100.0
	ad := math.Abs(delta)

	if ad < 2.0 {
		return ""
	}

	mapIntensity := func(v, maxDev, minInt, maxInt float64) float64 {
		if v < 0 {
			v = 0
		}
		if v > maxDev {
			v = maxDev
		}

		return minInt + v/maxDev*(maxInt-minInt)
	}

	if delta < 0 {
		intensity := mapIntensity(-delta, 50.0, 10.0, 30.0)
		return fmt.Sprintf(`\cellcolor{green!%.0f} `, intensity)
	} else {
		intensity := mapIntensity(delta, 50.0, 10.0, 30.0)
		return fmt.Sprintf(`\cellcolor{red!%.0f} `, intensity)
	}
}
