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

The HTML API documentation is exposed at `http://localhost:8080/docs`, and its source files live in the repository's `docs/` directory while still being embedded into the application binary.

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
| `AUTH_TOKEN_SECRET` | `apostolepis-dev-secret` | HMAC secret used to sign bearer tokens |
| `AUTH_TOKEN_TTL` | `86400` | Bearer token lifetime in seconds |
| `GOOGLE_CLIENT_ID` | `dev-google-client-id.apps.googleusercontent.com` | Google OAuth client ID used to validate identity tokens |

When running the API outside Compose, set `API_ADDRESS`, `MONGO_URI`, `AUTH_TOKEN_SECRET`, `GOOGLE_CLIENT_ID`, and optionally `MONGO_DATABASE`, `MONGO_CONNECT_TIMEOUT`, and `AUTH_TOKEN_TTL` directly.

If `API_ADDRESS` is unset, the API also honors the platform-provided `PORT` environment variable and binds to `:$PORT`. That makes the image compatible with Railway and other container platforms that inject the listening port at runtime.

## Railway Deployment

Railway can build the image directly from the repository `Dockerfile`; `docker-compose.yml` is only for local development.

Set these Railway variables on the service:

| Variable | Required | Notes |
| --- | --- | --- |
| `MONGO_URI` | Yes | Use the MongoDB connection string from Railway MongoDB or another external MongoDB instance |
| `MONGO_DATABASE` | No | Defaults to `apostolepis` |
| `MONGO_CONNECT_TIMEOUT` | No | Defaults to `20` seconds in the app |
| `AUTH_TOKEN_SECRET` | Yes | Secret used to sign and verify bearer tokens |
| `AUTH_TOKEN_TTL` | No | Defaults to `86400` seconds in the app |
| `GOOGLE_CLIENT_ID` | Yes | Google OAuth client ID accepted by the login endpoint |
| `API_ADDRESS` | No | Leave unset on Railway so the app binds to `PORT` automatically |

Recommended Railway settings:

- Builder: `Dockerfile`
- Start command: leave empty and use the image entrypoint
- Healthcheck path: `/health`
- Public networking: expose the HTTP service

## API

| Method | Path | Description |
| --- | --- | --- |
| `GET` | `/health` | API and MongoDB health |
| `GET` | `/docs` | Interactive API documentation |
| `POST` | `/v1/products` | Create a product |
| `GET` | `/v1/products` | List products |
| `GET` | `/v1/products/:id` | Get a product |
| `PUT` | `/v1/products/:id` | Update a product |
| `DELETE` | `/v1/products/:id` | Delete a product |
| `POST` | `/v1/products/:id/stock-adjustments` | Apply a signed stock delta |
| `GET` | `/v1/products/:id/stock` | Get current stock |
| `POST` | `/v1/auth/login` | Validate a Google identity token, create the user if needed, and return a bearer token |
| `GET` | `/v1/users` | List users |
| `GET` | `/v1/users/:id` | Get a user |
| `GET` | `/v1/users/:id/orders` | Get a user's orders |

Product SKUs and user emails are unique. Stock adjustments are atomic and cannot reduce quantity below zero.

Administrative privileges are read from the persisted user document in MongoDB through the `is_admin` field. Protected routes require an `Authorization: Bearer <token>` header. Tokens are issued by `POST /v1/auth/login` after a Google identity token is validated against the configured Google client ID, and admin-only routes still enforce the current `is_admin` value from MongoDB.

Users are not created through a general public CRUD endpoint. They are provisioned through the Google-authenticated flow. The login endpoint validates a Google identity token, extracts the verified Google profile, creates a regular user when that email is not yet present in MongoDB, and then exchanges that user for a signed API token. An administrator is therefore a persisted Google user whose `is_admin` field is `true` in MongoDB.

Example login:

```sh
curl -X POST http://localhost:8080/v1/auth/login \
	-H 'Content-Type: application/json' \
	-d '{"identity_token":"<google-id-token>"}'
```

Example product creation:

```sh
curl -X POST http://localhost:8080/v1/products \
	-H 'Authorization: Bearer <token>' \
	-H 'Content-Type: application/json' \
	-d '{"name":"Field Guide","sku":"FG-1","price":25}'
```

### Bruno Collection

The repository root is a Bruno collection. Open this folder in Bruno and select the `Local` environment to use the ready-made requests for health, docs, products, inventory, and users.

Set `googleIdentityToken`, then call the login request to capture a token for the protected Bruno requests. Update `productId` and `userId` before running ID-based requests.

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