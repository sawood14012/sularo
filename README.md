# sularo

A minimal test harness for [Crossplane](https://crossplane.io) compositions. Each test case renders a composition with a given XR claim and diffs the output against a golden file — no config file, no server, no magic.

```
$ sularo test
1..3
ok 1 - example-vpc
ok 2 - example-vpc/private
not ok 3 - example-rds
    --- expected
    +++ got
    ...
```

---

## Table of contents

- [Installation](#installation)
- [Quick start](#quick-start)
- [Project layout](#project-layout)
- [Commands](#commands)
  - [sularo test](#sularo-test)
  - [sularo update](#sularo-update)
  - [sularo init](#sularo-init)
- [Test case structure](#test-case-structure)
  - [Single variant](#single-variant)
  - [Multiple variants](#multiple-variants)
  - [Skipping a test](#skipping-a-test)
- [Composition discovery](#composition-discovery)
- [Function pipelines](#function-pipelines)
- [Schema validation](#schema-validation)
- [Output formats](#output-formats)
- [Watch mode](#watch-mode)
- [CI integration](#ci-integration)
- [Annotations reference](#annotations-reference)

---

## Installation

```bash
go install github.com/sawood14012/sularo/cmd/sularo@latest
```

Or build from source:

```bash
git clone https://github.com/sawood14012/sularo
cd sularo
go build -o sularo ./cmd/sularo
```

**Prerequisites:**
- The `crossplane` CLI must be on your `$PATH`. Install it from [docs.crossplane.io](https://docs.crossplane.io/latest/cli/).
- A container runtime is required to render compositions that use a function pipeline. Docker and Podman both work — sularo auto-detects a running Podman socket when Docker isn't present and sets `DOCKER_HOST` for the `crossplane render` subprocess. Set `DOCKER_HOST` explicitly to override.

---

## Quick start

```bash
# 1. Scaffold a new test case from an existing XR
sularo init my-vpc \
  --xr path/to/your/xr.yaml \
  --composition compositions/vpc.yaml \
  --functions functions/pat.yaml

# 2. Populate the golden file from the live render
sularo update --filter my-vpc

# 3. Run the tests
sularo test

# 4. Edit your composition, re-run to see what changed
sularo test --watch
```

---

## Project layout

```
.
├── cmd/sularo/main.go          # CLI entry point
├── compositions/               # Shared compositions (referenced by annotation)
│   └── example-vpc.yaml
├── functions/                  # Function runtime configs
│   └── patch-and-transform.yaml
├── crds/                       # Optional: CRD schemas for pre-diff validation
│   └── xvpc.yaml
└── tests/                      # One subdirectory per test case
    └── example-vpc/
        ├── xr.yaml             # Input XR/claim (with sularo annotations)
        ├── xr-private.yaml     # Second variant (optional)
        ├── expected.yaml       # Golden output for xr.yaml
        └── expected-private.yaml  # Golden output for xr-private.yaml
```

---

## Commands

### sularo test

Runs all test cases under `./tests/` and prints results.

```bash
sularo test [flags]
```

| Flag | Default | Description |
|------|---------|-------------|
| `--filter <substring>` | | Run only cases whose name contains the substring |
| `--format <fmt>` | `tap` | Output format: `tap`, `junit`, `json` |
| `--watch` | | Re-run on file changes (see [Watch mode](#watch-mode)) |
| `-v, --verbose` | | Print extra detail for passing tests |

**Exit codes:** `0` if all tests pass (or are skipped), `1` if any fail.

```bash
# Run all tests
sularo test

# Run a single test case
sularo test --filter example-vpc

# Run a specific variant
sularo test --filter example-vpc/private

# Output JUnit XML for CI
sularo test --format junit > results.xml

# Stream JSON results
sularo test --format json
```

---

### sularo update

Re-renders every test case and overwrites its `expected.yaml` with the current output. Use this after intentionally changing a composition to keep golden files in sync.

```bash
sularo update [flags]
```

| Flag | Default | Description |
|------|---------|-------------|
| `--filter <substring>` | | Update only cases whose name contains the substring |

```bash
# Update all golden files
sularo update

# Update a single case
sularo update --filter example-vpc
```

Skipped test cases (via `sularo.crossplane.io/skip`) are reported but not modified.

---

### sularo init

Scaffolds a new test case directory under `./tests/`.

```bash
sularo init <name> --xr <path> --composition <path> [--functions <path>]
```

| Flag | Required | Description |
|------|----------|-------------|
| `--xr` | yes | Path to source XR file to copy from |
| `--composition` | yes | Repo-relative path to the composition |
| `--functions` | no | Repo-relative path to the functions file |

What it does:
1. Reads the source XR file and injects `sularo.crossplane.io/composition` (and optionally `sularo.crossplane.io/functions`) into `metadata.annotations`, preserving existing content.
2. Writes `tests/<name>/xr.yaml`.
3. Creates a stub `tests/<name>/expected.yaml` with a hint to run `sularo update`.

```bash
sularo init my-rds \
  --xr path/to/rds-xr.yaml \
  --composition compositions/rds.yaml \
  --functions functions/pat.yaml

# Then populate the golden file:
sularo update --filter my-rds
```

---

## Test case structure

### Single variant

The minimal test case has two files:

```
tests/my-test/
├── xr.yaml        # Input XR with sularo annotations
└── expected.yaml  # Expected rendered output
```

`xr.yaml` example:

```yaml
apiVersion: example.org/v1alpha1
kind: XVPC
metadata:
  name: my-vpc
  annotations:
    sularo.crossplane.io/composition: compositions/vpc.yaml
    sularo.crossplane.io/functions: functions/pat.yaml
spec:
  region: us-east-1
```

`expected.yaml` uses **subset matching** — only the fields you declare are asserted. Extra fields in the render output are ignored.

```yaml
# Only assert the fields you care about — extras in the render output are ignored
apiVersion: example.org/v1alpha1
kind: XVPC
metadata:
  name: my-vpc
spec:
  region: us-east-1
---
apiVersion: ec2.aws.upbound.io/v1beta1
kind: VPC
spec:
  forProvider:
    region: us-east-1
    cidrBlock: 10.0.0.0/16
```

### Multiple variants

Place additional `xr-<suffix>.yaml` files alongside `xr.yaml`. Each needs a matching `expected-<suffix>.yaml`:

```
tests/example-vpc/
├── xr.yaml               # → test name: example-vpc
├── expected.yaml
├── xr-private.yaml       # → test name: example-vpc/private
└── expected-private.yaml
```

Each variant is an independent test case. Annotations are read per-file, so each variant can point to a different composition if needed.

```bash
# Run only the private variant
sularo test --filter example-vpc/private

# Update only the private variant's golden file
sularo update --filter private
```

### Skipping a test

Add the skip annotation to an XR to exclude it from runs without deleting it:

```yaml
metadata:
  annotations:
    sularo.crossplane.io/skip: "true"
```

Skipped cases appear in all output formats:
- TAP: `ok N - example-vpc # SKIP`
- JUnit: `<skipped/>`
- JSON: `"status": "skip"`

---

## Composition discovery

sularo resolves the composition for each test case in this order:

1. **Local file** — `composition.yaml` inside the test directory.
2. **Annotation** — `sularo.crossplane.io/composition: <path>` on the XR. Path is relative to the repo root.

If neither is found the test fails with a clear error. The annotation approach is recommended — it keeps test directories clean and supports multiple compositions for the same XR kind.

```yaml
annotations:
  sularo.crossplane.io/composition: compositions/vpc.yaml
```

---

## Function pipelines

If your composition uses a function pipeline, provide the function runtime config in one of two ways:

1. **Local file** — `functions.yaml` inside the test directory.
2. **Annotation** — `sularo.crossplane.io/functions: <path>` on the XR.

If neither is present, sularo runs `crossplane render` with just the XR and composition (valid for non-pipeline compositions).

```yaml
annotations:
  sularo.crossplane.io/composition: compositions/vpc.yaml
  sularo.crossplane.io/functions: functions/pat.yaml
```

Example `functions/pat.yaml`:

```yaml
apiVersion: pkg.crossplane.io/v1
kind: Function
metadata:
  name: function-patch-and-transform
spec:
  package: xpkg.upbound.io/crossplane-contrib/function-patch-and-transform:v0.10.4
```

`crossplane render` pulls the function package via Docker on first run. To use a locally running function process instead (e.g. for development), add `render.crossplane.io/runtime: Development` to the Function's annotations and run the function on `localhost:9443`.

---

## Schema validation

If a `./crds/` directory exists, sularo loads all CRD YAML files from it at startup and validates each rendered resource against its CRD's `openAPIV3Schema` **before** running the diff. Resources whose GVK has no matching CRD are silently skipped — so you only need to add CRDs for the resources you want to validate.

```
crds/
├── xvpc.yaml        # CRD for example.org/v1alpha1/XVPC
└── vpc.yaml         # CRD for ec2.aws.upbound.io/v1beta1/VPC
```

Both v1 and v1beta1 CRD formats are supported. Kubernetes-specific extensions (`x-kubernetes-*`, `nullable`) are automatically stripped before validation so standard JSON Schema validators can process them.

A schema failure is reported before the diff step:

```
not ok 1 - example-vpc
    schema validation failed:
      XVPC my-vpc: jsonschema: '/spec/region' does not validate with ...
```

If `./crds/` does not exist, schema validation is silently skipped — fully backwards-compatible.

---

## Output formats

### TAP (default)

Standard [Test Anything Protocol](https://testanything.org/) output, readable by most CI systems and TAP consumers.

```
1..3
ok 1 - example-vpc
ok 2 - example-vpc/private  # SKIP
not ok 3 - example-rds
    --- expected
    +++ got
    ...diff...
```

### JUnit XML

```bash
sularo test --format junit > results.xml
```

```xml
<?xml version="1.0" encoding="UTF-8"?>
<testsuites name="sularo" tests="3" failures="1" skipped="1">
  <testsuite name="sularo" tests="3" failures="1" skipped="1">
    <testcase name="example-vpc" classname="sularo" time="0.12"/>
    <testcase name="example-vpc/private" classname="sularo" time="0.00">
      <skipped/>
    </testcase>
    <testcase name="example-rds" classname="sularo" time="0.09">
      <failure message="diff mismatch">--- expected ...</failure>
    </testcase>
  </testsuite>
</testsuites>
```

### JSON

```bash
sularo test --format json
```

```json
{
  "total": 3,
  "passed": 1,
  "failed": 1,
  "skipped": 1,
  "results": [
    { "name": "example-vpc", "status": "pass", "duration_ms": 120 },
    { "name": "example-vpc/private", "status": "skip", "duration_ms": 0 },
    { "name": "example-rds", "status": "fail", "message": "...", "duration_ms": 90 }
  ]
}
```

---

## Watch mode

```bash
sularo test --watch
```

sularo watches `./tests/`, `./compositions/`, `./functions/`, and `./crds/` for changes. When any file is saved, the suite re-runs automatically (debounced 300ms). The screen is cleared between runs and the changed file is shown at the top.

Works with all other flags:

```bash
# Watch a single test in verbose mode
sularo test --watch --filter my-vpc --verbose

# Watch with JSON output piped to jq
sularo test --watch --format json | jq '.results[] | select(.status=="fail")'
```

Press `Ctrl+C` to stop.

---

## CI integration

### GitHub Actions

```yaml
name: Composition tests
on: [push, pull_request]

jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4

      - name: Install crossplane CLI
        run: |
          curl -sL "https://raw.githubusercontent.com/crossplane/crossplane/master/install.sh" | sh
          sudo mv crossplane /usr/local/bin/

      - name: Install sularo
        run: go install github.com/sawood14012/sularo/cmd/sularo@latest

      - name: Run tests
        run: sularo test --format junit > results.xml

      - name: Publish test results
        uses: EnricoMi/publish-unit-test-result-action@v2
        if: always()
        with:
          files: results.xml
```

### GitLab CI

```yaml
composition-tests:
  image: golang:latest
  script:
    - go install github.com/sawood14012/sularo/cmd/sularo@latest
    - sularo test --format junit > results.xml
  artifacts:
    reports:
      junit: results.xml
```

---

## Annotations reference

All annotations go on the XR's `metadata.annotations`. Paths are always relative to the **repo root** (where you run `sularo`), not the test directory.

| Annotation | Required | Description |
|-----------|----------|-------------|
| `sularo.crossplane.io/composition` | yes* | Repo-relative path to the composition file |
| `sularo.crossplane.io/functions` | no | Repo-relative path to the function runtime config |
| `sularo.crossplane.io/skip` | no | Set to `"true"` to skip this test case |

*Not required if a `composition.yaml` exists inside the test directory (local file takes precedence).
