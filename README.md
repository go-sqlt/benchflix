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

echo "2018年に公開された、タイトルに英単語「shark」が含まれているすべての映画を一覧にしてください" | go run semantic/main.go  
{"search":"shark","year_added":2018,"min_rating":0,"limit":1000}
[{831546 Envoy: Shark Cull 2021-07-21 00:00:00 +0000 UTC 7.3 [Andrea Borella]} {801604 Shark: The Beginning 2021-11-26 00:00:00 +0000 UTC 7.3 [Johnny Chae]} {1008437 Baby Shark's Big Movie 2024-02-07 00:00:00 +0000 UTC 6.724 [Alan Foreman]} {960258 Shark Bait 2022-05-13 00:00:00 +0000 UTC 5.7 [James Nunn]} {461108 Toxic Shark 2017-08-03 00:00:00 +0000 UTC 5.6 [Cole Sharpe]} {1011679 Shark Side of the Moon 2022-08-12 00:00:00 +0000 UTC 5.1 [Glenn Campbell Tammy Klein]} {216539 Ghost Shark 2013-08-22 00:00:00 +0000 UTC 5 [Griff Furst]} {65055 Shark Night 3D 2011-09-02 00:00:00 +0000 UTC 4.992 [David R. Ellis]} {522438 6-Headed Shark Attack 2018-08-18 00:00:00 +0000 UTC 4.7 [Mark Atkins]} {433536 Shark Babes 2015-10-01 00:00:00 +0000 UTC 4.6 [Jim Wynorski]} {728069 Shark Season 2020-07-28 00:00:00 +0000 UTC 4.5 [Jared Cohn]} {342927 3-Headed Shark Attack 2015-07-20 00:00:00 +0000 UTC 4.427 [Christopher Ray]} {86703 2-Headed Shark Attack 2012-06-26 00:00:00 +0000 UTC 4.3 [Christopher Ray]} {460218 5-Headed Shark Attack 2017-07-10 00:00:00 +0000 UTC 4.2 [Nico De Leon]} {347849 Zombie Shark 2015-07-20 00:00:00 +0000 UTC 4.2 [Misty Talley]} {83880 Super Shark 2011-12-08 00:00:00 +0000 UTC 4.2 [Fred Olen Ray]} {63749 Swamp Shark 2011-06-25 00:00:00 +0000 UTC 4.108 [Griff Furst]} {343097 Mega Shark vs. Kolossus 2015-06-18 00:00:00 +0000 UTC 4.1 [Christopher Ray]} {246594 Mega Shark vs. Mecha Shark 2014-01-28 00:00:00 +0000 UTC 4 [Emile Edwin Smith]} {52454 Mega Shark vs. Crocosaurus 2010-12-21 00:00:00 +0000 UTC 3.9 [Christopher Ray]} {299553 Shark Lake 2015-10-02 00:00:00 +0000 UTC 3.5 [Jerry Dugan]} {120846 Jurassic Shark 2012-09-21 00:00:00 +0000 UTC 2.4 [Brett Kelly]}]
```