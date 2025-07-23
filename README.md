# Benchflix

```sh
## generate params
go run params/main.go --num=1000 > params.json


go test -bench='^Benchmark/.*/List$' -benchmem -timeout=60m -count=12 -size=100 > bench_list_100.txt
go test -bench='^Benchmark/.*/List$' -benchmem -timeout=60m -count=12 -size=1000 > bench_list_1000.txt

go test -bench='^Benchmark/.*/ListPreload$' -benchmem -timeout=60m -count=12 -size=100 > bench_list_preload_100.txt
go test -bench='^Benchmark/.*/ListPreload$' -benchmem -timeout=60m -count=12 -size=1000 > bench_list_preload_1000.txt

go test -bench='^Benchmark/.*/Dashboard$' -benchmem -timeout=60m -count=12 -size=100 > bench_dashboard_100.txt
go test -bench='^Benchmark/.*/Dashboard$' -benchmem -timeout=60m -count=12 -size=1000 > bench_dashboard_1000.txt

go test -bench='^Benchmark/.*/DashboardPreload$' -benchmem -timeout=60m -count=12 -size=100 > bench_dashboard_preload_100.txt
go test -bench='^Benchmark/.*/DashboardPreload$' -benchmem -timeout=60m -count=12 -size=1000 > bench_dashboard_preload_1000.txt


cat *_100.txt | go run charts/main.go -title='100 Params' -frameworks='SQL,PGX,SQUIRREL,SQLX,GORM,SQLC,SQLT,SQLT-Cache'
cat *_1000.txt | go run charts/main.go -title='1000 Params' -frameworks='SQL,PGX,SQUIRREL,SQLX,GORM,SQLC,SQLT,SQLT-Cache'
```

## Semantic Query Example:

```sh
export OPENAI_API_KEY=...
echo "List all movies directed by Ben Affleck with a rating of at least 9." | go run semantic/main.go
{"search":"Ben Affleck","year_added":0,"min_rating":9,"limit":1000}
[{964980 Air 2023-04-05 00:00:00 +0000 UTC 7.337 [Ben Affleck]} {68734 Argo 2012-10-11 00:00:00 +0000 UTC 7.278 [Ben Affleck]} {23168 The Town 2010-09-15 00:00:00 +0000 UTC 7.2 [Ben Affleck]} {259695 Live by Night 2016-12-25 00:00:00 +0000 UTC 6.249 [Ben Affleck]}]
```