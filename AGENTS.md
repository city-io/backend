# city.io Backend

Guidance for AI agents working in this repository. Read this fully before making changes.
It covers both **backend development** (the Go server in this repo) and **client/frontend
integration** (the RPC contract a web client consumes) — see the "Client / frontend API
reference" section for the latter.

## What this is

`cityio` is the Go backend for **city.io**, a real-time, tick-based multiplayer city-building
game. The world is a 75×75 grid of tiles. Players own cities; cities contain buildings;
buildings produce resources and grow population on a timer; barracks train troops that spawn as
mobile **armies** which march across the map. Almost all game state lives in memory as
**actors** (Proto-Actor), with PostgreSQL used as a periodic write-behind backup rather than the
source of truth during runtime. Armies can be trained, fed, ordered, split, merged, fought,
retreated, and used to capture settlements.

- Module: `cityio` (see `go.mod`), Go 1.25.2
- Entry point: `cmd/main.go`
- Actor framework: `asynkron/protoactor-go` (with clustering)
- API: **Connect RPC** (`connectrpc.com/connect`) over h2c; generated from proto by `buf`
- DB access: `pgx/v5` + `sqlc`-generated code; migrations via `goose`
- Config: `caarlos0/env`; logging: `log/slog` + `lmittmann/tint`; metrics: Prometheus

## Architecture at a glance

```
Connect RPC (rpc)  ──▶  services  ──▶  cluster (contracts.ClusterProvider)  ──▶  actors  ──┐
        │                                                                │             │
   stream (pub/sub) ◀───────────────────────── actors publish ──────────┘   per-entity in-memory
        │                                                                    state + tickers
   StreamState RPC ──▶ client                                                     │
                                                          persistence.Store (contracts.Store)  ──▶  Postgres
```

- **`cmd/main.go`** — composition root. Loads config, sets up logging, connects the DB,
  builds the persistence store and cluster runtime, runs world setup, then serves Connect
  RPC over h2c + CORS (and `/metrics` + `/healthz` on the same port).
- **`internal/domain`** — pure domain entities and enums (`User`, `City`, `Building`, `Army`,
  `Tile`, `Coordinates`, `TerrainGrid`, `NullTime`, plus
  `CityType`/`BuildingType`/`TroopType`/`TerrainType`). **No framework imports.** This package
  must stay dependency-free; the sqlc-generated `database` package imports it for the
  `Coordinates` composite type.
- **`internal/actors`** — the heart of the system. One actor per live entity, six kinds:
  - `userActor`, `cityActor`, `buildingActor`, `tileActor`, `armyActor`, `battleActor` — each embeds `baseActor`.
  - `buildingActor` delegates type-specific behavior to a `buildingActorImpl`
    (`cityCenter.go`, `townCenter.go`, `house.go`, `farm.go`, `mine.go`, `barracks.go`) via
    `Create` / `Destroy` / `Handle` hooks. `barracks.go` is the troop **producer**: it holds a
    durable FIFO training queue and a one-shot completion timer. Completed orders retry an
    idempotent `armyActor` spawn until it succeeds.
  - `armyActor` (`army.go`) owns one army: persistence, a 250ms movement ticker, tile presence,
    nearest-owned-settlement food-upkeep attribution, composition-aware weighted
    8-directional terrain pathfinding, merging, combat enrollment, retreat, and capture orders.
  - `battleActor` owns one active battle: alliance-ready attacker/defender sides, simultaneous
    one-second combat ticks, deterministic whole-unit casualties, and resolution.
  - Actors persist through the injected `contracts.Store` (`state.Store`): reads/creates/deletes
    hit the DB immediately; `Enqueue*` coalesces updates for the background writer.
- **`internal/persistence`** — `Store` (implements `contracts.Store`), the single sink for
  persistence. Reads, creates and deletes go straight to the `database.Querier` (the pgx pool
  is concurrency-safe). `Enqueue*` buffers updates per entity (`user`/`city`/`building`/`army`)
  in mutex-guarded maps (latest-write-wins); a background goroutine started by `Start` flushes
  them to Postgres on a ticker via snapshot-and-swap (so enqueues never block on a flush).
  `Stop` does a final flush. Per-user explored coordinates are inserted synchronously and
  idempotently because they are durable game knowledge, not write-behind actor state.
- **`internal/services`** — thin orchestration layer called by the rpc/setup layers. Functions
  take `(ctx, cluster, ...)` (and the `store` where needed) and translate requests into actor
  messages. DTOs that callers send in live here (`inputs.go`: `CreateUserRequest`, `CityInput`,
  `BuildingInput`, `ArmyInput`). Files: `users.go`, `cities.go`, `buildings.go`, `armies.go`.
- **`internal/messages`** — the actor message types (the protocol). Plain structs, grouped by
  domain (`user.go`, `city.go`, `buildings.go`, `tile.go`, `army.go`, `general.go`).
