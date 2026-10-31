# Code conventions

## Architecture

- Keep domain entities and repository ports independent of Echo, databases, and infrastructure.
- Use cases depend on domain ports. Inject concrete adapters at the composition root in `cmd/api`.
- Keep HTTP handlers thin: parse input, call a use case, and map the result or error.
- Pass `context.Context` through operations that can block. Use `logger.FromContext(ctx)` for request-scoped logs and keep request data out of access log query strings. Wrap infrastructure errors and do not expose internal details in HTTP responses.
- Repository implementations must preserve the documented filter, ordering, pagination, cancellation, and error semantics. Bind SQL values and allowlist identifiers. Test persistent adapters against a real database.
- Add environment settings to the typed configuration in `internal/infrastructure/config`, with defaults and validation where needed. Do not commit secrets.
- Keep this starter small and application-agnostic. Add infrastructure only when a project needs it.

## Error handling

- Use `internal/domain/apperrors` for expected application failures: not found, conflict, validation, business rule, rate limit, and gone. Messages on these errors must be safe to return to clients.
- Translate infrastructure-specific failures at the adapter or use-case boundary. Wrap unexpected causes with `eris.Wrap` for operation context, then use `apperrors.NewInternal`; never put secrets, database errors, or stack traces in client messages.
- Preserve `context.Canceled` and `context.DeadlineExceeded` so cancellation and timeout handling can work. Use `errors.Is` and `errors.As` to inspect wrapped causes and typed errors.
- Return errors to the central HTTP error handler. It maps validation and business-rule failures to 422, malformed requests to 400, not found to 404, conflict to 409, gone to 410, and rate limits to 429. Unexpected failures return a generic 500; log causes and stack traces with the request-scoped logger.
- Do not log an error in every layer. Log it at the handling boundary, and never expose an internal cause or an untrusted error message in the response.

## Formatting and tests

Use `gofmt` and add focused tests for behavior and boundaries.
