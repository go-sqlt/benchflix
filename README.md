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