package main

import (
	"bufio"
	"fmt"
	"math"
	"os"
	"slices"
	"strconv"
	"strings"

	"github.com/go-sqlt/benchflix"
)

type Data struct {
	Function string
	CC       float64
	HV       float64
	MI       float64
}

func main() {
	scan := bufio.NewScanner(os.Stdin)

	var sql, pgx, squirrel, sqlx, gorm, sqlc, sqlt []Data

	for scan.Scan() {
		text := scan.Text()

		fnStart := strings.Index(text, "Function name:")
		if fnStart < 0 {
			continue
		}

		fnEnd := strings.Index(text, ", Cyclomatic Complexity:")

		var funcName string

		switch strings.TrimSpace(text[fnStart+len("Function name:") : fnEnd]) {
		case "QueryList":
			funcName = "List"
		case "QueryListPreload":
			funcName = "ListPreload"
		case "QueryDashboard":
			funcName = "Dashboard"
		case "QueryDashboardPreload":
			funcName = "DashboardPreload"
		}

		if funcName == "" {
			continue
		}

		ccStart := strings.Index(text, "Cyclomatic Complexity:") + len("Cyclomatic Complexity:")
		ccEnd := strings.Index(text, ", Halstead Volume:")
		ccStr := strings.TrimSpace(text[ccStart:ccEnd])

		hvStart := strings.Index(text, "Halstead Volume:") + len("Halstead Volume:")
		hvEnd := strings.Index(text, ", Maintainability Index:")
		hvStr := strings.TrimSpace(text[hvStart:hvEnd])

		miStart := strings.Index(text, "Maintainability Index:") + len("Maintainability Index:")
		miStr := strings.TrimSpace(text[miStart:])

		cc, err1 := strconv.Atoi(ccStr)
		hv, err2 := strconv.ParseFloat(hvStr, 64)
		mi, err3 := strconv.Atoi(miStr)
		if err1 != nil || err2 != nil || err3 != nil {
			fmt.Fprintf(os.Stderr, "Fehler beim Parsen: %v %v %v\n", err1, err2, err3)
			continue
		}

		data := Data{
			Function: funcName,
			CC:       float64(cc),
			HV:       hv,
			MI:       float64(mi),
		}

		switch {
		case strings.Contains(text, "sqlflix"):
			sql = append(sql, data)
		case strings.Contains(text, "pgxflix"):
			pgx = append(pgx, data)
		case strings.Contains(text, "squirrelflix"):
			squirrel = append(squirrel, data)
		case strings.Contains(text, "sqlxflix"):
			sqlx = append(sqlx, data)
		case strings.Contains(text, "gormflix"):
			if strings.Contains(text, "Preload") {
				gorm = append(gorm, data)
			}
		case strings.Contains(text, "sqlcflix"):
			sqlc = append(sqlc, data)
		case strings.Contains(text, "sqltflix"):
			sqlt = append(sqlt, data)
		}
	}

	Table("wartbarkeit_sql", "Wartbarkeitsmesswerte von SQL", sql)
	Table("wartbarkeit_pgx", "Wartbarkeitsmesswerte von PGX", pgx)
	Table("wartbarkeit_squirrel", "Wartbarkeitsmesswerte von SQUIRREL", squirrel)
	Table("wartbarkeit_sqlx", "Wartbarkeitsmesswerte von SQLX", sqlx)
	Table("wartbarkeit_gorm", "Wartbarkeitsmesswerte von GORM", gorm)
	Table("wartbarkeit_sqlc", "Wartbarkeitsmesswerte von SQLC", sqlc)
	Table("wartbarkeit_sqlt", "Wartbarkeitsmesswerte von SQLT", sqlt)

	frameworks := map[string][]Data{
		"SQL":      sql,
		"PGX":      pgx,
		"SQUIRREL": squirrel,
		"SQLX":     sqlx,
		"GORM":     gorm,
		"SQLC":     sqlc,
		"SQLT":     sqlt,
	}

	Szenario("wartbarkeit_list", "Wartbarkeitsmesswerte des List-Szenarios", "List", frameworks)
	Szenario("wartbarkeit_listpreload", "Wartbarkeitsmesswerte des ListPreload-Szenarios", "ListPreload", frameworks)
	Szenario("wartbarkeit_dashboard", "Wartbarkeitsmesswerte des Dashboard-Szenarios", "Dashboard", frameworks)
	Szenario("wartbarkeit_dashboardpreload", "Wartbarkeitsmesswerte des DashboardPreload-Szenarios", "DashboardPreload", frameworks)

	Maintainability("wartbarkeit", "Maintainability-Indices", frameworks)
}

func Table(output, title string, data []Data) {
	file := benchflix.Must(os.Create(fmt.Sprintf("data/%s.tex", output)))

	fmt.Fprintf(file, `\begin{table}[ht]
\centering
\caption{%s}
\begin{tabular}{lrrrr}
\toprule
Szenario & CC & HV & MI \\
\midrule`, title)

	for _, d := range data {
		fmt.Fprintf(file, `
	%s & %g & %s & %g \\`, d.Function, d.CC, benchflix.Thousand(strconv.FormatFloat(math.Round(d.HV), 'f', 0, 64)), d.MI,
		)
	}

	fmt.Fprintf(file, `
\bottomrule
\end{tabular}
\label{tab:%s}
\end{table}
	`, output)
}

func Szenario(output, title string, function string, frameworks map[string][]Data) {
	file := benchflix.Must(os.Create(fmt.Sprintf("data/%s.tex", output)))

	fmt.Fprintf(file, `\begin{table}[ht]
\centering
\caption{%s}
\begin{tabular}{lrrrr}
\toprule
Framework & CC & HV & MI \\
\midrule`, title)

	for _, framework := range []string{"SQL", "PGX", "SQUIRREL", "SQLX", "GORM", "SQLC", "SQLT"} {
		for _, d := range frameworks[framework] {
			if d.Function == function {
				fmt.Fprintf(file, `
	%s & %g & %s & %g \\`, framework, d.CC, benchflix.Thousand(strconv.FormatFloat(math.Round(d.HV), 'f', 0, 64)), d.MI,
				)
			}
		}
	}

	fmt.Fprintf(file, `
\bottomrule
\end{tabular}
\label{tab:%s}
\end{table}
	`, output)
}

func Maintainability(output, title string, frameworks map[string][]Data) {
	file := benchflix.Must(os.Create(fmt.Sprintf("data/%s.tex", output)))

	fmt.Fprintf(file, `\begin{table}[ht]
\centering
\caption{%s}
\begin{tabular}{lrrrr}
\toprule
Framework & List & ListPreload & Dashboard & DashboardPreload \\
\midrule`, title)

	for _, framework := range []string{"SQL", "PGX", "SQUIRREL", "SQLX", "GORM", "SQLC", "SQLT"} {
		fmt.Fprintf(file, `
	%s`, framework)

		for _, szenario := range []string{"List", "ListPreload", "Dashboard", "DashboardPreload"} {
			index := slices.IndexFunc(frameworks[framework], func(d Data) bool {
				return d.Function == szenario
			})
			if index < 0 {
				fmt.Fprintf(file, ` & -`)

				continue
			}

			fmt.Fprintf(file, ` & %g`, frameworks[framework][index].MI)
		}

		fmt.Fprintf(file, ` \\`)
	}

	fmt.Fprintf(file, `
\bottomrule
\end{tabular}
\label{tab:%s}
\end{table}
	`, output)
}
