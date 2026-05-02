# Architecture

sularo is structured as a small Go module with three layers:

```
cmd/sularo/          # CLI wiring (cobra commands, flag parsing)
internal/test/       # Core logic
  discover.go        # Walk ./tests/, build []Case
  runner.go          # Execute cases, return []Result
  diff.go            # Parse YAML docs, subset-match, go-cmp diff
  update.go          # Rewrite expected.yaml from live render
  watch.go           # fsnotify loop, debounce, screen clear
  init.go            # Scaffold new test case dirs
  result.go          # Result / Status types
  schema/            # CRD loading and JSON Schema validation
    loader.go        # Parse CRDs, compile schemas into Registry
    validator.go     # Validate []map[string]any against Registry
  format/            # Output formatters
    format.go        # Formatter interface
    tap.go           # TAP writer
    junit.go         # JUnit XML writer
    json.go          # JSON writer
```

## Data flow

```
Discover(root)
  └─ for each dir: find xr*.yaml files
       └─ read annotations → resolve composition + functions paths
       └─ produce []Case

Run(root, filter)
  ├─ schema.LoadDir("./crds")   → Registry
  ├─ Discover(root)             → []Case
  └─ for each Case:
       ├─ crossplane render xr composition [--function-runtime-configs functions]
       ├─ Registry.Validate(rendered docs)   [skipped if registry empty]
       ├─ Diff(rendered, expected)           [subset match via projectSubset]
       └─ Result{Name, Status, Message, Duration}

format.Formatter.Write(w, []Result)
  └─ TAP | JUnit | JSON
```

## Key design decisions

**Subset matching over exact match.** Golden files declare only the fields the author cares about. `projectSubset` recursively copies only the keys present in `expected` from the actual output before running `go-cmp`. This avoids noise from auto-generated fields (UIDs, timestamps, ownerReferences) that change between runs.

**Annotation-based composition resolution.** A single composition directory can contain many compositions for the same XR kind. Explicit annotation references (`sularo.crossplane.io/composition`) are unambiguous and scale to large repos, unlike kind-matching which breaks as soon as two compositions target the same XR type.

**No import cycle between `test` and `test/format`.** The `format` package imports `test` (for `Result`), so `test` cannot import `format`. Watch mode passes a callback (`func(changedFile string)`) from `main.go` into `Watch()` instead of accepting a `Formatter` — the caller is responsible for formatting.

**Schema validation is opt-in and non-breaking.** If `./crds/` is absent, validation is skipped entirely. This lets existing repos adopt sularo before they've organised their CRDs.
