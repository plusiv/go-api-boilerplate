# Go API boilerplate

A small HTTP API starter using Go 1.27 and Echo v5 with framework-independent domain and application layers. Replace the example item feature when starting a project.

## Run

Install Go 1.27 and run `make run`. The server listens on `:8080` by default. Set `HTTP_ADDR` to change the listen address, for example `HTTP_ADDR=127.0.0.1:9000 make run`.

```sh
curl http://localhost:8080/healthz
curl -X POST http://localhost:8080/items -H 'Content-Type: application/json' -d '{"id":"1","name":"Example"}'
curl 'http://localhost:8080/items?name=Example&limit=20&offset=0'
curl http://localhost:8080/items/1
```

The repository stores data **in memory**. It is empty after a restart and is not shared between instances. IDs are supplied by the client for this example. This starter has no authentication, authorization, persistent storage, or deployment configuration; add these for your application.

## Architecture

- `internal/domain`: entities, typed application errors, and repository ports (interfaces) with no web or database framework dependencies.
- `internal/application/usecases`: validation and orchestration against the ports.
- `internal/adapters/inbound/http`: Echo routes and transport mapping.
- `internal/adapters/outbound/memory`: a concurrency-safe example repository.
- `cmd/api`: composition root, where concrete implementations are connected.
- `internal/server`: HTTP server lifecycle and graceful shutdown.
- `internal/infrastructure/logger`: request-scoped structured logger helpers.
- `internal/infrastructure/config`: typed environment and dotenv configuration.

A repository filter combines its present predicates with AND. `In: nil` means no membership constraint; an empty non-nil `In` matches nothing. `NotIn` excludes its values. List filters run before pagination. Results are sorted by ID, then paginated with a zero-based offset and a limit from 1 to 100. Offset pages may shift when concurrent writes occur; use keyset pagination if stable traversal across changes is required. The HTTP example exposes exact `name` matching, `limit`, and `offset`; the remaining filter operators are available at the repository boundary.

When adding a database adapter, preserve these semantics, bind values as query parameters, allowlist any sortable column names, propagate context, map storage errors to domain errors, and run the same behavior tests against the real database. Do not assume SQL NULL comparisons behave like Go values.

## Configuration

Configuration is loaded from environment variables and an optional `.env` file in the working directory. Existing environment variables take precedence over `.env` values. Invalid settings stop the application before it starts listening. Copy `.env.example` to `.env` for local development; do not commit secrets.

| Variable | Default | Description |
| --- | --- | --- |
| `HTTP_ADDR` | `:8080` | Address on which the server listens. |
| `HTTP_REQUEST_TIMEOUT` | `60s` | Positive Go duration for cooperative request cancellation. |
| `CORS_ALLOWED_ORIGINS` | empty | Comma-separated browser origins; empty disables CORS. |
| `METRICS_ENABLED` | `false` | Enable the authenticated Prometheus scrape endpoint. |
| `METRICS_TOKEN` | empty | Bearer token required when metrics are enabled. |
| `PPROF_ENABLED` | `false` | Enable authenticated HTTP profiling endpoints. |
| `PPROF_TOKEN` | empty | Bearer token required when HTTP profiling is enabled. |

## Error handling

Expected application errors have a category and a client-safe message. The HTTP adapter maps validation and business-rule errors to 422, missing resources to 404, conflicts to 409, removed resources to 410, and rate limits to 429. Internal and unexpected failures are logged with their causes and return a generic 500. Wrap unexpected dependency errors with `eris.Wrap` and `apperrors.NewInternal`; preserve context cancellation and deadline errors.

## HTTP request handling

The validating binder applies `validate` tags after binding input. Invalid fields return HTTP 422 with field names and failed rules; malformed request bodies return HTTP 400. Use cases still enforce their own invariants because they can be called from other transports.

