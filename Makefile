include .env

.PHONY: all build start generate start-db stop-db status-db

all:
	go run cmd/*.go

build:
	go build -o bin/cityio cmd/*.go

start:
	bin/cityio

generate:
	sqlc generate

start-db:
	@test -f ~/.local/pg/cityio/PG_VERSION || initdb -D ~/.local/pg/cityio -U cityio --auth=trust --encoding=UTF8
	@pg_ctl -D ~/.local/pg/cityio status >/dev/null 2>&1 || pg_ctl -D ~/.local/pg/cityio -l ~/.local/pg/cityio.log -o "-p 5432 -k /tmp" -w start
	@psql -h localhost -p 5432 -U cityio -d postgres -tAc "SELECT 1 FROM pg_database WHERE datname='cityio'" | grep -q 1 || createdb -h localhost -p 5432 -U cityio cityio

stop-db:
	pg_ctl -D ~/.local/pg/cityio -o "-p 5432 -k /tmp" stop

status-db:
	pg_ctl -D ~/.local/pg/cityio status