- **`internal/cluster`** — implements `contracts.ClusterProvider`. Registers the actor "kinds"
  (`user`, `city`, `tile`, `building`, `army`, `battle`), injects the `contracts.Store` and logging context
  onto each actor. Uses the in-memory test cluster provider in non-prod and consul in prod.
- **`internal/contracts`** — shared dependency interfaces (`ClusterProvider`, `Store`,
  `WorldProvider`) that keep `services`/`actors`/`rpc` independent of concrete infrastructure.
- **`internal/rpc`** — Connect RPC handlers (one file per service: `user.go`, `city.go`,
  `building.go`, `army.go`, `map.go`, `config.go`, plus `rpc.go` for wiring) over the generated
  code in `internal/gen`. `NewServer(shutdownCtx, cluster, store, jwtSecret)` builds the handler;
  a JWT interceptor (`internal/auth`) guards authenticated methods; a metrics interceptor wraps
  everything.
- **`internal/auth`** — JWT issuing/verification and the Connect auth interceptor (covers both
  unary and streaming). `publicProcedures` lists the tokenless endpoints.
- **`internal/mapping`** — domain↔proto conversion helpers used by `rpc` (e.g. `CityToProto`,
  `ArmyToProto`, `EntitiesToBag`, typed-ID + enum maps, and viewer-specific field hiding).
- **`internal/stream`** — in-process pub/sub triggers backing the server-streaming `StreamState`
  RPC. Private user changes target their owner; world changes wake every subscriber, whose RPC
  projection applies visibility. The RPC emits complete-entity deltas and an authoritative
  snapshot every five seconds.
- **`internal/battles`** — concurrency-safe registry of active battle snapshots. Battle actors
  update it; RPC projections read it for unary/list/stream entity bags. Battles are intentionally
  ephemeral and are not restored after a process restart.
- **`internal/gen`** — generated Connect/protobuf code (from `buf`). Two sub-packages:
  - `internal/gen/cityio/entity/v1` (`entityv1`) — entity messages (`User`, `City`,
    `Building`, `Army`, `ArmyOrder`, `Battle`, `TroopStack`, `Tile`), typed IDs (`UserId`, `CityId`,
    `BuildingId`, `ArmyId`, `ArmyOrderId`, `BattleId`, `TileId`), enums (`CityType`, `BuildingType`, `TroopType`, `TerrainType`),
    `Coordinates`, `Rate`, and `EntityBag` (a collection of mixed entities).
  - `internal/gen/cityio/service/v1` (`servicev1`) — RPC request/response messages. The
    `servicev1connect` sub-package has the Connect service interfaces and handler constructors.
  Do not hand-edit.
- **`internal/database`** — `sqlc`-generated query code (`*.sql.go`, `models.go`, `querier.go`,
  `db.go`) plus hand-written `database.go` (`NewDB`) and `utils.go` (row→domain `ToModel`
  converters and `ToPGTimestamp`). **Do not hand-edit generated files**; change the SQL in
  `db/queries` and run `sqlc generate`.
- **`internal/config`** — env-driven config, parsed once in `Load()`.
- **`internal/logger`** — global slog setup (`Setup`) and a context-aware handler. `With(ctx,
  k, v, ...)` attaches attributes to a context; any `slog.*Context(ctx, ...)` call then emits
  them automatically. This is how actor type, environment, phase, etc. ride along on logs.
- **`internal/constants`** — tunables (map size, tick frequencies, costs) and the stat tables:
  buildings in `buildings.go`, troops in `troops.go`.
- **`internal/metrics`** — Prometheus metric definitions + the RPC interceptor + a periodic
  world-snapshot gauge filler.
- **`internal/worldgen`** — deterministic terrain and settlement generation. A cryptographic
  seed is chosen for an empty database and persisted in `world_state`; smooth elevation/moisture
  fields reproduce the same coherent terrain regions on later boots. Footprint-aware placement
  reserves capital sites before generating neutral towns. `World` also restores occupied
  settlement footprints and allocates terrain-valid sites for later registrations.
- **`internal/setup`** — `Run()` seeds neutral towns only when all gameplay tables are empty,
  restores persisted actors, and registers the development test user only when it is missing.
- **`scripts/troops.py`** — a dev-only helper that drives `ArmyService` over the Connect JSON
  API (no `grpcurl` needed). See "Client / frontend API reference → Local testing".

## Proto structure

Proto definitions are split into two packages under `proto/cityio/`; `buf generate` writes the
Go output to `internal/gen`. A frontend generates its own client from the same files.