Each request receives an `X-Request-ID` response header. A supplied ID is reused if it is at most 128 characters and contains no spaces or tabs; otherwise a new ID is generated. The request logger is attached to the request context. Use `logger.FromContext(ctx)` to log with the same request ID in downstream code. Access logs include the method, matched route, status, and latency; they omit query strings. Internal errors are logged and their details are not returned to clients.

Requests have a 512 KiB body limit and a cooperative timeout of 60 seconds. Change the timeout with `HTTP_REQUEST_TIMEOUT`, for example `HTTP_REQUEST_TIMEOUT=45s`. Operations must respect context cancellation for the timeout to stop work; it does not force a non-cooperative handler to return. Adjust or skip the middleware and server write timeout for streaming endpoints.

Cross-Origin Resource Sharing (CORS), which allows browser applications on other origins to call the API, is disabled by default. Set `CORS_ALLOWED_ORIGINS` to a comma-separated list of trusted origins, for example `https://app.example.com,https://admin.example.com`. The middleware exposes `X-Request-ID` and does not allow credentials by default. Configure allowed methods and headers in the middleware when your application needs more.

On SIGINT or SIGTERM, the server stops accepting new connections and waits up to 10 seconds for active HTTP requests. It then closes remaining connections if needed. Request contexts are passed to the use case and repository. Timeouts protect request headers, reads, and idle connections. The server write timeout is disabled so it does not interrupt the configurable request timeout; set an appropriate write timeout for your deployment and streaming needs.

## Prometheus metrics

Prometheus collects numeric measurements by periodically requesting a *scrape endpoint*. Set `METRICS_ENABLED=true` and `METRICS_TOKEN` to a long random secret to enable `GET /metrics`. It requires `Authorization: Bearer <token>`. For a local check, export the token in your shell and run:

```sh
curl -H "Authorization: Bearer $METRICS_TOKEN" http://localhost:8080/metrics
```

The dedicated registry exposes Go runtime, process, build, and HTTP request metrics. HTTP metrics include request counts, duration, and request and response sizes. Their labels use route patterns instead of individual URLs; unmatched routes share an empty label. The host label is empty and unknown methods are grouped as `OTHER` to limit the number of time series. Scrapes and profiling routes are excluded from HTTP metrics. Additional collectors can be registered in `internal/adapters/inbound/http/metrics.go`.

For example, a Prometheus server on the same host can use:

```yaml
scrape_configs:
  - job_name: go-api
    scrape_interval: 30s
    static_configs:
      - targets: ["127.0.0.1:8080"]
    authorization:
      type: Bearer
      credentials_file: /etc/prometheus/secrets/go-api-token
```

Store the token in the file on the Prometheus server and restrict access to it. Use TLS when the token crosses a network, and restrict the endpoint to trusted clients. Prometheus and its storage are not included in this starter.

## Profiling

To record CPU samples for the lifetime of the process, start the API with:

```sh
go run ./cmd/api -cpuprofile=cpu.pprof
```

Stop the API with SIGINT or SIGTERM to flush the file. Inspect it with `go tool pprof cpu.pprof`. The output path is overwritten if it already exists.

To enable the HTTP profiling endpoints, set `PPROF_ENABLED=true` and set `PPROF_TOKEN` to a long random secret. For example, with the API running locally:

```sh
curl -H "Authorization: Bearer $PPROF_TOKEN" 'http://localhost:8080/metrics/pprof/heap' -o heap.pprof
go tool pprof heap.pprof
```

The index is at `/metrics/pprof/`; CPU profiles, traces, command-line arguments, symbols, and runtime profiles are available under that path. Profiling requests bypass the normal request timeout so long captures can finish. Profile data can reveal sensitive application information and consume resources. Keep the endpoint on a trusted network, use TLS when sending the token over a network, and restrict access. Profiles may also contain sensitive information. CPU profiling through the file flag and the HTTP endpoint cannot run at the same time.

## Checks

Run `make check` for formatting validation, vet, build, and race-enabled tests. Run `make lint` with a recent golangci-lint v2 built with a Go version at least as new as this module. `make fmt` formats Go files. `make clean` removes build output.
