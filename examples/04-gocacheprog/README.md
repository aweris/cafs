# gocacheprog

GOCACHEPROG implementation using CAFS.

## Build

```bash
go build -o gocacheprog .
```

## Usage

```bash
# Basic (local only)
export GOCACHEPROG=$PWD/gocacheprog
go build ./...

# With remote sync (standard Docker image ref format)
export GOCACHEPROG=$PWD/gocacheprog
export GOCACHEPROG_REF=ttl.sh/myorg/gocache:main
export GOCACHEPROG_PULL=true
export GOCACHEPROG_PUSH=true
go build ./...
```

## Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `GOCACHEPROG_REF` | `gocache/default:main` | Full image ref (registry/repo:tag) |
| `GOCACHEPROG_PULL` | `false` | Pull on start |
| `GOCACHEPROG_PUSH` | `false` | Push on close |
| `GOCACHEPROG_DIR` | `~/.cache/cafs` | Local storage |
