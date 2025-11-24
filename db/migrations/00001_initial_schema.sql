-- +goose Up
-- +goose StatementBegin
CREATE TYPE coordinates AS (
    x int,
    y int
);

CREATE TABLE users (
    user_id     VARCHAR(36) PRIMARY KEY,
    email       VARCHAR(100) NOT NULL UNIQUE,
    username    VARCHAR(100) NOT NULL UNIQUE,
    password    VARCHAR(64)  NOT NULL,
    gold        BIGINT NOT NULL CHECK (gold >= 0),
    food        BIGINT NOT NULL CHECK (food >= 0),
    created_at  TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE TABLE cities (
    city_id         VARCHAR(36) PRIMARY KEY,
    type            VARCHAR(100) NOT NULL,
    owner           VARCHAR(36) NULL,
    name            VARCHAR(100) NOT NULL,
    population      DOUBLE PRECISION NOT NULL DEFAULT 0 CHECK (population >= 0),
    population_cap  DOUBLE PRECISION NOT NULL DEFAULT 0 CHECK (population_cap >= 0),
    start_coords    COORDINATES NOT NULL,
    size            INTEGER NOT NULL,
    created_at      TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMP NOT NULL DEFAULT NOW(),

    CONSTRAINT city_xy_unique UNIQUE (start_coords)
);

CREATE TABLE buildings (
    building_id         VARCHAR(36) PRIMARY KEY,
    city_id             VARCHAR(36) NOT NULL,
    type                VARCHAR(100) NOT NULL,
    level               INTEGER NOT NULL DEFAULT 0 CHECK (level >= 0),
    target_level        INTEGER NOT NULL DEFAULT 1 CHECK (target_level >= 1),
    coords              COORDINATES NOT NULL,
    construction_start  TIMESTAMP NULL,
    construction_end    TIMESTAMP NULL,
    created_at          TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMP NOT NULL DEFAULT NOW(),

    CONSTRAINT buildings_coords_unique UNIQUE (coords),

    CONSTRAINT buildings_city_fk
        FOREIGN KEY (city_id) REFERENCES cities (city_id)
        ON DELETE CASCADE
);
-- +goose StatementEnd


-- +goose Down
-- +goose StatementBegin
DROP TABLE buildings;
DROP TABLE cities;
DROP TABLE users;
DROP TYPE coordinates;
-- +goose StatementEnd
