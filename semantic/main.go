package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"go/format"
	"io"
	"net/http"
	"os"
	"reflect"

	"github.com/go-sqlt/benchflix"
	"github.com/go-sqlt/benchflix/sqltflix"
	"github.com/go-sqlt/sqlt"
	"github.com/jackc/pgx/v5/pgxpool"
	_ "github.com/jackc/pgx/v5/stdlib"
)

const openAIURL = "https://api.openai.com/v1/chat/completions"

func main() {
	conn, resource := benchflix.InitializePostgres("Semantic")

	defer resource.Close()

	cfg := benchflix.Must(pgxpool.ParseConfig(conn))

	pool := benchflix.Must(pgxpool.NewWithConfig(context.Background(), cfg))

	repo := sqltflix.New(pool, sqlt.ExpressionSize(1000))

	prompt := benchflix.Must(io.ReadAll(os.Stdin))

	params := Send[benchflix.ListParams](string(prompt))

	movies, err := repo.QueryList(context.Background(), params)
	if err != nil {
		panic(err)
	}

	fmt.Println(movies)
}

func PrintStruct[T any]() string {
	t := reflect.TypeFor[T]()

	code := fmt.Sprintf("type %s struct {\n", t.Name())
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		code += fmt.Sprintf("\t%s %s `%s`\n", field.Name, field.Type, field.Tag)
	}
	code += "}\n"

	return string(benchflix.Must(format.Source([]byte(code))))
}

func Send[T any](prompt string) T {
	reqBody := map[string]any{
		"model": "gpt-4.1-mini",
		"messages": []map[string]string{
			{
				"role": "system",
				"content": `You convert natural language into a JSON object that matches the following Go struct (all field names are in snake_case).
				` + PrintStruct[T]() + `
				Return only valid JSON, no code blocks.
				This is the query generated based on the input of ListParams:
				SELECT
					m.id                    {{ Scan.Int.To "ID" }}
					, m.title               {{ Scan.String.To "Title" }}
					, m.added_at            {{ Scan.Time.To "AddedAt" }}
					, m.rating              {{ Scan.Float.To "Rating" }}
					, d.directors           {{ Scan.StringSlice.To "Directors" }}
				FROM movies m
				LEFT JOIN LATERAL (
					SELECT ARRAY_AGG(p.name ORDER BY p.name) AS directors
					FROM movie_directors md
					JOIN people p ON p.id = md.person_id
					WHERE md.movie_id = m.id
				) d ON true
				WHERE
					(
						{{ .Search }} = ''
						OR to_tsvector('simple', m.title) @@ plainto_tsquery('simple', {{ .Search }})
						OR EXISTS (
							SELECT 1
							FROM movie_directors md
							JOIN people p ON p.id = md.person_id
							WHERE md.movie_id = m.id
							AND to_tsvector('simple', p.name) @@ plainto_tsquery('simple', {{ .Search }})
						)
					)
					AND ({{ .YearAdded }} = 0 OR EXTRACT(YEAR FROM m.added_at) = {{ .YearAdded }})
					AND ({{ .MinRating }} = 0 OR m.rating >= {{ .MinRating }})
				ORDER BY m.rating DESC
				LIMIT CASE WHEN {{ .Limit }} BETWEEN 1 AND 1000 THEN {{ .Limit }} ELSE 1000 END;
				`,
			},
			{
				"role":    "user",
				"content": string(prompt),
			},
		},
	}

	req, err := http.NewRequest("POST", openAIURL, bytes.NewBuffer(benchflix.Must(json.Marshal(reqBody))))
	if err != nil {
		panic(err)
	}
	req.Header.Set("Authorization", "Bearer "+os.Getenv("OPENAI_API_KEY"))
	req.Header.Set("Content-Type", "application/json")

	client := http.Client{}
	resp := benchflix.Must(client.Do(req))
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		panic("Fehler:" + string(benchflix.Must(io.ReadAll(resp.Body))))
	}

	var result struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		panic(err)
	}

	fmt.Println(result.Choices[0].Message.Content)

	var t T
	if err := json.Unmarshal([]byte(result.Choices[0].Message.Content), &t); err != nil {
		panic(err)
	}

	return t
}
