FROM golang:alpine AS build

WORKDIR /app

COPY . .

RUN apk update && apk add make
RUN go build -o bin/cityio cmd/*.go

FROM alpine:latest

WORKDIR /app

COPY --from=build /app/bin/cityio /app/cityio

EXPOSE 8080

ENTRYPOINT ["/app/cityio"]
