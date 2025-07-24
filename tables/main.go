package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"slices"
	"strconv"
	"strings"

	"github.com/go-sqlt/benchflix"
	"github.com/montanaflynn/stats"
	"golang.org/x/tools/benchmark/parse"
)

var (
	framework = flag.String("framework", "SQL", "Framwork")
	cores     = flag.Int("cores", 12, "Number of cores")
	szenarios = flag.String("szenarios", "", "Szanarios")
	params    = flag.String("params", "", "Number of Params")
)

type Data struct {
	Szenario          string
	NsPerOp           []float64
	AllocedBytesPerOp []float64
	AllocsPerOp       []float64
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

		szenario := strings.TrimSuffix(parts[2], "-"+strconv.Itoa(*cores))

		index := slices.IndexFunc(data, func(d Data) bool { return d.Szenario == szenario })
		if index < 0 {

			data = append(data, Data{
				Szenario:          szenario,
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

	NsPerOp := make([]stats.Quartiles, len(SZENARIOS))
	BytesPerOp := make([]stats.Quartiles, len(SZENARIOS))
	AllocsPerOp := make([]stats.Quartiles, len(SZENARIOS))

	for _, d := range data {
		index := slices.Index(SZENARIOS, d.Szenario)
		if index < 0 {
			continue
		}

		NsPerOp[index] = benchflix.Must(stats.Quartile(d.NsPerOp))
		BytesPerOp[index] = benchflix.Must(stats.Quartile(d.AllocedBytesPerOp))
		AllocsPerOp[index] = benchflix.Must(stats.Quartile(d.AllocsPerOp))
	}

	fmt.Printf(`
\begin{table}[ht]
\centering
\caption{%s: Nanosekunden pro Operation für %s Parameter}
\begin{tabular}{lccccc}
\toprule
Szenario & 1. Quartil & 2. Quartil & 3. Quartil & Quartilabstand \\
\midrule`, *framework, *params)

	for i, q := range NsPerOp {
		fmt.Printf(`
	%s & %g & %g & %g & %g \\`, SZENARIOS[i], q.Q1, q.Q2, q.Q3, q.Q3-q.Q1)
	}

	fmt.Printf(`
\bottomrule
\end{tabular}
\label{tab:benchmark_%s_nsperop_%s}
\end{table}
	`, strings.ToLower(*framework), *params)

	fmt.Printf(`
\begin{table}[ht]
\centering
\caption{%s: Speicherverbrauch pro Operation für %s Parameter}
\begin{tabular}{lccccc}
\toprule
Szenario & 1. Quartil & 2. Quartil & 3. Quartil & Quartilabstand \\
\midrule`, *framework, *params)

	for i, q := range BytesPerOp {
		fmt.Printf(`
	%s & %g & %g & %g & %g \\`, SZENARIOS[i], q.Q1, q.Q2, q.Q3, q.Q3-q.Q1)
	}

	fmt.Printf(`
\bottomrule
\end{tabular}
\label{tab:benchmark_%s_bytesperop_%s}
\end{table}
	`, strings.ToLower(*framework), *params)

		fmt.Printf(`
\begin{table}[ht]
\centering
\caption{%s: Allokationen pro Operation für %s Parameter}
\begin{tabular}{lccccc}
\toprule
Szenario & 1. Quartil & 2. Quartil & 3. Quartil & Quartilabstand \\
\midrule`, *framework, *params)

	for i, q := range AllocsPerOp {
		fmt.Printf(`
	%s & %g & %g & %g & %g \\`, SZENARIOS[i], q.Q1, q.Q2, q.Q3, q.Q3-q.Q1)
	}

	fmt.Printf(`
\bottomrule
\end{tabular}
\label{tab:benchmark_%s_allocsperop_%s}
\end{table}
	`, strings.ToLower(*framework), *params)
}
