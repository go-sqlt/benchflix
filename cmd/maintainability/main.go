package main

import (
	"bufio"
	"fmt"
	"math"
	"os"
	"strconv"
	"strings"

	"slices"

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
			gorm = append(gorm, data)
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

	Vergleich("wartbarkeit_relativ_pgx", "Relative Wartbarkeitsmesswerte von PGX zu SQL in \\%", pgx, sql)
	Vergleich("wartbarkeit_relativ_squirrel", "Relative Wartbarkeitsmesswerte von SQUIRREL zu SQL in \\%", squirrel, sql)
	Vergleich("wartbarkeit_relativ_sqlx", "Relative Wartbarkeitsmesswerte von SQLX zu SQL in \\%", sqlx, sql)
	Vergleich("wartbarkeit_relativ_gorm", "Relative Wartbarkeitsmesswerte von GORM zu SQL in \\%", gorm, sql)
	Vergleich("wartbarkeit_relativ_sqlc", "Relative Wartbarkeitsmesswerte von SQLC zu SQL in \\%", sqlc, sql)
	Vergleich("wartbarkeit_relativ_sqlt", "Relative Wartbarkeitsmesswerte von SQLT zu SQL in \\%", sqlt, sql)
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
	%s & %g & %g & %g \\`, d.Function, d.CC, math.Round(d.HV), d.MI,
		)
	}

	fmt.Fprintf(file, `
\bottomrule
\end{tabular}
\label{tab:%s}
\end{table}
	`, output)
}

func Vergleich(output, title string, data []Data, sql []Data) {
	file := benchflix.Must(os.Create(fmt.Sprintf("data/%s.tex", output)))

	fmt.Fprintf(file, `\begin{table}[ht]
\centering
\caption{%s}
\begin{tabular}{lrrrr}
\toprule
Szenario & CC & HV & MI \\
\midrule`, title)

	for _, d := range data {
		base := sql[slices.IndexFunc(sql, func(r Data) bool { return r.Function == d.Function })]

		fmt.Fprintf(file, `
	%s & %s %g & %s %g & %s %g \\`,
			d.Function,
			cellColor(math.Round(d.CC/base.CC*1000.0)/10),
			math.Round(d.CC/base.CC*1000.0)/10,
			cellColor(math.Round(d.HV/base.HV*1000.0)/10),
			math.Round(d.HV/base.HV*1000.0)/10,
			cellColor(math.Round(d.MI/base.MI*1000.0)/10*-1+200),
			math.Round(d.MI/base.MI*1000.0)/10,
		)
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