```
proto/cityio/
  entity/v1/             # package cityio.entity.v1
    ids.proto             # typed IDs (UserId, CityId, BuildingId, ArmyId, ArmyOrderId, BattleId, TileId)
    common.proto          # enums (CityType, BuildingType, TroopType), Coordinates, Rate
    user.proto            # User entity message
    city.proto            # City entity message
    building.proto        # Building entity message
    army.proto            # Army + TroopStack entity messages
    army_order.proto      # active move/attack/conquer/retreat orders and routes
    battle.proto          # active battles and alliance-ready sides
    tile.proto            # TerrainType + Tile
    bag.proto             # EntityBag (users/cities/buildings/armies/orders/battles/tiles)
  service/v1/             # package cityio.service.v1
    user.proto            # UserService RPCs (incl. StreamState) + req/resp
    city.proto            # CityService RPCs
    building.proto        # BuildingService RPCs
    army.proto            # ArmyService RPCs
    map.proto             # MapService RPCs
    config.proto          # ConfigService RPCs
    state.proto           # snapshots, deltas, typed tombstones, tile visibility
```

Procedure names are `/cityio.service.v1.<Service>/<Method>`; the `publicProcedures` map in
`internal/auth` uses this prefix. Field/message details and per-RPC behaviour are documented in
the "Client / frontend API reference" section below.

## Game model & rules (shared frontend/backend mental model)

- **Map:** a `MapSize`×`MapSize` (75×75) grid. A tile is addressed by `(x, y)` and has one of
  eight terrain types: grassland, plains, forest, hills, mountains, desert, marsh, or water.
  Terrain is reconstructed from the persisted world seed on every boot and revealed as raw tile
  entities only after exploration; a new seed is created only for an empty database. Terrain
  affects settlement placement and army movement, but not production.
  Buildings and armies live on tiles; armies stack (multiple armies + a building can share a
  tile).
- **Cities:** a `size`×`size` block (capitals are `CitySize` = 5). `start` is the top-left
  corner; the center is `start + size/2`. Type is `city` (player capital) or `town` (neutral,
  unowned). New player cities temporarily start with a completed level-1 farm and barracks in
  addition to their city center. A city has `population` (grows logistically toward
  `population_cap`), and the cap is the sum of its buildings' population contributions.
- **Buildings:** typed structures inside a city. Types: `city_center`, `town_center`, `barracks`,
  `house`, `farm`, `mine`. Levels 1..`MAX_BUILDING_LEVEL` (10). Building/upgrading takes
  construction time; while under construction `level != target_level`. City/town centers can't be
  demolished. New level-0 construction produces nothing; an existing building continues operating
  at 75% of its current completed level's production while upgrading. Fractional per-tick output is
  carried so the long-run rate remains exact. Stat tables (cost, construction time, production,
  population) live in `constants/buildings.go` and are exposed to clients via
  `ConfigService.GetGameConfig`.
- **Resources & economy:** two resources, `gold` and `food`, pooled per **user** (not per city).
  Centers/mines produce gold; farms produce food. Each city consumes food upkeep =
  `(population − military_population) × FoodPerPopPerHour` (48/hr) **plus** the upkeep of any
  armies attributed to it. A city consumes its own food first, deposits surplus to the user pool,
  and draws the shortfall from the pool. Starvation and population change compare stable hourly
  production/upkeep rates rather than rounded per-tick food units, while the pool still transfers
  whole units using carried remainders. A locally-under-producing city is consistently `starving`
  and its population declines. All rates are per-hour.
- **Troops & armies:** a barracks trains batches of troops (`soldier`, `archer`, `cavalry`,
  `artillery`). A completed batch spawns an `Army` at the barracks tile. An army has a tile
  position and optional references to its active `ArmyOrder` and `Battle`. Orders own their
  objective, remaining route, and ETA. Movement follows a lowest-time route choosing among
  all 8 neighbours. A diagonal has the same base cost as an orthogonal step. Movement uses a
  250ms timing quantum with carried fractional progress, but state is streamed only when the army actually enters a tile. An
  army moves at the speed of its slowest troop: cavalry takes 825ms per normal tile,
  soldiers/archers take 1.65s, and artillery takes 2.475s. Marsh multiplies that time by two,
  mountains by three, and water is impassable to current land armies. Armies can stack, and two
  same-owner armies on the same tile can be merged.
  - **Combat:** hostile armies sharing a tile enter a battle. Battles tick once per second and
    compute both sides' damage from the pre-tick composition, so casualties are simultaneous.
    Attack, defense, and HP determine fractional expected losses; battle-local carry converts
    them into deterministic whole-unit deaths over time. There is no persistent army/unit health
    or wounded state. A zero-unit army is deleted. A side can contain multiple users: additional
    attackers targeting a participant join the opposing side, leaving formal alliance policy for
    the future diplomacy layer.
  - **Conquest:** an army ordered to conquer fights defenders on the settlement center tile and
    then must hold it uncontested for 30 seconds. Any new defender resets capture progress.
    Completion transfers the existing settlement and its buildings to the attacker.
  - **Population carve-out:** training reserves population into `city.military_population`, capped
    at `MilitaryPopulationFraction` (0.35) of the city's population. Civilians
    (`population − military_population`) drive city food upkeep, so a standing army is
    exploit-free. Reserved population is not released on merge (troops keep their origin's
    reservation); a release path will come with combat/disband later.
  - **Food upkeep:** each army's food upkeep is added to its **nearest owned settlement's**
    upkeep, recomputed (by Chebyshev distance) as the army marches. Cities and captured towns
    both qualify because they share the `City` domain model.
  - **Troop stat table** (tier-1; balance knobs in `constants/troops.go`; Atk/Def/HP are stored
    and used by the one-second combat calculation):

    | Type      | Gold | Train/unit (s) | Move (s) | Food/hr | Pop | Atk | Def | HP  |
    |-----------|------|----------------|----------|---------|-----|-----|-----|-----|
    | soldier   | 50   | 5              | 1.650    | 60      | 1   | 10  | 10  | 100 |
    | archer    | 75   | 7              | 1.650    | 60      | 1   | 15  | 5   | 70  |
    | cavalry   | 150  | 10             | 0.825    | 180     | 1   | 20  | 12  | 120 |
    | artillery | 300  | 15             | 2.475    | 120     | 3   | 40  | 3   | 60  |

  - **Barracks training capacity** (troops per in-progress batch) = `5 × barracksLevel`. Extra
    orders persist and queue FIFO per barracks; more barracks = more concurrent training. A
    batch takes `troop count × per-troop train time`; a barracks with pending orders cannot be
    upgraded or demolished. `GetGameConfig`
    does **not** expose troop stats yet — a client needs them hardcoded or we should add a troop
    config message (TODO).
