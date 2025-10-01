package main

import (
	"fmt"
	"math"
	"os"

	"github.com/go-sqlt/benchflix"
	"github.com/montanaflynn/stats"
)

func main() {
	b := benchflix.Must(benchflix.ReadAll(os.Stdin))

	Quartilsdispersionskoeffizient(b, "quartilsdispersionskoeffizient_latenz_sql", "Quartilsdispersionskoeffizient der Latenz von SQL in \\%", "SQL", benchflix.NsPerOp{})
	Quartilsdispersionskoeffizient(b, "quartilsdispersionskoeffizient_latenz_pgx", "Quartilsdispersionskoeffizient der Latenz von PGX in \\%", "PGX", benchflix.NsPerOp{})
	Quartilsdispersionskoeffizient(b, "quartilsdispersionskoeffizient_latenz_squirrel", "Quartilsdispersionskoeffizient der Latenz von SQUIRREL in \\%", "SQUIRREL", benchflix.NsPerOp{})
	Quartilsdispersionskoeffizient(b, "quartilsdispersionskoeffizient_latenz_sqlx", "Quartilsdispersionskoeffizient der Latenz von SQLX in \\%", "SQLX", benchflix.NsPerOp{})
	Quartilsdispersionskoeffizient(b, "quartilsdispersionskoeffizient_latenz_gorm", "Quartilsdispersionskoeffizient der Latenz von GORM in \\%", "GORM", benchflix.NsPerOp{})
	Quartilsdispersionskoeffizient(b, "quartilsdispersionskoeffizient_latenz_sqlc", "Quartilsdispersionskoeffizient der Latenz von SQLC in \\%", "SQLC", benchflix.NsPerOp{})
	Quartilsdispersionskoeffizient(b, "quartilsdispersionskoeffizient_latenz_sqlt", "Quartilsdispersionskoeffizient der Latenz von SQLT in \\%", "SQLT", benchflix.NsPerOp{})
	Quartilsdispersionskoeffizient(b, "quartilsdispersionskoeffizient_latenz_sqltcache", "Quartilsdispersionskoeffizient der Latenz von SQLT-Cache in \\%", "SQLT-Cache", benchflix.NsPerOp{})

	Quartilsdispersionskoeffizient(b, "quartilsdispersionskoeffizient_speicherverbrauch_sql", "Quartilsdispersionskoeffizient des Speicherverbrauchs von SQL in \\%", "SQL", benchflix.BytesPerOp{})
	Quartilsdispersionskoeffizient(b, "quartilsdispersionskoeffizient_speicherverbrauch_pgx", "Quartilsdispersionskoeffizient des Speicherverbrauchs von PGX in \\%", "PGX", benchflix.BytesPerOp{})
	Quartilsdispersionskoeffizient(b, "quartilsdispersionskoeffizient_speicherverbrauch_squirrel", "Quartilsdispersionskoeffizient des Speicherverbrauchs von SQUIRREL in \\%", "SQUIRREL", benchflix.BytesPerOp{})
	Quartilsdispersionskoeffizient(b, "quartilsdispersionskoeffizient_speicherverbrauch_sqlx", "Quartilsdispersionskoeffizient des Speicherverbrauchs von SQLX in \\%", "SQLX", benchflix.BytesPerOp{})
	Quartilsdispersionskoeffizient(b, "quartilsdispersionskoeffizient_speicherverbrauch_gorm", "Quartilsdispersionskoeffizient des Speicherverbrauchs von GORM in \\%", "GORM", benchflix.BytesPerOp{})
	Quartilsdispersionskoeffizient(b, "quartilsdispersionskoeffizient_speicherverbrauch_sqlc", "Quartilsdispersionskoeffizient des Speicherverbrauchs von SQLC in \\%", "SQLC", benchflix.BytesPerOp{})
	Quartilsdispersionskoeffizient(b, "quartilsdispersionskoeffizient_speicherverbrauch_sqlt", "Quartilsdispersionskoeffizient des Speicherverbrauchs von SQLT in \\%", "SQLT", benchflix.BytesPerOp{})
	Quartilsdispersionskoeffizient(b, "quartilsdispersionskoeffizient_speicherverbrauch_sqltcache", "Quartilsdispersionskoeffizient des Speicherverbrauchs von SQLT-Cache in \\%", "SQLT-Cache", benchflix.BytesPerOp{})

	Median(b, "latenz_sql", "Median-Latenz von SQL [ns/op]", "SQL", benchflix.NsPerOp{})
	Median(b, "latenz_pgx", "Median-Latenz von PGX [ns/op]", "PGX", benchflix.NsPerOp{})
	Median(b, "latenz_squirrel", "Median-Latenz von SQUIRREL [ns/op]", "SQUIRREL", benchflix.NsPerOp{})
	Median(b, "latenz_sqlx", "Median-Latenz von SQLX [ns/op]", "SQLX", benchflix.NsPerOp{})
	Median(b, "latenz_gorm", "Median-Latenz von GORM [ns/op]", "GORM", benchflix.NsPerOp{})
	Median(b, "latenz_sqlc", "Median-Latenz von SQLC [ns/op]", "SQLC", benchflix.NsPerOp{})
	Median(b, "latenz_sqlt", "Median-Latenz von SQLT [ns/op]", "SQLT", benchflix.NsPerOp{})
	Median(b, "latenz_sqltcache", "Median-Latenz von SQLT-Cache [ns/op]", "SQLT-Cache", benchflix.NsPerOp{})

	Median(b, "speicherverbrauch_sql", "Median-Speicherverbrauch von SQL [bytes/op]", "SQL", benchflix.BytesPerOp{})
	Median(b, "speicherverbrauch_pgx", "Median-Speicherverbrauch von PGX [bytes/op]", "PGX", benchflix.BytesPerOp{})
	Median(b, "speicherverbrauch_squirrel", "Median-Speicherverbrauch von SQUIRREL [bytes/op]", "SQUIRREL", benchflix.BytesPerOp{})
	Median(b, "speicherverbrauch_sqlx", "Median-Speicherverbrauch von SQLX [bytes/op]", "SQLX", benchflix.BytesPerOp{})
	Median(b, "speicherverbrauch_gorm", "Median-Speicherverbrauch von GORM [bytes/op]", "GORM", benchflix.BytesPerOp{})
	Median(b, "speicherverbrauch_sqlc", "Median-Speicherverbrauch von SQLC [bytes/op]", "SQLC", benchflix.BytesPerOp{})
	Median(b, "speicherverbrauch_sqlt", "Median-Speicherverbrauch von SQLT [bytes/op]", "SQLT", benchflix.BytesPerOp{})
	Median(b, "speicherverbrauch_sqltcache", "Median-Speicherverbrauch von SQLT-Cache [bytes/op]", "SQLT-Cache", benchflix.BytesPerOp{})

	RelativerVergleich(b, "relative_latenz_pgx", "Relative Median-Latenz von PGX zu SQL in \\%", "PGX", benchflix.NsPerOp{})
	RelativerVergleich(b, "relative_latenz_squirrel", "Relative Median-Latenz von SQUIRREL zu SQL in \\%", "SQUIRREL", benchflix.NsPerOp{})
	RelativerVergleich(b, "relative_latenz_sqlx", "Relative Median-Latenz von SQLX zu SQL in \\%", "SQLX", benchflix.NsPerOp{})
	RelativerVergleich(b, "relative_latenz_gorm", "Relative Median-Latenz von GORM zu SQL in \\%", "GORM", benchflix.NsPerOp{})
	RelativerVergleich(b, "relative_latenz_sqlc", "Relative Median-Latenz von SQLC zu SQL in \\%", "SQLC", benchflix.NsPerOp{})
	RelativerVergleich(b, "relative_latenz_sqlt", "Relative Median-Latenz von SQLT zu SQL in \\%", "SQLT", benchflix.NsPerOp{})
	RelativerVergleich(b, "relative_latenz_sqltcache", "Relative Median-Latenz von SQLT-Cache zu SQL in \\%", "SQLT-Cache", benchflix.NsPerOp{})

	RelativerVergleich(b, "relativer_speicherverbrauch_pgx", "Relativer Median-Speicherverbrauch von PGX zu SQL in \\%", "PGX", benchflix.BytesPerOp{})
	RelativerVergleich(b, "relativer_speicherverbrauch_squirrel", "Relativer Median-Speicherverbrauch von SQUIRREL zu SQL in \\%", "SQUIRREL", benchflix.BytesPerOp{})
	RelativerVergleich(b, "relativer_speicherverbrauch_sqlx", "Relativer Median-Speicherverbrauch von SQLX zu SQL in \\%", "SQLX", benchflix.BytesPerOp{})
	RelativerVergleich(b, "relativer_speicherverbrauch_gorm", "Relativer Median-Speicherverbrauch von GORM zu SQL in \\%", "GORM", benchflix.BytesPerOp{})
	RelativerVergleich(b, "relativer_speicherverbrauch_sqlc", "Relativer Median-Speicherverbrauch von SQLC zu SQL in \\%", "SQLC", benchflix.BytesPerOp{})
	RelativerVergleich(b, "relativer_speicherverbrauch_sqlt", "Relativer Median-Speicherverbrauch von SQLT zu SQL in \\%", "SQLT", benchflix.BytesPerOp{})
	RelativerVergleich(b, "relativer_speicherverbrauch_sqltcache", "Relativer Median-Speicherverbrauch von SQLT-Cache zu SQL in \\%", "SQLT-Cache", benchflix.BytesPerOp{})
}

