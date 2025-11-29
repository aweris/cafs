# Remote Sync

Push and pull to OCI registries.

## Run

```bash
go run .
```

## What it shows

- `Push(ctx)` — Upload to remote registry
- `Push(ctx, "v1", "latest")` — Push to multiple tags
- `WithAutoPull("always")` — Pull on open
- `Pull(ctx)` — Manual pull

## Notes

Uses [ttl.sh](https://ttl.sh) - free anonymous registry with 1-hour expiry.
