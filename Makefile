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
	mkdir -p ~/.local/pg/cityio
	pg_ctl -D ~/.local/pg/cityio -l ~/.local/pg/cityio.log -o "-p 5432 -k /tmp" -w start

stop-db:
	pg_ctl -D ~/.local/pg/cityio -o "-p 5432 -k /tmp" stop

status-db:
	pg_ctl -D ~/.local/pg/cityio status