func RelativerVergleich(b benchflix.Benchmark, output, title string, framework string, unit benchflix.Unit) {
	file := benchflix.Must(os.Create(fmt.Sprintf("data/%s.tex", output)))

	fmt.Fprintf(file, `
\begin{table}[ht]
\setlength{\extrarowheight}{-1pt}
\centering
\caption{%s}
\begin{tabular}{lrrrrrrr}
\toprule
Szenario & Parameter & ${\Delta \tilde{X}_1}$ & ${\Delta \tilde{X}_2}$ & ${\Delta \tilde{X}_3}$ & ${\Delta \tilde{X}_4}$ & ${\Delta \tilde{X}_5}$ \\
\midrule`, title)

	for _, szenario := range []string{"List", "ListPreload", "Dashboard", "DashboardPreload"} {
		for _, size := range []string{"500", "5000"} {
			s, ok := b[size][framework][szenario]
			if !ok {
				continue
			}

			sizedisplay := size

			if size == "5000" {
				sizedisplay = "5.000"
			}

			fmt.Fprintf(file, `
	%s & %s`,
				szenario, sizedisplay,
			)

			for _, chunk := range []string{"1", "2", "3", "4", "5"} {
				c, ok := s[chunk]
				if !ok {
					panic(fmt.Sprintf("%s %s %s %s", size, framework, szenario, chunk))
				}

				baseX := benchflix.Must(stats.Median(unit.Unit(b[size]["SQL"][szenario][chunk])))

				x := benchflix.Must(stats.Median(unit.Unit(c)))

				percent := x / baseX * 100

				fmt.Fprintf(file, ` & %s %.1f`, cellColor(percent), percent)
			}

			fmt.Fprintf(file, ` \\`)
		}
	}

	fmt.Fprintf(file, `
\bottomrule
\end{tabular}
\label{tab:%s}
\end{table}
	`, output)
}