- **Vision:** a player sees any tile within Chebyshev distance `VisionRadius` (3) of any tile of
  a city they own or the current tile of any army they own. This gates what read RPCs return
  (see visibility rules below). Seen coordinates are persisted in `explored_tiles`: terrain stays
  known after vision leaves, while occupancy and dynamic entities are removed. Army movement
  persists and streams newly explored terrain whenever an army enters a tile.
- **Tick cadence** (`constants/constants.go`): city tick 3s, building tick 3s, army movement
  quantum 250ms, DB backup flush 2s, user backup 10s. Rates are normalised to per-hour
  (`SecondsPerHour` 3600).

## Client / frontend API reference

This is what a web client (or any external caller) needs. The proto files under `proto/cityio/`
are the source of truth; generate a typed client from them (e.g. `connect-es` / `connect-query`
for TypeScript) rather than hand-writing request types.

### Transport & endpoints

- **Protocol:** Connect RPC served over **HTTP/2 cleartext (h2c)** on `API_PORT` (default
  `8080`). connect-go handlers accept the Connect protocol, gRPC, and gRPC-Web, so a browser
  client can use `connect-web`/`connect-query`. Unary calls are also reachable as a plain
  `POST` with `Content-Type: application/json` (handy for curl/scripts).
- **Procedure path:** `/cityio.service.v1.<Service>/<Method>` — e.g.
  `/cityio.service.v1.UserService/Login`.
- **CORS:** allows origins `http://localhost:5173`, `http://localhost:4173`, and any
  `*.prayujt.com`; `AllowCredentials` is on; Connect/gRPC headers are exposed.
- **Ops endpoints (same port, plain HTTP):** `GET /healthz` → `ok`; `GET /metrics` → Prometheus.

### Auth

- JWT (HS256). Obtain a token from `UserService.Register` or `UserService.Login`, then send it
  on every other call as `Authorization: Bearer <token>`. Claims carry `userId`, `username`,
  `email`.
- **Public (no token):** `UserService/Register`, `UserService/Login`,
  `ConfigService/GetGameConfig`. Everything else returns `Unauthenticated` without a valid token.
- The `StreamState` streaming handler is also authenticated and is scoped to the token's user.

### Common types

- **Typed IDs** — `UserId`/`CityId`/`BuildingId`/`ArmyId`/`TrainingOrderId` are each
  `{ "value": "<uuid>" }`.
  `TileId` is the tile's coordinate identity: `{ "x": int32, "y": int32 }`.
- **Coordinates** — `{ "x": int32, "y": int32 }`.
- **Rate** — `{ "value": int64, "scale": int32 }`; the real rate is `value / scale` **per
  second**. Resource flows use `scale = 3600` (per hour). The client divides to display.
- **Enums** (proto JSON uses the full SCREAMING_SNAKE name):
  - `CityType`: `CITY_TYPE_CITY`, `CITY_TYPE_TOWN` (+ `..._UNSPECIFIED`).
  - `BuildingType`: `BUILDING_TYPE_CITY_CENTER`, `_TOWN_CENTER`, `_BARRACKS`, `_HOUSE`, `_FARM`,
    `_MINE` (+ `_UNSPECIFIED`).
  - `TroopType`: `TROOP_TYPE_SOLDIER`, `_ARCHER`, `_CAVALRY`, `_ARTILLERY` (+ `_UNSPECIFIED`).
- **EntityBag** — a flat collection of raw entities returned by list/map/stream responses:
  `{ users[], cities[], buildings[], armies[], tiles[], army_orders[], battles[] }`. Stream typed deletion/hidden IDs live
  in `StateDelta`, not in the entity collection.
