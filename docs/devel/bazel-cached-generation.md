# Bazel-Cached Code Generation

This document describes how code generation is cached using Bazel to improve development workflow efficiency.

## Overview

The `make generate` command runs various code generation tools. Some of these operations are now cached using Bazel, which means:

- **First run**: Full generation (same as before)
- **Subsequent runs**: If source files haven't changed, Bazel uses cached outputs, significantly reducing generation time

## Cached Generation Targets

The following generation outputs are cached by Bazel:

### Manifest Files (`manifests/generated/`)
- `kubevirt-priority-class.yaml`
- `kv-resource.yaml`
- `kubevirt-network-policies.yaml.in`
- `kubevirt-cr.yaml.in`
- `rbac-operator.authorization.k8s.yaml.in`
- `operator-csv.yaml.in`

### API Documentation
- `api/openapi-spec/swagger.json`

### Observability Documentation
- `docs/observability/metrics.md`

## Make Targets

### `make generate`
Full generation, including both cached and non-cached parts:
- Runs all code generators (deepcopy-gen, client-gen, controller-gen, etc.)
- Uses Bazel caching for manifest and doc generation
- Runs gazelle and buildifier
- Syncs kubevirtci and common-instancetypes

### `make generate-fast`
Fast generation using only Bazel-cached parts:
- Only regenerates Bazel-cached outputs (manifests, openapi spec, docs)
- Runs gazelle and buildifier
- **Use this when**: You only modified code that affects manifest generation

### `make generate-cached`
Generates only the Bazel-cached files:
- Manifest files
- OpenAPI spec
- Metrics documentation

### `make verify-generated-cached`
Verifies that Bazel-cached generated files are up-to-date:
- Useful in CI pipelines
- Returns non-zero exit code if files are stale

## How It Works

1. **Bazel genrules** in `BUILD.bazel` define the generation targets
2. These rules use the same generator tools (`resource-generator`, `csv-generator`, etc.)
3. Bazel tracks input files and caches outputs in `bazel-bin/`
4. The `hack/sync-generated-from-bazel.sh` script copies outputs to the source tree

## Bazel Targets

Individual generation targets in `//:BUILD.bazel`:

```
//:generate-kubevirt-priority-class
//:generate-kv-resource
//:generate-kubevirt-network-policies
//:generate-kubevirt-cr
//:generate-rbac-operator
//:generate-operator-csv
//:generate-openapi-spec
//:generate-metrics-doc
```

Grouped targets:
```
//:generated-manifests    # All manifest files
//:all-generated          # All cached generation outputs
```

## Building Individual Targets

You can build individual generation targets directly:

```bash
# Build all cached generation outputs
bazel build //:all-generated

# Build only manifest files
bazel build //:generated-manifests

# Build only OpenAPI spec
bazel build //:generate-openapi-spec
```

## Extending Cached Generation

To add new cached generation:

1. Add a `genrule` target in `BUILD.bazel`
2. Update `hack/sync-generated-from-bazel.sh` to sync the new output
3. Add the new target to the appropriate filegroup (`:generated-manifests` or `:all-generated`)

Example genrule:
```starlark
genrule(
    name = "generate-my-file",
    srcs = [],
    outs = ["generated-dir/my-file.yaml"],
    cmd = "$(location //tools/my-generator) --flag=value > $@",
    tools = ["//tools/my-generator"],
    visibility = ["//visibility:public"],
)
```

## Troubleshooting

### Cache misses
If Bazel rebuilds when you expect caching:
1. Check if any source files changed that are dependencies of the generators
2. Verify Bazel's cache directory is intact
3. Use `bazel query` to inspect dependencies

### Stale generated files
If `make verify-generated-cached` fails:
```bash
make generate-cached
```

### Inspecting cache state
```bash
# Check what would be built
bazel build //:all-generated --nobuild

# Show action graph
bazel aquery //:all-generated
```
