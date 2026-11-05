# AGENTS.md

Follow the project coding conventions in [CONVENTIONS.md](CONVENTIONS.md).

## Transactions

Use `repositories.Transactioner` and transaction-aware repositories when several writes must succeed or fail together. See `internal/adapters/outbound/persistence/gormstore` for the GORM implementation. Bind every participating repository with `WithTx` inside the callback, check its error, and propagate the request context.

## Validation

Run `make check` and `make lint` for changes. Document any check that could not run.