- **TileVisibilityState** — per-user tile knowledge: `UNEXPLORED`, `EXPLORED`, or `VISIBLE`.
  `EXPLORED` tiles include remembered terrain but sanitized occupancy.

### Entity shapes & visibility

- **User** `{ user_id, email, username, gold, food, food_income (Rate), food_upkeep (Rate) }` —
  password is never sent.
- **City** — public fields: `city_id, type, owner? (UserId), name, population (double),
  population_cap (double), start (Coordinates, top-left), size, starving (bool),
  population_growth (Rate), military_population (double)`. **Owner-only** fields (nil for
  non-owners): `food_production, food_upkeep, net_food_flow (Rate)`. See
  `mapping.HidePrivateCityFields`.
- **Building** `{ building_id, city_id, type, level, target_level, coords, construction_start?,
  construction_end? }` — under construction when `level != target_level` (timestamps present).
- **Army** `{ army_id, owner (UserId), coords, composition_visibility, troops[], order_id?, battle_id? }` —
  composition is exact for the owner and explicitly hidden from unauthorized viewers; private
  order/battle references are removed from sanitized foreign armies.
- **ArmyOrder** `{ army_order_id, army_id, move|attack_army|conquer_settlement|retreat,
  remaining_route{known_steps[], hidden_segment_end?}, estimated_remaining_duration? }` —
  ephemeral owner-projected active state. Known steps are exact explored geometry; the optional
  endpoint marks an undisclosed connector rather than another exact step. Completion,
  cancellation, failure, or replacement produces a tombstone; there are no terminal status/reason
  records.
- **Battle** `{ battle_id, tile_id, attackers, defenders, started_at, next_tick_at }` — ephemeral
  active combat. Both sides contain repeated user and army IDs so future allies can share a side.
- **Tile** `{ tile_id: TileId, terrain, city_id?, building_id?, army_ids[] }` — terrain is
  immutable for the current generated world; occupancy references resolve through the same
  `EntityBag`.
- **Visibility rules** (enforced server-side):
  - Reads that expose the world (`GetMap`, `ListBuildings`) filter cities/buildings/armies to
    those within `VisionRadius` of an owned city or army; non-owned city economy fields are
    stripped.
  - `GetCity` and `GetBuilding` return `NotFound` if the target isn't visible. `GetTile` returns
    terrain-only data for explored hidden tiles and `NotFound` only for unexplored tiles.
  - `GetArmy`: the **owner** can always fetch their own army and active order/battle (even out of
    vision); others need vision on its tile and receive sanitized private state.
  - `ListCities` and `ListArmies` are **owner-scoped** — they return your own entities regardless
    of vision. `StreamState` begins with the same owner-scoped snapshot and includes visible
    world deltas revealed by moving armies.

### Services & RPCs

**UserService**
- `Register(email, username, password) → { user_id, token }` — *public*. `email`/`username`
  required; `password` ≥ 8 chars; `AlreadyExists` on duplicate email/username. Auto-creates a
  capital city for the new user.
- `Login(identifier, password) → { token, user }` — *public*. `identifier` is email or username.
- `GetUser(user_id) → { user }`.
- `DeleteUser(user_id) → {}`.
- `StreamState() → stream { revision, snapshot | delta }` — server-streaming and owner-scoped.
  The first frame and every five-second repair frame are authoritative snapshots. Between them,
  deltas carry complete entity upserts, typed `deleted` and `hidden` ID bags, and tile-visibility
  transitions, including army-order/battle upserts and tombstones. Apply snapshots by replacement and
  deltas by ID. World updates are projected to every player who can currently see them; moving
  armies reveal and persist terrain.

**CityService**
- `GetCity(city_id) → { city }` — vision-gated; economy fields owner-only.
- `CreateCity(type, owner?, name, size) → { city }` — placed on a random empty block.
- `ListCities() → { city_ids[], entities(cities) }` — your owned cities.

**BuildingService**
- `CreateBuilding(city_id, type, coords) → { building }` — must own the city; starts construction
  (spawns at level 0 → target 1).
- `GetBuilding(building_id) → { building }` — vision-gated.
- `UpgradeBuilding(building_id) → {}` — must own; deducts gold and starts construction to the next
  level. Errors: `FailedPrecondition` (`InsufficientGold`, `ConstructionInProgress`,
  `TrainingInProgress` for a barracks with queued orders, `MaxLevelReached`).
- `DeleteBuilding(building_id) → {}` — must own; city/town centers and barracks with pending
  training orders can't be demolished (`FailedPrecondition`).
- `ListBuildings(city_id) → { buildings[] }` — vision-filtered.

