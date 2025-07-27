package main

import (
	"fmt"
	"io"
	"math"
	"os"
	"reflect"
	"strings"

	"github.com/go-sqlt/benchflix"
	"github.com/montanaflynn/stats"
)

func main() {
	b := benchflix.Must(benchflix.ReadAll(os.Stdin))

	NsPerOp("SQL", b.SQL, benchflix.Framework{})
	NsPerOp("PGX", b.PGX, b.SQL)
	NsPerOp("SQUIRREL", b.SQUIRREL, b.SQL)
	NsPerOp("SQLX", b.SQLX, b.SQL)
	NsPerOp("GORM", b.GORM, b.SQL)
	NsPerOp("SQLC", b.SQLC, b.SQL)
	NsPerOp("SQLT", b.SQLT, b.SQL)
	NsPerOp("SQLT-Cache", b.SQLTCACHE, b.SQL)

	BytesPerOp("SQL", b.SQL, benchflix.Framework{})
	BytesPerOp("PGX", b.PGX, b.SQL)
	BytesPerOp("SQUIRREL", b.SQUIRREL, b.SQL)
	BytesPerOp("SQLX", b.SQLX, b.SQL)
	BytesPerOp("GORM", b.GORM, b.SQL)
	BytesPerOp("SQLC", b.SQLC, b.SQL)
	BytesPerOp("SQLT", b.SQLT, b.SQL)
	BytesPerOp("SQLT-Cache", b.SQLTCACHE, b.SQL)

	AllocsPerOp("SQL", b.SQL, benchflix.Framework{})
	AllocsPerOp("PGX", b.PGX, b.SQL)
	AllocsPerOp("SQUIRREL", b.SQUIRREL, b.SQL)
	AllocsPerOp("SQLX", b.SQLX, b.SQL)
	AllocsPerOp("GORM", b.GORM, b.SQL)
	AllocsPerOp("SQLC", b.SQLC, b.SQL)
	AllocsPerOp("SQLT", b.SQLT, b.SQL)
	AllocsPerOp("SQLT-Cache", b.SQLTCACHE, b.SQL)
}

func NsPerOp(name string, framework benchflix.Framework, base benchflix.Framework) {
	file := benchflix.Must(os.Create(fmt.Sprintf("data/%s_nsperop.tex", strings.ToLower(name))))

	delta := fmt.Sprintf(`& ${\Delta M_{%s,SQL}}$`, name)
	if reflect.DeepEqual(base, benchflix.Framework{}) {
		delta = ""
	}

	fmt.Fprintf(file, `
\begin{table}[ht]
\centering
\caption{%s: Nanosekunden pro Operation}
\begin{tabular}{lrrrr}
\toprule
Szenario & Params & ${M_{%s}}$ & ${QA_{%s}}$ %s  \\
\midrule
`, name, name, name, delta)

	Print(file, "List", "100", framework, base, func(f benchflix.Framework) (stats.Quartiles, error) {
		return stats.Quartile(f.List.Hundred.NsPerOp)
	})
	Print(file, "ListPreload", "100", framework, base, func(f benchflix.Framework) (stats.Quartiles, error) {
		return stats.Quartile(f.ListPreload.Hundred.NsPerOp)
	})
	Print(file, "Dashboard", "100", framework, base, func(f benchflix.Framework) (stats.Quartiles, error) {
		return stats.Quartile(f.Dashboard.Hundred.NsPerOp)
	})
	Print(file, "DashboardPreload", "100", framework, base, func(f benchflix.Framework) (stats.Quartiles, error) {
		return stats.Quartile(f.DashboardPreload.Hundred.NsPerOp)
	})
	Print(file, "List", "1000", framework, base, func(f benchflix.Framework) (stats.Quartiles, error) {
		return stats.Quartile(f.List.Thousand.NsPerOp)
	})
	Print(file, "ListPreload", "1000", framework, base, func(f benchflix.Framework) (stats.Quartiles, error) {
		return stats.Quartile(f.ListPreload.Thousand.NsPerOp)
	})
	Print(file, "Dashboard", "1000", framework, base, func(f benchflix.Framework) (stats.Quartiles, error) {
		return stats.Quartile(f.Dashboard.Thousand.NsPerOp)
	})
	Print(file, "DashboardPreload", "1000", framework, base, func(f benchflix.Framework) (stats.Quartiles, error) {
		return stats.Quartile(f.DashboardPreload.Thousand.NsPerOp)
	})

	fmt.Fprintf(file, `
\bottomrule
\end{tabular}
\label{tab:benchmark_%s_nsperop}
\end{table}
	`, strings.ToLower(name))
}

