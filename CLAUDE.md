# CLAUDE.md

Guidance for AI agents working in this repository. Read this fully before making changes.

## What this is

`cityio` is the Go backend for **city.io**, a real-time, tick-based multiplayer city-building
game. The world is a 128×128 grid of tiles. Players own cities; cities contain buildings;
buildings produce resources and grow population on a timer. Almost all game state lives in
memory as **actors** (Proto-Actor), with PostgreSQL used as a periodic write-behind backup
rather than the source of truth during runtime.

- Module: `cityio` (see `go.mod`), Go 1.25.2
- Entry point: `cmd/main.go`
- Actor framework: `asynkron/protoactor-go` (with clustering)
- DB access: `pgx/v5` + `sqlc`-generated code; migrations via `goose`
- Config: `caarlos0/env`; logging: `log/slog` + `lmittmann/tint`

## Architecture at a glance

```
HTTP (api)  ──▶  services  ──▶  cluster (ports.ClusterProvider)  ──▶  actors  ──▶  databaseActor  ──▶  Postgres
                                                                         │
                                                       per-entity in-memory state + tickers
```

- **`cmd/main.go`** — composition root. Loads config, sets up logging, connects the DB,
  builds the cluster runtime, runs world setup, then starts the HTTP server.
- **`internal/domain`** — pure domain entities and enums (`User`, `City`, `Building`, `Tile`,
  `Coordinates`, `NullTime`, plus `CityType`/`BuildingType`). **No framework imports.** This
  package must stay dependency-free; the sqlc-generated `database` package imports it for the
  `Coordinates` composite type.
- **`internal/actors`** — the heart of the system. One actor per live entity:
  - `userActor`, `cityActor`, `buildingActor`, `tileActor` — each embeds `baseActor`.
  - `buildingActor` delegates type-specific behavior to a `buildingActorImpl`
    (`cityCenter.go`, `townCenter.go`, `house.go`, `farm.go`, `mine.go`, `barracks.go`) via
    `Create` / `Destroy` / `Handle` hooks.
  - `databaseActor` (`database.go`) is the single sink for persistence: it buffers
    `Update*` messages in maps (latest-write-wins) and batch-flushes them to Postgres on a
    ticker. Create/Delete/queries pass through immediately.
- **`internal/services`** — thin orchestration layer called by the API/setup. Functions take
  `(ctx, cluster, input)` and translate requests into actor messages. DTOs that callers send
  in live here (`inputs.go`: `CreateUserRequest`, `CityInput`, `BuildingInput`).
- **`internal/messages`** — the actor message types (the protocol). Plain structs, grouped by
  domain (`user.go`, `city.go`, `buildings.go`, `tile.go`, `database.go`, `general.go`).
- **`internal/cluster`** — implements `ports.ClusterProvider`. Registers actor "kinds",
  spawns the database actor, wires the logging context onto each actor. Uses the in-memory
  test cluster provider in non-prod and consul in prod.
- **`internal/ports`** — interfaces that decouple layers (notably `ClusterProvider`), so
  `services`/`actors` depend on an interface rather than the concrete `cluster` package.
- **`internal/database`** — `sqlc`-generated query code (`*.sql.go`, `models.go`, `querier.go`,
  `db.go`) plus hand-written `database.go` (`NewDB`) and `utils.go` (row→domain `ToModel`
  converters and `ToPGTimestamp`). **Do not hand-edit generated files**; change the SQL in
  `db/queries` and run `sqlc generate`.
- **`internal/config`** — env-driven config, parsed once in `Load()`.
- **`internal/logger`** — global slog setup (`Setup`) and a context-aware handler. `With(ctx,
  k, v, ...)` attaches attributes to a context; any `slog.*Context(ctx, ...)` call then emits
  them automatically. This is how actor type, environment, phase, etc. ride along on logs.
- **`internal/ws`** — websocket connection registry and outbound payload types (`types.go`).
- **`internal/constants`** — tunables (map size, tick frequencies, costs) and the building
  stat tables (`buildings.go`).
- **`internal/setup`** — `Run()` seeds/restores the world on boot (see gotcha below).
- **`internal/api`** — HTTP layer (gorilla/mux + cors + JWT). **Currently mostly disabled.**

## How data flows (typical patterns)

- **Request/response across actors:** `cluster.Request(kind, identity, msg)` returns
  `(any, error)`; the receiving actor replies with `ctx.Respond(...)`. Used when a result or
  ack is needed (e.g. gold deduction before an upgrade).
- **Fire-and-forget:** `cluster.Tell(kind, identity, msg)` or `ctx.Send(...)`. Used for state
  nudges (resource production, population cap changes). Errors are only logged, not propagated.
- **Persistence:** actors send `Create*/Update*/Delete*` messages to `cluster.DB()` (the
  database actor). `Update*` are buffered and batched; everything else is immediate.
- **Timers:** most actors start a `time.Ticker` goroutine that sends themselves a
  `PeriodicOperationMessage` every N seconds (frequencies in `constants`).

## Build / run / database

The app is normally run from the repo root via the `Makefile` (which `include`s `.env`).

```bash
make all        # go run cmd/*.go   (build + run)
make build      # build to bin/cityio
make start      # run bin/cityio
make generate   # sqlc generate (regenerate internal/database from db/queries + schema)
```