**ArmyService**
- `TrainTroops(barracks_id, type, count) → { order }` — must own the barracks' city; the
  barracks must be finished (not under construction); `count` ∈ `[1, 5 × barracksLevel]`. Reserves
  `count × popCost` military population (≤ 35% of city population) and deducts `count × goldCost`.
  Errors: `FailedPrecondition` (`InsufficientGold`, insufficient trainable population, training
  capacity exceeded, construction in progress); `InvalidArgument` (bad count/type). After the
  batch's total train time (`count × per-troop time`) an `Army` spawns at the barracks tile
  (observe it via `StreamState`).
  Multiple orders queue FIFO per barracks. The returned order includes its future `army_id` and,
  when it is at the front, `started_at` and `completes_at`.
- `ListTrainingOrders(barracks_id) → { orders[] }` — owner-only current FIFO queue for a
  barracks, including the active order and orders waiting behind it.
- `GetArmy(army_id) → { army_id, entities(army, army_order?, battle?) }` — owner always; others
  need vision and receive sanitized private state.
- `PreviewArmyRoute(army_id, destination) → { route, estimated_duration }` —
  owner-only backend route preview. The UI derives whether the requested destination is reached
  by comparing it with the end of `route`. Unknown tiles are assumed to be ordinary land without
  revealing terrain. `route.known_steps` is exact contiguous explored geometry; when
  `route.hidden_segment_end` is present, the gap to it is undisclosed geometry that clients may
  render as an uncertain straight connector but must not treat as exact route steps. Streamed
  `ArmyOrder.remaining_route` uses the same shape.
- `MoveArmy(army_id, destination) → {}` — must own; sets the marching destination (clamped to the
  map). Missing destinations are invalid. Unknown terrain is planned as ordinary land and the
  the remaining route stays stable while it remains optimal and traversable, and is replaced when
  newly revealed terrain invalidates it or exposes a faster route. If the target proves to be water
  or becomes unreachable, the army stops at the closest reachable explored land instead of leaking
  that fact up front.
- `AttackArmy(army_id, target_army_id) → {}` — must own the attacker and currently see the
  hostile target. The order follows updated visible coordinates, stops at the last-known tile if
  contact is lost, and starts or joins a battle on contact.
- `ConquerSettlement(army_id, city_id) → {}` — must own the army and see the hostile or neutral
  settlement. The army fights center-tile defenders, then captures it after a 30-second
  uncontested hold.
- `RetreatArmy(army_id) → {}` — removes the army from its active battle and routes it to the
  nearest owned settlement.
- `MergeArmies(target_army_id, source_army_id) → {}` — must own both; both must be on the same
  tile. The source's troops fold into the target and the source army disappears.
- `SplitArmy(army_id, troops[]) → { army_id, entities(source_army, new_army, army_order?) }` —
  must own the source and it cannot be in battle. Counts are detached into a new idle army on
  the same tile while the source retains its active order. At least one troop must remain in the
  source; splitting does not change reserved military population.
- `ListArmies() → { army_ids[], entities(armies, army_orders, battles) }` — your armies and their
  active command/combat state (all, regardless of vision).

**MapService**
- `GetMap() → { tile_ids[], entities(tiles, cities, buildings, armies), tile_visibility[] }` —
  all tile IDs remain map roots, but only explored tile entities are included. Visible tiles have
  occupancy; explored hidden tiles have remembered terrain only. Dynamic entities are
  vision-filtered and non-owned city economy is stripped.
- `GetTile(tile_id) → { tile, visibility }` — explored-gated; hidden results contain terrain only.

**ConfigService**
- `GetGameConfig() → { map_size, city_size, vision_radius, building_tick (Duration),
  city_tick (Duration), buildings[]: { type, levels[]: { level, cost[], construction_time
  (Duration), production[], population } } }` — *public*, static tunables for the client. Note:
  troop stats are **not** exposed here yet (see Game model → troop stat table).

### Error codes (Connect)

`Unauthenticated` (missing/invalid token) · `PermissionDenied` (not owner) · `NotFound` (missing
or not visible) · `FailedPrecondition` (game-rule rejection: funds/population/capacity/
construction/max-level/same-tile-merge) · `InvalidArgument` (bad input) · `AlreadyExists`
(duplicate email/username) · `Internal` (unexpected). Error messages are human-readable.

### Local testing (no grpcurl)

With the server running (`make all`) use the dev helper — it speaks Connect JSON over HTTP and
caches a JWT for the seeded test user (`cityio@example.com` / `cityio`):

```bash
python3 scripts/troops.py smoke                       # full flow: build barracks → train → spawn → move
python3 scripts/troops.py login                       # cache a token
python3 scripts/troops.py cities                       # list owned cities (+ coords)
python3 scripts/troops.py barracks                     # build a barracks in your first city
python3 scripts/troops.py train <barracksId> soldier 5
python3 scripts/troops.py queue <barracksId>            # inspect its durable FIFO queue
python3 scripts/troops.py armies                        # list your armies
python3 scripts/troops.py move <armyId> <x> <y>
python3 scripts/troops.py merge <targetId> <sourceId>
python3 scripts/troops.py split <armyId> soldier 2
```

