# Quickstart

Basic cafs operations in ~20 lines.

## Run

```bash
go run .
```

## What it shows

- `Blobs().Put()` — Store content, get digest
- `Index().Set()` — Map key to digest
- `Index().Get()` — Lookup digest by key
- `Blobs().Get()` — Load content by digest
- `Push()` — Sync to OCI registry
