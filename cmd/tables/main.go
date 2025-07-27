package main

import (
	"fmt"
	"io"
	"math"
	"os"
	"strings"

	"github.com/go-sqlt/benchflix"
	"github.com/montanaflynn/stats"
)

func main() {
	b := benchflix.Must(benchflix.ReadAll(os.Stdin))

	PrintNsPerOp("SQL", b.SQL)
	PrintNsPerOp("PGX", b.PGX)
	PrintNsPerOp("SQUIRREL", b.SQUIRREL)
	PrintNsPerOp("SQLX", b.SQLX)
	PrintNsPerOp("GORM", b.GORM)
	PrintNsPerOp("SQLC", b.SQLC)
	PrintNsPerOp("SQLT", b.SQLT)
	PrintNsPerOp("SQLT-Cache", b.SQLTCACHE)

	PrintBytesPerOp("SQL", b.SQL)
	PrintBytesPerOp("PGX", b.PGX)
	PrintBytesPerOp("SQUIRREL", b.SQUIRREL)
	PrintBytesPerOp("SQLX", b.SQLX)
	PrintBytesPerOp("GORM", b.GORM)
	PrintBytesPerOp("SQLC", b.SQLC)
	PrintBytesPerOp("SQLT", b.SQLT)
	PrintBytesPerOp("SQLT-Cache", b.SQLTCACHE)

	PrintAllocsPerOp("SQL", b.SQL)
	PrintAllocsPerOp("PGX", b.PGX)
	PrintAllocsPerOp("SQUIRREL", b.SQUIRREL)
	PrintAllocsPerOp("SQLX", b.SQLX)
	PrintAllocsPerOp("GORM", b.GORM)
	PrintAllocsPerOp("SQLC", b.SQLC)
	PrintAllocsPerOp("SQLT", b.SQLT)
	PrintAllocsPerOp("SQLT-Cache", b.SQLTCACHE)
}

func PrintNsPerOp(name string, f benchflix.Framework) {
	file := benchflix.Must(os.Create(fmt.Sprintf("data/%s_nsperop.tex", strings.ToLower(name))))

	fmt.Fprintf(file, `\begin{table}[ht]
\centering
\caption{%s: Nanosekunden pro Operation}
\begin{tabular}{lrrrrrr}
\toprule
Szenario & Params & Q1 & Q2 & Q3 & QA & QA/Q2 \\
\midrule`, name)

	FprintParams(file, "List", "100", IgnoreErr(stats.Quartile(f.List.Hundred.NsPerOp)))
	FprintParams(file, "ListPreload", "100", IgnoreErr(stats.Quartile(f.ListPreload.Hundred.NsPerOp)))
	FprintParams(file, "Dashboard", "100", IgnoreErr(stats.Quartile(f.Dashboard.Hundred.NsPerOp)))
	FprintParams(file, "DashboardPreload", "100", IgnoreErr(stats.Quartile(f.DashboardPreload.Hundred.NsPerOp)))
	FprintParams(file, "List", "1000", IgnoreErr(stats.Quartile(f.List.Thousand.NsPerOp)))
	FprintParams(file, "ListPreload", "1000", IgnoreErr(stats.Quartile(f.ListPreload.Thousand.NsPerOp)))
	FprintParams(file, "Dashboard", "1000", IgnoreErr(stats.Quartile(f.Dashboard.Thousand.NsPerOp)))
	FprintParams(file, "DashboardPreload", "1000", IgnoreErr(stats.Quartile(f.DashboardPreload.Thousand.NsPerOp)))

	fmt.Fprintf(file, `
\bottomrule
\end{tabular}
\label{tab:benchmark_%s_nsperop}
\end{table}
	`, strings.ToLower(name))
}

