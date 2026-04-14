# Copilot Instructions

## Build, Test, and Lint

```bash
make all          # full pipeline: clean → tidy → format → build → test
make build        # go build ./...
make test         # go test ./...
make test-cached  # go test ./... -count=1  (bypass cache)
make format       # gofmt -s -w -l .
go mod tidy       # sync dependencies
```

Run a single test:
```bash
go test -run TestFunctionName ./slapi/
go test -run TestFunctionName ./sldb/
go test -run TestFunctionName ./slutil/
```

## Architecture

Three packages with a strict dependency direction: `slutil` ← `slapi` / `sldb`

- **`slutil`** — Foundation layer. Defines `ExecContext` (runtime caller info), `ZLObjectKey` (`"sl"`), `ZLJSONItem` (for test deserialization), `StaticNow`/`StaticNowFunc` (deterministic time for tests), context helpers, and message-formatting utilities.
- **`slapi`** — API layer error type. `APIError` embeds `slutil.ExecContext` and carries HTTP-level fields (status code, path, method, request/caller/owner IDs, params).
- **`sldb`** — Database layer error type. `DatabaseError` embeds `slutil.ExecContext` and carries DB-level fields (table, query, operation, constraint). `EnumDBErrorType` maps DB error kinds to HTTP status codes via `.HTTPStatus()`.
- **`example`** — Shows how to wrap a `DatabaseError` inside an `APIError` to propagate errors up the stack.

All zerolog log output is nested under the JSON key `"sl"` (the value of `slutil.ZLObjectKey`).

## Key Conventions

### Error construction

- Use `LogNew(apiErr APIError) error` to **log and return** an `APIError`.
- Use `New(apiErr APIError) error` to **only construct** an `APIError` without logging.
- Use `LogNewDBErr(newDBErr NewDBErr) error` to log and return a `DatabaseError`. Always pass `NewDBErr` (not `DatabaseError` directly) — this keeps `ExecContext` internal.
- Both log functions call `slutil.GetExecContext(3)` internally; never set `ExecContext` manually when using them (except in tests where `GenerateNonRandomAPIError` does it explicitly).

### Context-driven API errors

`LogCtxMsg` and its variants (`LogCtx`, `LogCtxF`, `LogCtxInternal`, etc.) pull fields from a `context.Context` using the string key constants defined in `slapi` (`CallerIDKey`, `RequestIDKey`, `PathKey`, etc.). Always store those values into context using these same constants.

### MarshalZerologObject

Both `APIError` and `DatabaseError` implement `zerolog.LogObjectMarshaler`. Log them via:
```go
log.Error().Object(slutil.ZLObjectKey, &myErr).Send()
```

### Error chain traversal

Use `FindAPIErrors(err)` / `FindOutermostAPIError(err)` and `FindDatabaseErrors(err)` / `FindOutermostDatabaseError(err)` to extract typed errors from wrapped chains. Both types implement `Unwrap()`.

### Testing pattern

Tests redirect zerolog to a temp file to capture and parse JSON output:
1. Create a temp file with `os.CreateTemp("", slutil.TempFileNameAPILogs)` (or `TempFileNameDBLogs`).
2. Set `log.Logger = zerolog.New(tempFile).With().Timestamp().Logger()`.
3. Set `zerolog.TimestampFunc = slutil.StaticNowFunc` for deterministic timestamps.
4. After the action under test, `json.Unmarshal` the file contents into `slutil.ZLJSONItem` to assert fields.
5. Clean up the temp file in a teardown function.