func BytesPerOp(name string, framework benchflix.Framework, base benchflix.Framework) {
	file := benchflix.Must(os.Create(fmt.Sprintf("data/%s_bytesperop.tex", strings.ToLower(name))))

	delta := fmt.Sprintf(`& ${\Delta M_{%s,SQL}}$`, name)
	if reflect.DeepEqual(base, benchflix.Framework{}) {
		delta = ""
	}

	fmt.Fprintf(file, `
\begin{table}[ht]
\centering
\caption{%s: Speicherverbrauch pro Operation}
\begin{tabular}{lrrrr}
\toprule
Szenario & Params & ${M_{%s}}$ & ${QA_{%s}}$ %s \\
\midrule
`, name, name, name, delta)

	Print(file, "List", "100", framework, base, func(f benchflix.Framework) (stats.Quartiles, error) {
		return stats.Quartile(f.List.Hundred.BytesPerOp)
	})
	Print(file, "ListPreload", "100", framework, base, func(f benchflix.Framework) (stats.Quartiles, error) {
		return stats.Quartile(f.ListPreload.Hundred.BytesPerOp)
	})
	Print(file, "Dashboard", "100", framework, base, func(f benchflix.Framework) (stats.Quartiles, error) {
		return stats.Quartile(f.Dashboard.Hundred.BytesPerOp)
	})
	Print(file, "DashboardPreload", "100", framework, base, func(f benchflix.Framework) (stats.Quartiles, error) {
		return stats.Quartile(f.DashboardPreload.Hundred.BytesPerOp)
	})
	Print(file, "List", "1000", framework, base, func(f benchflix.Framework) (stats.Quartiles, error) {
		return stats.Quartile(f.List.Thousand.BytesPerOp)
	})
	Print(file, "ListPreload", "1000", framework, base, func(f benchflix.Framework) (stats.Quartiles, error) {
		return stats.Quartile(f.ListPreload.Thousand.BytesPerOp)
	})
	Print(file, "Dashboard", "1000", framework, base, func(f benchflix.Framework) (stats.Quartiles, error) {
		return stats.Quartile(f.Dashboard.Thousand.BytesPerOp)
	})
	Print(file, "DashboardPreload", "1000", framework, base, func(f benchflix.Framework) (stats.Quartiles, error) {
		return stats.Quartile(f.DashboardPreload.Thousand.BytesPerOp)
	})

	fmt.Fprintf(file, `
\bottomrule
\end{tabular}
\label{tab:benchmark_%s_bytesperop}
\end{table}
	`, strings.ToLower(name))
}

func AllocsPerOp(name string, framework benchflix.Framework, base benchflix.Framework) {
	file := benchflix.Must(os.Create(fmt.Sprintf("data/%s_allocsperop.tex", strings.ToLower(name))))

	delta := fmt.Sprintf(`& ${\Delta M_{%s,SQL}}$`, name)
	if reflect.DeepEqual(base, benchflix.Framework{}) {
		delta = ""
	}

	fmt.Fprintf(file, `
\begin{table}[ht]
\centering
\caption{%s: Allokationen pro Operation}
\begin{tabular}{lrrrr}
\toprule
Szenario & Params & ${M_{%s}}$ & ${QA_{%s}}$ %s \\
\midrule
`, name, name, name, delta)

	Print(file, "List", "100", framework, base, func(f benchflix.Framework) (stats.Quartiles, error) {
		return stats.Quartile(f.List.Hundred.AllocsPerOp)
	})
	Print(file, "ListPreload", "100", framework, base, func(f benchflix.Framework) (stats.Quartiles, error) {
		return stats.Quartile(f.ListPreload.Hundred.AllocsPerOp)
	})
	Print(file, "Dashboard", "100", framework, base, func(f benchflix.Framework) (stats.Quartiles, error) {
		return stats.Quartile(f.Dashboard.Hundred.AllocsPerOp)
	})
	Print(file, "DashboardPreload", "100", framework, base, func(f benchflix.Framework) (stats.Quartiles, error) {
		return stats.Quartile(f.DashboardPreload.Hundred.AllocsPerOp)
	})
	Print(file, "List", "1000", framework, base, func(f benchflix.Framework) (stats.Quartiles, error) {
		return stats.Quartile(f.List.Thousand.AllocsPerOp)
	})
	Print(file, "ListPreload", "1000", framework, base, func(f benchflix.Framework) (stats.Quartiles, error) {
		return stats.Quartile(f.ListPreload.Thousand.AllocsPerOp)
	})
	Print(file, "Dashboard", "1000", framework, base, func(f benchflix.Framework) (stats.Quartiles, error) {
		return stats.Quartile(f.Dashboard.Thousand.AllocsPerOp)
	})
	Print(file, "DashboardPreload", "1000", framework, base, func(f benchflix.Framework) (stats.Quartiles, error) {
		return stats.Quartile(f.DashboardPreload.Thousand.AllocsPerOp)
	})

	fmt.Fprintf(file, `
\bottomrule
\end{tabular}
\label{tab:benchmark_%s_allocsperop}
\end{table}
	`, strings.ToLower(name))
}

func Print(w io.Writer, szenario string, params string, framework, base benchflix.Framework, fn func(benchflix.Framework) (stats.Quartiles, error)) {
	f, err := fn(framework)
	if err != nil {
		return
	}

	b, err := fn(base)
	if err != nil {
		if err == stats.ErrEmptyInput {
			fmt.Fprintf(w, `
	%s & %s & %g & %g \\`,
				szenario, params, math.Round(f.Q2), math.Round(f.Q3-f.Q1))
		}

		return
	}

	fmt.Fprintf(w, `
	%s & %s & %g & %g & %.1f\%% \\`,
		szenario, params, math.Round(f.Q2), math.Round(f.Q3-f.Q1), math.Round(f.Q2/b.Q2*1000)/10-100)
}
