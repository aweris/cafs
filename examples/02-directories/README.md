# Directories

Directory hashing and change detection.

## Run

```bash
go run .
```

## What it shows

- `Index().Hash(prefix)` — Compute hash of all entries under prefix
- `Index().List(prefix)` — Iterate entries under prefix
- Change detection via hash comparison
- Deduplication (same content = same digest)
