# gocacheprog

GOCACHEPROG implementation using CAFS (local only).

Use `cafs` CLI for remote sync operations.

## Build

```bash
go build -o gocacheprog .
```

## Usage

```bash
# Local only
export GOCACHEPROG=$PWD/gocacheprog
go build ./...

# With remote sync (use cafs CLI)
export GOCACHEPROG=$PWD/gocacheprog
export GOCACHEPROG_REF=ttl.sh/myorg/gocache:main

cafs pull $GOCACHEPROG_REF     # warm up cache
go build ./...                  # fast local builds
cafs push $GOCACHEPROG_REF     # sync back
```

## Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `GOCACHEPROG_REF` | `gocache/default:main` | Namespace ref |
| `GOCACHEPROG_DIR` | `~/.cache/cafs` | Local storage |

## Demo

Run the full workflow demo:

```bash
./demo.sh
```
