.PHONY: all build

all:
	go run cmd/*.go

build:
	go build -o bin/cityio cmd/*.go
