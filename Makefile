include .env

.PHONY: all build start generate db-start db-stop db-status

all:
	go run cmd/*.go

build:
	go build -o bin/cityio cmd/*.go

start:
	bin/cityio

generate:
	sqlc generate

db-start:
	pg_ctl -D ~/.local/pg/cityio -l ~/.local/pg/cityio.log -o "-p 5432 -k /tmp" -w start

db-stop:
	pg_ctl -D ~/.local/pg/cityio -o "-p 5432 -k /tmp" stop

db-status:
	pg_ctl -D ~/.local/pg/cityio status
