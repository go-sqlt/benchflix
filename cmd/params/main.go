package main

import (
	"encoding/json"
	"fmt"
	"log"
	"math"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/go-sqlt/benchflix"
)

type Data map[int][5]benchflix.Stats

func main() {
	files, err := filepath.Glob("data/stats_*.json")
	if err != nil {
		log.Fatal(err)
	}

	data := Data{}

	for _, file := range files {
		f, err := os.Open(file)
		if err != nil {
			log.Fatal(err)
		}

		var stats benchflix.Stats

		if err := json.NewDecoder(f).Decode(&stats); err != nil {
			panic(err)
		}

		f.Close()

		base := filepath.Base(file)
		parts := strings.Split(strings.TrimSuffix(base, ".json"), "_")
		if len(parts) >= 3 {
			size := benchflix.Must(strconv.Atoi(parts[1]))

			arr, ok := data[size]
			if !ok {
				arr = [5]benchflix.Stats{}
			}

			arr[benchflix.Must(strconv.Atoi(parts[2]))-1] = stats
			data[size] = arr
		} else {
			panic(file)
		}
	}

	latexEscape := func(s string) string {
		replacer := strings.NewReplacer(
			`&`, `\&`,
			`%`, `\%`,
			`$`, `\$`,
			`#`, `\#`,
			`_`, `\_`,
			`{`, `\{`,
			`}`, `\}`,
			`~`, `\textasciitilde{}`,
			`^`, `\textasciicircum{}`,
			`\`, `\textbackslash{}`,
		)
		return replacer.Replace(s)
	}

	file := benchflix.Must(os.Create("data/parameterverteilung.tex"))

	fmt.Fprintf(file, `
\renewcommand{\arraystretch}{0.7}
\begin{longtable}{lllrrrrr}
\caption{Parameterverteilung in \%%}\label{tab:parameterverteilung}\\
\toprule
Parameter & Feld & Wert & ${H_1}$ & ${H_2}$ & ${H_3}$ & ${H_4}$ & ${H_5}$ \\
\midrule
\endfirsthead

\toprule
Parameter & Feld & Wert & ${H_1}$ & ${H_2}$ & ${H_3}$ & ${H_4}$ & ${H_5}$ \\
\midrule
\endhead

\endfoot

\endlastfoot`)

	for _, size := range []int{500, 5_000} {
		chunks := data[size]

		for _, value := range benchflix.Search {
			fmt.Fprintf(file, `
	%d & Search & %s`, size, latexEscape(fmt.Sprint(value)),
			)

			for _, chunk := range chunks {
				fmt.Fprintf(file, ` & %d`, chunk.Search[value])
			}

			fmt.Fprintf(file, ` \\`)
		}

		for _, value := range benchflix.YearAdded {
			fmt.Fprintf(file, `
	%d & YearAdded & %s`, size, latexEscape(fmt.Sprint(value)),
			)

			for _, chunk := range chunks {
				fmt.Fprintf(file, ` & %d`, chunk.YearAdded[strconv.FormatInt(value, 10)])
			}

			fmt.Fprintf(file, ` \\`)
		}

		for _, value := range benchflix.MinRating {
			fmt.Fprintf(file, `
	%d & MinRating & %s`, size, latexEscape(fmt.Sprint(value)),
			)

			for _, chunk := range chunks {
				fmt.Fprintf(file, ` & %d`, chunk.MinRating[strconv.FormatFloat(value, 'f', 1, 64)])
			}

			fmt.Fprintf(file, ` \\`)
		}

		for _, value := range benchflix.Limit {
			fmt.Fprintf(file, `
	%d & Limit & %s`, size, latexEscape(fmt.Sprint(value)),
			)

			for _, chunk := range chunks {
				fmt.Fprintf(file, ` & %d`, chunk.Limit[strconv.FormatUint(value, 10)])
			}

			fmt.Fprintf(file, ` \\`)
		}

		for _, value := range benchflix.Sort {
			fmt.Fprintf(file, `
	%d & Sort & %s`, size, latexEscape(fmt.Sprint(value)),
			)

			for _, chunk := range chunks {
				fmt.Fprintf(file, ` & %d`, chunk.Sort[value])
			}

			fmt.Fprintf(file, ` \\`)
		}

		for _, value := range benchflix.Desc {
			fmt.Fprintf(file, `
	%d & Desc & %s`, size, latexEscape(fmt.Sprint(value)),
			)

			for _, chunk := range chunks {
				fmt.Fprintf(file, ` & %d`, chunk.Desc[strconv.FormatBool(value)])
			}

			fmt.Fprintf(file, ` \\`)
		}

		for _, value := range benchflix.WithDirectors {
			fmt.Fprintf(file, `
	%d & WithDirectors & %s`, size, latexEscape(fmt.Sprint(value)),
			)

			for _, chunk := range chunks {
				fmt.Fprintf(file, ` & %d`, chunk.WithDirectors[strconv.FormatBool(value)])
			}

			fmt.Fprintf(file, ` \\`)
		}
	}

	fmt.Fprintf(file, `
\bottomrule
\label{tab:parameterverteilung}
\end{longtable}`)
}

func cellColor(percent float64) string {
	return fmt.Sprintf(`\cellcolor{orange!%.0f} `, math.Abs(percent))
}
