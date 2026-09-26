# city.io-backend
Backend for city.io, written in Golang

## Container builds

Go images are built with ko v0.19.1 using the Go version in `go.mod`.
CI saves Go modules and compiler outputs between commits. Image repositories,
tags, target architectures, and deployment triggers are preserved.

To build locally with ko and Docker installed:

```sh
ko build ./cmd --local --platform=linux/amd64
```

SQL files from `db/migrations` are embedded in the Go binary, so migrations
remain available in containers and local builds without a working-directory dependency.
