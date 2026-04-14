# Copilot Instructions

## Build, Test, and Lint

```bash
just                  # clean + tidy + format + build + test (default recipe)
go build ./...
go test ./...
go test ./... -count=1        # bypass test cache
go test -run TestName ./slapi/  # run a single test in a package
gofmt -s -w -l .              # format
go mod tidy
```

## Architecture

This is a Go library providing structured JSON error logging via [zerolog](https://github.com/rs/zerolog). It has four packages plus a testutil helper:

- **`slutil`** — Foundation layer. Provides `ExecContext` (runtime caller capture via `runtime.Caller`), the generic `FromCtxSafe[T]` context helper, message formatting helpers (`PrettyErrMsg`, `PrettyInfoMsg`, etc.), and `ZLJSONItem` (the top-level zerolog JSON shape).
- **`slutil/testutil`** — Test helpers for use in tests across this module. Provides `StaticNow`/`StaticNowFunc` (deterministic timestamps), `TempFileName*` constants, and `UneraseMapString`/`UneraseMapStringArray` (re-type `map[string]any` after JSON round-trip). Import this in `_test.go` files only.
- **`slapi`** — API layer errors. `APIError` embeds `slutil.ExecContext` and implements `zerolog.LogObjectMarshaler`. Provides `LogCtxMsg`/`LogNew` (log + return error) and `New` (return only). Context values are extracted via `FromCtxSafe` using the string key constants defined in this package.
- **`sldb`** — Database layer errors. `DatabaseError` with `EnumDBErrorType` (string enum with `Valid()`, `HTTPStatus()`, `String()` methods). Uses a separate `NewDBErr` input type so callers don't have to set the public `ExecContext` field directly.
- **`sllm`** — LLM provider errors. `LLMError` follows the same pattern as `sldb`, capturing `Provider`, `Model`, `ErrorCode`, `ErrorType`, `FinishReason`, token counts, and `RequestID`. Use `NewLLMErr` as the input type. See the doc comment in `sllm/llm_error.go` for the OpenAI SDK bridge pattern.

All structured log entries are nested under the JSON key `"sl"` (`slutil.ZLObjectKey`). A logged entry looks like:
```json
{"level":"error","sl":{...fields...},"time":"..."}
```

## Key Conventions

**`ExecContext` depth parameter**: `GetExecContext(caller int)` captures the runtime call stack. All public `Log*`/`New*` functions pass `3` to skip past themselves and `addDefaults` to land on the actual caller. If you add a new wrapper layer, increment accordingly.

**Embed `ExecContext`, don't name it**: Both error structs embed `slutil.ExecContext` without a field name. This flattens the fields into the parent struct. The JSON tag `json:"execContext"` is present but struct embedding means the key does NOT appear in zerolog output — fields are serialized directly via `MarshalZerologObject`.

**`NewXxxErr` input types**: `ExecContext` fields must be public for `json.Marshal` in tests, but callers shouldn't set them manually. Each error package exposes a `NewXxxErr` input struct (without `ExecContext`) as the argument to `LogNewXxxErr`.

**Error wrapping**: All error types implement `Unwrap()`. Use `errors.As`/`errors.Is` for type-checking. `FindXxxErrors`/`FindOutermostXxxError` walk the full chain.

**Test pattern**: Tests redirect zerolog to a temp file (name from `slutil/testutil`), call the function under test, then `Sync()`+`Close()` the file, read and `json.Unmarshal` the contents into `slutil.ZLJSONItem`, and assert on both the returned error struct and the JSON output. Use `testutil.StaticNowFunc` for `zerolog.TimestampFunc` to get deterministic timestamps.

**Context keys**: `slapi` defines string constants for all context keys (`CallerIDKey`, `RequestIDKey`, etc.). These same strings are used as JSON field names in `MarshalZerologObject`.

**Test helpers stay in `slutil/testutil`**: Never add test helpers (`StaticNow`, `TempFileName*`, `UneraseMap*`, fixture generators) to non-`_test.go` files in production packages.
