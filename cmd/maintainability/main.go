package main

import (
	"bufio"
	"fmt"
	"math"
	"os"
	"strconv"
	"strings"

	"github.com/go-sqlt/benchflix"
)

type Data struct {
	Function string
	CC       int
	HV       float64
	MI       int
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
		case "NewRepository":
			funcName = "NewRepository"
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

		if cc == 1 && hv < 100 {
			continue
		}

		data := Data{
			Function: funcName,
			CC:       cc,
			HV:       hv,
			MI:       mi,
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
			gorm = append(gorm, data)
		case strings.Contains(text, "sqlcflix"):
			sqlc = append(sqlc, data)
		case strings.Contains(text, "sqltflix"):
			sqlt = append(sqlt, data)
		}
	}

	Fprint("SQL", sql)
	Fprint("PGX", pgx)
	Fprint("SQUIRREL", squirrel)
	Fprint("SQLX", sqlx)
	Fprint("GORM", gorm)
	Fprint("SQLC", sqlc)
	Fprint("SQLT", sqlt)
}

func Fprint(name string, data []Data) {
	file := benchflix.Must(os.Create(fmt.Sprintf("data/%s_maintainability.tex", strings.ToLower(name))))

	fmt.Fprintf(file, `\begin{table}[ht]
\centering
\caption{%s: Wartbarkeit}
\begin{tabular}{lrrrr}
\toprule
Szenario & CC & HV & MI \\
\midrule`, name)

	for _, d := range data {
		fmt.Fprintf(file, `
	%s & %d & %g & %d \\`, d.Function, d.CC, math.Round(d.HV), d.MI,
		)
	}

	fmt.Fprintf(file, `
\bottomrule
\end{tabular}
\label{tab:%s_maintainability}
\end{table}
	`, strings.ToLower(name))
}
