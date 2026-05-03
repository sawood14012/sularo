# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project

`sularo` is a Go CLI test harness for [Crossplane](https://crossplane.io) compositions. It walks `./tests/`, runs `crossplane render` on each XR, optionally validates the output against CRD schemas in `./crds/`, and diffs the result against a golden `expected.yaml` using subset matching.

The `crossplane` CLI must be on `$PATH` for `sularo test` and `sularo update` to actually render anything — pure unit tests (`go test ./...`) do not need it.

## Common commands

```bash
# Build and vet
go build ./...
go vet ./...

# Unit tests
go test ./...
go test ./internal/test -run TestDiff   # single test

# End-to-end against the bundled fixtures (requires crossplane CLI)
go run ./cmd/sularo test
go run ./cmd/sularo test --filter example-vpc
go run ./cmd/sularo test --watch
go run ./cmd/sularo update --filter example-vpc   # refresh golden files
```

CI (`.github/workflows/ci.yml`) runs only `go vet`, `go build`, `go test` — it does not exercise the `crossplane` integration path.

## Architecture

Three layers, with a strict no-cycle rule between `internal/test` and `internal/test/format`:

- `cmd/sularo/main.go` — Cobra wiring for `test`, `update`, `init`. Picks the formatter (`tap` | `junit` | `json`) and, in `--watch` mode, supplies a callback to `test.Watch` that re-runs `test.Run` and writes results.
- `internal/test/` — core logic. `Discover` walks `./tests/` and returns one `Case` per `xr*.yaml` file (suffix variants like `xr-private.yaml` produce a `<dir>/<suffix>` case name with a matching `expected-<suffix>.yaml`). `Run` shells out to `crossplane render`, runs optional schema validation, then diffs.
- `internal/test/runtime.go` — `dockerHostEnv()` auto-detects a Podman socket when no Docker socket is present and the user hasn't set `DOCKER_HOST`. `render()` injects the result into the subprocess env so `crossplane render` works on Docker- or Podman-only machines without manual setup. An explicit `DOCKER_HOST` always wins.
- `internal/test/format/` — output writers (TAP, JUnit, JSON). Imports `test` for the `Result` type. **`test` must never import `format`** — that's why `Watch` takes a `func(changedFile string)` callback instead of a `Formatter`.
- `internal/test/schema/` — loads CRDs from `./crds/` (v1 and v1beta1, stripping `x-kubernetes-*` and `nullable`) into a `Registry` and validates rendered docs by GVK. Empty registry ⇒ validation skipped silently (`./crds/` is opt-in).

### Composition + functions resolution

For each test case, `resolveComposition` / `resolveFunctions` (in `discover.go`) check, in order:
1. A local file in the test dir (`composition.yaml` / `functions.yaml`).
2. The annotation on the XR (`sularo.crossplane.io/composition`, `sularo.crossplane.io/functions`).

Annotation paths are **relative to the repo root** (where `sularo` is invoked), not the test dir. Composition is required (test fails otherwise); functions is optional. `sularo.crossplane.io/skip: "true"` short-circuits resolution and produces a `StatusSkip` result.

### Subset diffing

Golden `expected.yaml` files declare only the fields that matter. `projectSubset` in `diff.go` recursively copies only the keys present in `expected` from the rendered output before handing both to `go-cmp`. This is intentional — it filters out auto-generated noise (UIDs, timestamps, ownerReferences). Slices are compared verbatim. When extending the diff logic, preserve this asymmetry.

## Conventions (from docs/contributing.md)

- No comments explaining *what* the code does — names do that. One short comment only when the *why* is non-obvious.
- No backwards-compatibility shims — just change the code.
- Keep `internal/test` ↔ `internal/test/format` free of import cycles (see above).

### Adding a new output format

1. Create `internal/test/format/<name>.go` implementing `format.Formatter`.
2. Register it in the `switch outputFormat` block in `cmd/sularo/main.go`.
3. Update the `--format` flag description.

### Adding a new annotation

1. Add the constant to `internal/test/discover.go`.
2. Read it in `xrAnnotations()` and thread it through `Case`.
3. Document it in the Annotations reference in `README.md`.

## Releases

Two-stage on push to `main`:
1. `.github/workflows/release.yml` — semantic-release reads conventional commits (`feat` ⇒ minor, `fix`/`perf`/`refactor` ⇒ patch, `BREAKING CHANGE` ⇒ major; `docs`/`chore`/`test` ⇒ no release) and tags `vX.Y.Z`.
2. `.github/workflows/goreleaser.yml` — fires on the new `v*` tag and builds binaries via `.goreleaser.yml`. The CLI's `version`/`commit`/`date` vars in `cmd/sularo/main.go` are populated through `-ldflags` by goreleaser.

Use conventional commit prefixes so the release pipeline behaves correctly.
