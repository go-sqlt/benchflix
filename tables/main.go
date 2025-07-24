package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"slices"
	"strings"

	"github.com/go-sqlt/benchflix"
	"github.com/montanaflynn/stats"
	"golang.org/x/tools/benchmark/parse"
)

var (
	framework = flag.String("framework", "SQL", "Framwork")
	cores     = flag.String("cores", "12", "Number of cores")
	szenarios = flag.String("szenarios", "", "Szanarios")
)

type Data struct {
	Szenario          string
	Params            string
	NsPerOp           []float64
	AllocedBytesPerOp []float64
	AllocsPerOp       []float64
}

type Stats struct {
	Hundred  stats.Quartiles
	Thousand stats.Quartiles
}

func main() {
	flag.Parse()

	SZENARIOS := strings.Split(*szenarios, ",")

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

		if parts[1] != *framework {
			continue
		}

		szenario := parts[2]
		params := strings.TrimSuffix(parts[3], "-"+*cores)

		index := slices.IndexFunc(data, func(d Data) bool {
			return d.Szenario == szenario && params == d.Params
		})
		if index < 0 {

			data = append(data, Data{
				Szenario:          szenario,
				Params:            params,
				NsPerOp:           []float64{float64(b.NsPerOp)},
				AllocedBytesPerOp: []float64{float64(b.AllocedBytesPerOp)},
				AllocsPerOp:       []float64{float64(b.AllocsPerOp)},
			})

			continue
		}

		data[index].NsPerOp = append(data[index].NsPerOp, b.NsPerOp)
		data[index].AllocedBytesPerOp = append(data[index].AllocedBytesPerOp, float64(b.AllocedBytesPerOp))
		data[index].AllocsPerOp = append(data[index].AllocsPerOp, float64(b.AllocsPerOp))
	}

	NsPerOp := make([]Stats, len(SZENARIOS))
	BytesPerOp := make([]Stats, len(SZENARIOS))
	AllocsPerOp := make([]Stats, len(SZENARIOS))

	for _, d := range data {
		index := slices.Index(SZENARIOS, d.Szenario)
		if index < 0 {
			continue
		}

		switch d.Params {
		case "100":
			NsPerOp[index].Hundred = benchflix.Must(stats.Quartile(d.NsPerOp))
			BytesPerOp[index].Hundred = benchflix.Must(stats.Quartile(d.AllocedBytesPerOp))
			AllocsPerOp[index].Hundred = benchflix.Must(stats.Quartile(d.AllocedBytesPerOp))
		case "1000":
			NsPerOp[index].Thousand = benchflix.Must(stats.Quartile(d.NsPerOp))
			BytesPerOp[index].Thousand = benchflix.Must(stats.Quartile(d.AllocedBytesPerOp))
			AllocsPerOp[index].Thousand = benchflix.Must(stats.Quartile(d.AllocedBytesPerOp))
		default:
			panic(d.Params)
		}
	}

	fmt.Printf(`
\begin{table}[ht]
\centering
\caption{%s: Nanosekunden pro Operation}
\begin{tabular}{lcccccc}
\toprule
Szenario & Params & Q1 & Q2 & Q3 & QA \\
\midrule`, *framework)

	for i, q := range NsPerOp {
		fmt.Printf(`
	%s & %s & %g & %g & %g & %g \\`,
			SZENARIOS[i], "100", q.Hundred.Q1, q.Hundred.Q2, q.Hundred.Q3, q.Hundred.Q3-q.Hundred.Q1)
	}

	for i, q := range NsPerOp {
		fmt.Printf(`
	%s & %s & %g & %g & %g & %g \\`,
			SZENARIOS[i], "1000", q.Thousand.Q1, q.Thousand.Q2, q.Thousand.Q3, q.Thousand.Q3-q.Thousand.Q1)
	}

	fmt.Printf(`
\bottomrule
\end{tabular}
\label{tab:benchmark_%s_nsperop}
\end{table}
	`, strings.ToLower(*framework))

	fmt.Printf(`
\begin{table}[ht]
\centering
\caption{%s: Speicherverbrauch pro Operation}
\begin{tabular}{lccccc}
\toprule
Szenario & Params & Q1 & Q2 & Q3 & QA \\
\midrule`, *framework)

	for i, q := range BytesPerOp {
		fmt.Printf(`
	%s & %s & %g & %g & %g & %g \\`,
			SZENARIOS[i], "100", q.Hundred.Q1, q.Hundred.Q2, q.Hundred.Q3, q.Hundred.Q3-q.Hundred.Q1)
	}

	for i, q := range BytesPerOp {
		fmt.Printf(`
	%s & %s & %g & %g & %g & %g \\`,
			SZENARIOS[i], "1000", q.Thousand.Q1, q.Thousand.Q2, q.Thousand.Q3, q.Thousand.Q3-q.Thousand.Q1)
	}

	fmt.Printf(`
\bottomrule
\end{tabular}
\label{tab:benchmark_%s_bytesperop}
\end{table}
	`, strings.ToLower(*framework))

	fmt.Printf(`
\begin{table}[ht]
\centering
\caption{%s: Allokationen pro Operation}
\begin{tabular}{lccccc}
\toprule
Szenario & Params & Q1 & Q2 & Q3 & QA \\
\midrule`, *framework)

	for i, q := range AllocsPerOp {
		fmt.Printf(`
	%s & %s & %g & %g & %g & %g \\`,
			SZENARIOS[i], "100", q.Hundred.Q1, q.Hundred.Q2, q.Hundred.Q3, q.Hundred.Q3-q.Hundred.Q1)
	}

	for i, q := range AllocsPerOp {
		fmt.Printf(`
	%s & %s & %g & %g & %g & %g \\`,
			SZENARIOS[i], "1000", q.Thousand.Q1, q.Thousand.Q2, q.Thousand.Q3, q.Thousand.Q3-q.Thousand.Q1)
	}

	fmt.Printf(`
\bottomrule
\end{tabular}
\label{tab:benchmark_%s_allocsperop}
\end{table}
	`, strings.ToLower(*framework))
}
