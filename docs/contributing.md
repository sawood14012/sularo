# Contributing

## Prerequisites

- Go 1.21+
- `crossplane` CLI on `$PATH` (for running integration tests)

## Build

```bash
go build ./...
go vet ./...
```

## Run tests

```bash
# Unit tests
go test ./...

# End-to-end (requires crossplane CLI)
go run ./cmd/sularo test
```

## Adding a new output format

1. Create `internal/test/format/<name>.go` implementing `format.Formatter`.
2. Register it in the `switch outputFormat` block in `cmd/sularo/main.go`.
3. Add it to the `--format` flag description.

## Adding a new annotation

1. Add the constant to `internal/test/discover.go`.
2. Read it in `xrAnnotations()` and thread it through `Case`.
3. Document it in the [Annotations reference](../README.md#annotations-reference).

## Project conventions

- No comments explaining what the code does — names do that.
- One short comment only when the *why* is non-obvious.
- No backwards-compatibility shims — just change the code.
- Keep `internal/test` and `internal/test/format` free of import cycles. The boundary is: `format` imports `test` for `Result`; `test` never imports `format`.