func Median(b benchflix.Benchmark, output, title string, framework string, unit benchflix.Unit) {
	file := benchflix.Must(os.Create(fmt.Sprintf("data/%s.tex", output)))

	fmt.Fprintf(file, `
\begin{table}[ht]
\setlength{\extrarowheight}{-1pt}
\centering
\caption{%s}
\begin{tabular}{lrrrrrrr}
\toprule
Szenario & Parameter & ${\tilde{X}_1}$ & ${\tilde{X}_2}$ & ${\tilde{X}_3}$ & ${\tilde{X}_4}$ & ${\tilde{X}_5}$ \\
\midrule`, title)

	for _, szenario := range []string{"List", "ListPreload", "Dashboard", "DashboardPreload"} {
		for _, size := range []string{"500", "5000"} {
			s, ok := b[size][framework][szenario]
			if !ok {
				continue
			}

			sizedisplay := size

			if size == "5000" {
				sizedisplay = "5.000"
			}

			fmt.Fprintf(file, `
	%s & %s`,
				szenario, sizedisplay,
			)

			for _, chunk := range []string{"1", "2", "3", "4", "5"} {
				c, ok := s[chunk]
				if !ok {
					panic(fmt.Sprintf("%s %s %s %s", size, framework, szenario, chunk))
				}

				x := benchflix.Must(stats.Median(unit.Unit(c)))

				fmt.Fprintf(file, ` & %.f`, math.Round(x))
			}

			fmt.Fprintf(file, ` \\`)
		}
	}

	fmt.Fprintf(file, `
\bottomrule
\end{tabular}
\label{tab:%s}
\end{table}
	`, output)
}

func cellColor(percent float64) string {
	delta := percent - 100.0

	if delta < 0 {
		return fmt.Sprintf(`\cellcolor{green!%.0f} `, min(math.Abs(delta), 100)/2)
	} else {
		return fmt.Sprintf(`\cellcolor{red!%.0f} `, min(math.Abs(delta), 100)/2)
	}
}

func Quartilsdispersionskoeffizient(b benchflix.Benchmark, output, title string, framework string, unit benchflix.Unit) {
	file := benchflix.Must(os.Create(fmt.Sprintf("data/%s.tex", output)))

	fmt.Fprintf(file, `
\begin{table}[ht]
\setlength{\extrarowheight}{-1pt}
\centering
\caption{%s}
\begin{tabular}{lrrrrrrr}
\toprule
Szenario & Parameter & ${QDK_1}$ & ${QDK_2}$ & ${QDK_3}$& ${QDK_4}$ & ${QDK_5}$ \\
\midrule`, title)

	for _, szenario := range []string{"List", "ListPreload", "Dashboard", "DashboardPreload"} {
		for _, size := range []string{"500", "5000"} {
			s, ok := b[size][framework][szenario]
			if !ok {
				continue
			}

			sizedisplay := size

			if size == "5000" {
				sizedisplay = "5.000"
			}

			fmt.Fprintf(file, `
	%s & %s`,
				szenario, sizedisplay,
			)

			for _, chunk := range []string{"1", "2", "3", "4", "5"} {
				c, ok := s[chunk]
				if !ok {
					panic(fmt.Sprintf("%s %s %s %s", size, framework, szenario, chunk))
				}

				q2 := benchflix.Must(stats.Median(unit.Unit(c)))
				qa := benchflix.Must(stats.InterQuartileRange(unit.Unit(c)))

				cv := qa / q2 * 100

				fmt.Fprintf(file, ` & %s %.1f`,
					cellColor(cv+100), cv,
				)
			}

			fmt.Fprintf(file, ` \\`)
		}
	}

	fmt.Fprintf(file, `
\bottomrule
\end{tabular}
\label{tab:%s}
\end{table}
	`, output)
}