Run/build commands must be executed from the **repo root** — `NewDB` loads migrations from the
relative path `db/migrations`.

### Environment

Config comes from env vars (see `.env`, git-ignored). A single `PSQL_`-prefixed DB set is used;
deployments swap the values per environment.

```
ENVIRONMENT, API_PORT, JWT_SECRET
PSQL_HOST, PSQL_PORT, PSQL_DATABASE, PSQL_USERNAME, PSQL_PASSWORD
```

`.env.production` holds prod values (also git-ignored). Load with `set -a && source .env && set +a`.

### Local Postgres (dev)

There is no system/systemd Postgres on this machine; a local user-owned cluster is used:

```bash
# start (after reboot the cluster is NOT auto-started)
pg_ctl -D ~/.local/pg/cityio -l ~/.local/pg/cityio.log -o "-p 5432 -k /tmp" -w start
# stop
pg_ctl -D ~/.local/pg/cityio -o "-p 5432 -k /tmp" stop
# psql
psql -h localhost -p 5432 -U cityio -d cityio
```

Migrations can be run manually (the app also runs them itself — see gotcha):

```bash
GOOSE_DRIVER=postgres \
GOOSE_DBSTRING="host=localhost port=5432 user=cityio password=cityio dbname=cityio sslmode=disable" \
goose -dir db/migrations up
```

## Critical gotchas

- **The world is destroyed and rebuilt on every boot.** `NewDB` runs goose `down-to 0` (drops
  all tables) then `up`, and `setup.Run()` then regenerates ~490 random towns + a capital per
  user and registers a hardcoded test user (`cityio@example.com`). Restarting the app wipes
  state. Treat persistence as a backup, not durable storage, until this is gated.
- **The HTTP API is effectively disabled.** In `internal/api/api.go`, `addRoutes` is commented
  out and the router is empty, so every endpoint returns 404. Most handlers, auth middleware,
  and the websocket loop are commented out. Wiring these back up is expected future work.
- **Create writes are fire-and-forget.** Failures to persist a create are logged but not
  surfaced to the caller (the actor exists in memory regardless).
- **Generated code:** `internal/database/*.sql.go`, `db.go`, `models.go`, `querier.go` are
  produced by sqlc. Edit `db/queries/*.sql` / `db/migrations/*.sql` and `sqlc.yaml`, then
  regenerate. The only hand-written files in that package are `database.go` and `utils.go`.

## Conventions to follow

- **Logging:** always use the context-aware slog calls — `slog.InfoContext(ctx, ...)`,
  `slog.ErrorContext(ctx, ...)`, etc. — with key/value pairs (`"city_id", id`). In actors use
  `state.Ctx()` as the context. Enrich context with `logger.With(ctx, "key", val)` rather than
  formatting values into the message string. Don't introduce a new logger or `fmt.Printf`.
- **Layering:** keep `domain` framework-free. Actors talk to other actors and the DB actor
  through `ports.ClusterProvider`, never by importing `cluster` directly. Services orchestrate;
  they don't hold game logic that belongs in an actor.
- **Messages are the contract.** Add a new struct in `internal/messages` and handle it in the
  relevant actor's `Receive` (or a building impl's `Handle`) rather than adding ad-hoc methods.
- **New building types:** add the enum to `domain/building.go`, stat entries in
  `constants/buildings.go`, a `*Impl` in `internal/actors` implementing `buildingActorImpl`,
  and a case in `buildingActor.Receive`'s impl switch.
- **Errors:** return them up where a caller can act; otherwise log with context. Match the
  existing pattern in the file you're editing.

## Comment & style rules (important)

Match the existing codebase, which is deliberately sparse and lets clear names and small
functions carry the meaning. Do **not** make the code read like it was written by an AI.

- **Do not narrate the code.** No line-by-line comments restating what the next statement does
  (`// increment the counter`, `// loop over cities`, `// send the message`). The reader can
  see that.
- **Comment _why_, not _what_** — and only when the reason is non-obvious: a tricky invariant,
  a deliberate trade-off, a workaround, a `TODO` for known-incomplete work. The existing
  `// sqlc will parse "" into NULL` and the `TODO:` notes are the right level.
- **Follow Go doc conventions** for exported identifiers: a short doc comment starting with the
  identifier name (see `logger.With`, `database.NewDB`, `config.Load`). Keep these concise.
- **Don't add comments to code you didn't meaningfully change**, and don't add docstrings/type
  annotations purely for coverage.
- **No decorative banners, no changelog/edit-history comments, no "removed X" tombstones.** If
  code is dead, delete it (the codebase already keeps large commented-out blocks in `api/` —
  don't add more of that style elsewhere).
- Keep changes minimal and focused; prefer editing existing patterns over introducing new
  abstractions. Don't add error handling, fallbacks, or config for cases that can't occur.

## Before you finish

Run and make sure these are clean:

```bash
go build ./...
go vet ./...
gofmt -l internal/ cmd/      # should print nothing
```

There is currently **no test suite**. Don't claim something works because it builds — if you
change runtime behavior, exercise it (run the app, inspect the DB) or say what you verified.