func PrintBytesPerOp(name string, f benchflix.Framework) {
	file := benchflix.Must(os.Create(fmt.Sprintf("data/%s_bytesperop.tex", strings.ToLower(name))))

	fmt.Fprintf(file, `\begin{table}[ht]
\centering
\caption{%s: Speicherverbrauch pro Operation}
\begin{tabular}{lrrrrrr}
\toprule
Szenario & Params & Q1 & Q2 & Q3 & QA & QA/Q2 \\
\midrule`, name)

	FprintParams(file, "List", "100", IgnoreErr(stats.Quartile(f.List.Hundred.BytesPerOp)))
	FprintParams(file, "ListPreload", "100", IgnoreErr(stats.Quartile(f.ListPreload.Hundred.BytesPerOp)))
	FprintParams(file, "Dashboard", "100", IgnoreErr(stats.Quartile(f.Dashboard.Hundred.BytesPerOp)))
	FprintParams(file, "DashboardPreload", "100", IgnoreErr(stats.Quartile(f.DashboardPreload.Hundred.BytesPerOp)))
	FprintParams(file, "List", "1000", IgnoreErr(stats.Quartile(f.List.Thousand.BytesPerOp)))
	FprintParams(file, "ListPreload", "1000", IgnoreErr(stats.Quartile(f.ListPreload.Thousand.BytesPerOp)))
	FprintParams(file, "Dashboard", "1000", IgnoreErr(stats.Quartile(f.Dashboard.Thousand.BytesPerOp)))
	FprintParams(file, "DashboardPreload", "1000", IgnoreErr(stats.Quartile(f.DashboardPreload.Thousand.BytesPerOp)))

	fmt.Fprintf(file, `
\bottomrule
\end{tabular}
\label{tab:benchmark_%s_bytesperop}
\end{table}
	`, strings.ToLower(name))
}

func PrintAllocsPerOp(name string, f benchflix.Framework) {
	file := benchflix.Must(os.Create(fmt.Sprintf("data/%s_allocsperop.tex", strings.ToLower(name))))

	fmt.Fprintf(file, `\begin{table}[ht]
\centering
\caption{%s: Allokationen pro Operation}
\begin{tabular}{lrrrrrr}
\toprule
Szenario & Params & Q1 & Q2 & Q3 & QA & QA/Q2 \\
\midrule`, name)

	FprintParams(file, "List", "100", IgnoreErr(stats.Quartile(f.List.Hundred.AllocsPerOp)))
	FprintParams(file, "ListPreload", "100", IgnoreErr(stats.Quartile(f.ListPreload.Hundred.AllocsPerOp)))
	FprintParams(file, "Dashboard", "100", IgnoreErr(stats.Quartile(f.Dashboard.Hundred.AllocsPerOp)))
	FprintParams(file, "DashboardPreload", "100", IgnoreErr(stats.Quartile(f.DashboardPreload.Hundred.AllocsPerOp)))
	FprintParams(file, "List", "1000", IgnoreErr(stats.Quartile(f.List.Thousand.AllocsPerOp)))
	FprintParams(file, "ListPreload", "1000", IgnoreErr(stats.Quartile(f.ListPreload.Thousand.AllocsPerOp)))
	FprintParams(file, "Dashboard", "1000", IgnoreErr(stats.Quartile(f.Dashboard.Thousand.AllocsPerOp)))
	FprintParams(file, "DashboardPreload", "1000", IgnoreErr(stats.Quartile(f.DashboardPreload.Thousand.AllocsPerOp)))

	fmt.Fprintf(file, `
\bottomrule
\end{tabular}
\label{tab:benchmark_%s_allocsperop}
\end{table}
	`, strings.ToLower(name))
}

func IgnoreErr(q stats.Quartiles, err error) stats.Quartiles {
	return q
}

func FprintParams(w io.Writer, szenario string, params string, q stats.Quartiles) {
	if q.Q1 == 0 {
		return
	}

	fmt.Fprintf(w, `
		%s & %s & %g & %g & %g & %g & %.1f\%% \\`,
		szenario, params, math.Round(q.Q1), math.Round(q.Q2), math.Round(q.Q3), math.Round(q.Q3-q.Q1), math.Round((q.Q3-q.Q1)/q.Q2*1000)/10)
}
