# Benchflix

```sh
## generate params
go run params/main.go --size=1000 > params.json

go test -bench='^Benchmark/.*/List$/.*' -benchmem -timeout=120m -count=14 > list.bench
go test -bench='^Benchmark/.*/ListPreload$/.*' -benchmem -timeout=120m -count=14 > list_preload.bench
go test -bench='^Benchmark/.*/Dashboard$/.*' -benchmem -timeout=120m -count=14 > dashboard.bench
go test -bench='^Benchmark/.*/DashboardPreload$/.*' -benchmem -timeout=120m -count=14 > dashboard_preload.bench

cat *.bench | go run charts/main.go -frameworks='SQL,PGX,SQUIRREL,SQLX,GORM,SQLC,SQLT,SQLT-Cache'
```

## Semantic Query Example:

```sh
export OPENAI_API_KEY=...

echo "List all movies directed by Ben Affleck with a rating of at least 7." | go run semantic/main.go
{"search":"Ben Affleck","min_rating":7,"limit":1000,"sort":"rating","desc":false,"year_added":0,"with_directors":true}
[{23168 The Town 2010-09-15 00:00:00 +0000 UTC 7.2 [Ben Affleck]} {68734 Argo 2012-10-11 00:00:00 +0000 UTC 7.278 [Ben Affleck]} {964980 Air 2023-04-05 00:00:00 +0000 UTC 7.337 [Ben Affleck]}]

echo "2018年に公開された、タイトルに英単語「shark」が含まれているすべての映画と、その監督を一覧にしてください。" | go run semantic/main.go  
{"search":"shark","year_added":2018,"with_directors":true}
[{522438 6-Headed Shark Attack 2018-08-18 00:00:00 +0000 UTC 4.7 [Mark Atkins]}]
```

## Tables

```sh
cat *.bench | go run tables/main.go -framework=SQL -szenarios=List,ListPreload,Dashboard,DashboardPreload > table_sql.tex
cat *.bench | go run tables/main.go -framework=PGX -szenarios=List,ListPreload,Dashboard,DashboardPreload > table_pgx.tex
cat *.bench | go run tables/main.go -framework=SQUIRREL -szenarios=Dashboard,DashboardPreload > table_squirrel.tex
cat *.bench | go run tables/main.go -framework=SQLX -szenarios=List,ListPreload,Dashboard,DashboardPreload > table_sqlx.tex
cat *.bench | go run tables/main.go -framework=GORM -szenarios=ListPreload,DashboardPreload > table_gorm.tex
cat *.bench | go run tables/main.go -framework=SQLC -szenarios=List,ListPreload > table_sqlc.tex
cat *.bench | go run tables/main.go -framework=SQLT -szenarios=List,ListPreload,Dashboard,DashboardPreload > table_sqlt.tex
cat *.bench | go run tables/main.go -framework=SQLT-Cache -szenarios=List,ListPreload,Dashboard,DashboardPreload > table_sqlt_cache.tex
```