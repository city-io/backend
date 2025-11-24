include .env

.PHONY: all build start generate

all:
	go run cmd/*.go

build:
	go build -o bin/cityio cmd/*.go

start:
	bin/cityio

generate:
	sqlc generate
