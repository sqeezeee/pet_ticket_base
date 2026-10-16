# Starter refresh — 2026-09-21

This is the starting project, not completed solutions to the exercises.

* API, `make migrate-up`, and integration tests use golang-migrate with embedded SQL in `internal/infra/migration/migrations/`.
* The obsolete root Goose migrations are no longer shipped. There is no SQL generation step.
* Existing Goose databases are refused explicitly. Preserve the old database and create a new empty local database; set POSTGRES_DATABASE to it. Do not force a migration version or replay the initial schema over existing data. If old data is needed, plan a separate reviewed data transfer.
* Service transactions use `tickets.ContextWithTx` / `tickets.TxFromContext`. Reads in a transaction use SELECT FOR UPDATE before modifying the current ticket.
* POST /api/v1/tickets/:id/escalate returns 501 until Task 1 is implemented. Implement four priority levels, one-level escalation up to Critical, atomic persistence plus history, and a no-op at Critical. Do not restore the old comment-only handler.
* Assignment is still a demonstration of a status transition; it does not store operator ownership. Implement persisted assignment and concurrency control in the corresponding exercise.
* X-User-ID is a development identity stub, not production authentication. Do not expose this educational server publicly.

Run `go test ./...`, `go test -race ./...`, and the integration suite with Docker. Without Docker, PET_TICKET_TEST_DSN may point ONLY to a dedicated disposable local test database; each test creates and retains an isolated schema for inspection.
