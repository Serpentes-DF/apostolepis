# Apostolepis

A Go 1.25 and Gin API for products, inventory, and users, backed by MongoDB.

## Requirements

- Docker Engine with Docker Compose v2
- Make (optional convenience commands)

## Development

Copy `.env.example` to `.env` and adjust its non-production development values if needed. Start the API and its persistent development MongoDB:

```sh
docker compose up --build
```

The API is available at `http://localhost:8080` by default. MongoDB is not exposed to the host. The API waits for MongoDB's authenticated healthcheck before starting.

Equivalent Make commands:

```sh
make up
make down
```

`make down` removes development containers and networks but preserves the `mongodb-data` volume.

### Configuration

| Variable | Default | Purpose |
| --- | --- | --- |
| `API_PORT` | `8080` | API port exposed on the host by Compose |
| `MONGO_ROOT_USERNAME` | `apostolepis` | Development MongoDB root username |
| `MONGO_ROOT_PASSWORD` | `apostolepis-dev` | Development MongoDB root password |
| `MONGO_DATABASE` | `apostolepis` | Application database name |
| `MONGO_CONNECT_TIMEOUT` | `20` | MongoDB connection timeout in seconds |

When running the API outside Compose, set `API_ADDRESS`, `MONGO_URI`, `MONGO_DATABASE`, and optionally `MONGO_CONNECT_TIMEOUT` directly.

## API

| Method | Path | Description |
| --- | --- | --- |
| `GET` | `/health` | API and MongoDB health |
| `POST` | `/api/v1/products` | Create a product |
| `GET` | `/api/v1/products` | List products |
| `GET` | `/api/v1/products/:id` | Get a product |
| `PUT` | `/api/v1/products/:id` | Update a product |
| `DELETE` | `/api/v1/products/:id` | Delete a product |
| `POST` | `/api/v1/products/:id/stock-adjustments` | Apply a signed stock delta |
| `GET` | `/api/v1/products/:id/stock` | Get current stock |
| `GET` | `/api/v1/users` | List users |
| `GET` | `/api/v1/users/:id` | Get a user |

Product SKUs and user emails are unique. Stock adjustments are atomic and cannot reduce quantity below zero.

Administrative privileges are read from the persisted user document in MongoDB through the `is_admin` field. Protected routes require the `X-User-Email` request header and currently include product creation, product updates, product deletion, and stock adjustments.

Users are not created through a public HTTP endpoint. They must be provisioned by the Google-authenticated flow, which is the only supported creation path. An administrator is therefore a persisted Google user whose `is_admin` field is `true` in MongoDB.

Example product creation:

```sh
curl -X POST http://localhost:8080/api/v1/products \
	-H 'X-User-Email: admin@example.com' \
	-H 'Content-Type: application/json' \
	-d '{"name":"Field Guide","sku":"FG-1","price":25}'
```

## Testing

Tests live inside their owning Go packages. Test files use descriptive names ending in either `_unit_test.go` or `_integration_test.go`. Explicit `unit` and `integration` build tags ensure the suites never overlap.

### Run unit tests

Unit tests require no MongoDB or other infrastructure:

```sh
docker compose -f docker-compose.test.yml run --rm api-test-unit
```

Use `--build` to force rebuilding the test image, or run the Make target, which always builds:

```sh
make test-unit
```

The underlying command is `go test -tags=unit ./...`.

### Run integration tests

Integration tests start a real, ephemeral MongoDB with separate credentials, network, and storage from development:

```sh
docker compose -f docker-compose.test.yml up --build \
	--abort-on-container-exit \
	--exit-code-from api-test-integration \
	api-test-integration
docker compose -f docker-compose.test.yml down --volumes --remove-orphans
```

The requested shorter command also runs only integration tests and stops the stack when the test container exits:

```sh
docker compose -f docker-compose.test.yml up --build --abort-on-container-exit api-test-integration
```

Use the Make target in development and CI to propagate the test exit status and always remove test resources:

```sh
make test-integration
```

The underlying test command is `go test -tags=integration -count=1 ./...`. Each integration test creates a unique database and drops it during cleanup.

### Run all tests

```sh
make test
```

This runs unit tests first. Integration infrastructure starts only if every unit test passes.