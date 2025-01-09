FROM golang:alpine as build

WORKDIR /app

COPY . .

RUN apk update && apk add make
RUN make build

FROM alpine:latest

WORKDIR /app

COPY --from=build /app/bin/cityio /app/cityio

EXPOSE 8080

ENTRYPOINT ["/app/cityio"]