Raw curl equivalent for a login:

```bash
curl -s http://localhost:8080/cityio.service.v1.UserService/Login \
  -H 'Content-Type: application/json' \
  -d '{"identifier":"cityio@example.com","password":"cityio"}'
```

## How data flows (backend patterns)

- **Request/response across actors:** `cluster.Request(kind, identity, msg)` returns
  `(any, error)`; the receiving actor replies with `ctx.Respond(...)`. Used when a result or
  ack is needed (e.g. gold deduction before an upgrade, population reservation before training,
  army spawn on training completion).
- **Fire-and-forget:** `cluster.Tell(kind, identity, msg)` or `ctx.Send(...)`. Used for state
  nudges (resource production, population cap changes, army food-upkeep set/remove, tile
  add/remove army). Errors are only logged, not propagated.
- **Persistence:** actors call the injected `contracts.Store` directly — `Create*`/`Delete*`/reads
  hit Postgres immediately; `Enqueue*` buffers updates that the store's background writer
  batch-flushes.
- **Timers:** most actors start a `time.Ticker` goroutine that sends themselves a
  `PeriodicOperationMessage` every N seconds (frequencies in `constants`). For state that flips
  at a known future time (construction complete, training complete), an actor also arms a
  one-shot `time.AfterFunc` that fires the same `PeriodicOperationMessage` at that instant — the
  per-tick poll remains the idempotent safety net. See
  `buildingActor.scheduleConstructionComplete` and `barracksImpl.arm`.
- **Live client push:** actors publish state-change triggers to `internal/stream`; private user
  changes target their owner and world changes wake all subscribers. `StreamState` rebuilds each
  user's visibility projection, emits a delta when it changed, and sends a full repair snapshot
  every five seconds.

## Build / run / database

The app is normally run from the repo root via the `Makefile` (which `include`s `.env`).

```bash
make all        # go run cmd/*.go   (build + run)
make build      # build to bin/cityio
make start      # run bin/cityio
make generate   # sqlc generate (regenerate internal/database from db/queries + schema)
make start-db   # start the local Postgres cluster (not auto-started after reboot)
make stop-db    # stop the local Postgres cluster
make status-db  # check if Postgres is running
buf generate    # regenerate internal/gen from proto/cityio/{entity,service}/v1/*.proto
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

Migrations can be run manually (the app also runs them itself — see gotcha). Migrations live in
`db/migrations/` (`00001_initial_schema`, `00002_drop_derived_columns`, `00003_add_armies`,
`00004_add_training_orders`, `00005_add_explored_tiles`, `00006_add_world_state`):

```bash
GOOSE_DRIVER=postgres \
GOOSE_DBSTRING="host=localhost port=5432 user=cityio password=cityio dbname=cityio sslmode=disable" \
goose -dir db/migrations up
```

## Critical gotchas

- **World and player state survive restarts.** `NewDB` runs goose `up` without rolling migrations
  down. The world seed is stored in `world_state`, neutral towns are seeded only when every
  gameplay table is empty, and the hardcoded test user (`cityio@example.com`) is created only when
  missing. The commented `down-to 0` block in `NewDB` can be re-enabled for an intentional clean
  reset after a breaking change. New migrations still apply automatically on boot; a manual
  `goose up` is only useful if a running instance stays on old code.
- **The API is Connect RPC, served over h2c.** Handlers live in `internal/rpc`; auth is a JWT
  Connect interceptor. Live state is pushed to clients via the server-streaming `StreamState`
  RPC (backed by `internal/stream`), not websockets.
- **Create writes are fire-and-forget.** Failures to persist a create are logged but not
  surfaced to the caller (the actor exists in memory regardless).
- **Armies stack.** The `armies` table has **no** unique coords constraint — many armies (and a
  building) can share a tile. `troops` is stored as JSONB (`map[TroopType]int64`).
- **Generated code:** `internal/database/*.sql.go`, `db.go`, `models.go`, `querier.go` are
  produced by sqlc. Edit `db/queries/*.sql` / `db/migrations/*.sql` and `sqlc.yaml`, then
  regenerate. The only hand-written files in that package are `database.go` and `utils.go`.
  `internal/gen/` is produced by `buf generate` from `proto/cityio/entity/v1/*.proto` and
  `proto/cityio/service/v1/*.proto`. Do not hand-edit generated files in either package.
- **Nullable columns in sqlc batch UPDATEs can't carry NULL through `UNNEST` arrays.** Use a
  sentinel + `NULLIF` in SQL (existing code uses `NULLIF(v.owner, '')`; the armies batch uses a
  `-1` sentinel for optional coords, `NULLIF(v.dest_x, -1)`, and passes JSONB as `text[]` cast
  per row with `v.troops::jsonb`). Single-row inserts can use real nullable params.

## Conventions to follow

- **Logging:** always use the context-aware slog calls — `slog.InfoContext(ctx, ...)`,
  `slog.ErrorContext(ctx, ...)`, etc. — with key/value pairs (`"city_id", id`). In actors use
  `state.Ctx()` as the context. Enrich context with `logger.With(ctx, "key", val)` rather than
  formatting values into the message string. Don't introduce a new logger or `fmt.Printf`.
- **Layering:** keep `domain` framework-free. Actors talk to other actors through
  `contracts.ClusterProvider`, never by importing `cluster` directly, and persist through
  `contracts.Store`. Services orchestrate; they don't hold game logic that belongs in an actor.
- **Messages are the contract.** Add a new struct in `internal/messages` and handle it in the
  relevant actor's `Receive` (or a building impl's `Handle`) rather than adding ad-hoc methods.
- **New building types:** add the enum to `domain/building.go` + `common.proto`
  (`BuildingType`), stat entries in `constants/buildings.go`, a `*Impl` in `internal/actors`
  implementing `buildingActorImpl`, a case in `buildingActor.Receive`'s impl switch, and mapping
  entries in `internal/mapping`.
- **New troop types:** add the enum to `domain/army.go` + `common.proto` (`TroopType`), a stat
  entry in `constants/troops.go` (`troopStats`) and add it to `AllTroopTypes`, and mapping
  entries in `internal/mapping` (`troopTypeToProto`/`FromProto`). No new actor is needed — armies
  hold a `map[TroopType]int64`.
- **New terrain types:** add the enum to `domain/terrain.go` + `entity/v1/tile.proto`, update the
  mapping in `internal/mapping`, then define its generation thresholds and buildability in
  `internal/worldgen`.
- **New actor kind:** register it in `internal/cluster/cluster.go`'s `kinds` list (via the
  `spawn` closure so it gets `Store` + logging context injected), add a `New<Kind>Actor`
  constructor implementing `BaseActorInterface`, and wire persistence into `contracts.Store` +
  `internal/persistence` if it has its own table.
- **New streamed entity:** extend `stream.StateUpdate` (+ `recordPublish`), then handle the new
  field in BOTH the `StreamState` initial snapshot and the update loop in `internal/rpc/user.go`,
  and add it to `EntityBag` in proto + `mapping.EntitiesToBag`.
- **Errors:** return them up where a caller can act; otherwise log with context. Match the
  existing pattern in the file you're editing. Custom error types mostly use **pointer**
  receivers — see the pointer/value note below.
- **Pointer vs value in actor responses:** when an actor does `ctx.Respond(&SomeMessage{})`,
  every caller asserting on the result **must** use `res.(*SomeMessage)`, not
  `res.(SomeMessage)`. A pointer/value mismatch silently fails the type assertion. Note the
  `userActor` responds `messages.Ack{}` and `messages.InsufficientGoldError{}` as **values**,
  and the city relays them by value — match exactly what the `ctx.Respond(...)` sends.

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

## Git & PR Conventions

These rules are STRICT and MUST be followed exactly. No exceptions.

### Conventional Commits (REQUIRED)

Every commit message MUST follow the [Conventional Commits](https://www.conventionalcommits.org/) spec:

```
<type>(<optional scope>): <description>

<optional body>

<optional footer>
```

- **Allowed types:** `feat`, `fix`, `docs`, `style`, `refactor`, `perf`, `test`, `build`, `ci`, `chore`, `revert`
- The `<description>` MUST be lowercase, imperative mood ("add" not "added"/"adds"), and MUST NOT end with a period.
- Keep the subject line under 72 characters.
- Use a `!` after the type/scope (e.g. `feat!:`) or a `BREAKING CHANGE:` footer for breaking changes.
- Scope is optional but encouraged; use a short, lowercase noun (e.g. `feat(actors): ...`, `fix(rpc): ...`).
- Put the "why" in the body, not just the "what". Wrap body lines at 72 characters.

### PR Titles (REQUIRED)

- PR titles MUST also follow Conventional Commits format, identical to a commit subject line.
- Example: `feat(army): add nearest-city food-upkeep attribution`
- Do NOT include ticket numbers, emojis, or trailing punctuation in the title.

### PR Descriptions (REQUIRED)

PR descriptions MUST be clean, structured, and complete:

- Start with a short `## Summary` section (1-3 bullet points) explaining what changed and why.
- Include a `## Test plan` section with a markdown checklist of how the change was verified.
- Reference related issues where applicable (e.g. `Closes #123`).
- No filler, no AI attribution, no auto-generated noise.
- Keep it concise and skimmable — reviewers should understand the change without reading every line of the diff.

## Before you finish

Run and make sure these are clean:

```bash
go build ./...
go vet ./...
gofmt -l internal/ cmd/      # should print nothing
```

If you touched proto or SQL, regenerate first (`buf generate` / `sqlc generate`) and commit the
generated output alongside your change.

There is currently **no test suite**. Don't claim something works because it builds — if you
change runtime behavior, exercise it (run the app, drive it via `scripts/troops.py` or curl,
inspect the DB) or say what you verified.
