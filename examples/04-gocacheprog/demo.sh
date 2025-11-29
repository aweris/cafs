#!/bin/bash
set -e

# Demo: CLI + gocacheprog workflow
# Shows how to use cafs CLI for remote sync with gocacheprog for local builds

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"

REF="${GOCACHEPROG_REF:-ttl.sh/demo/gocache:main}"
CACHE_DIR="${GOCACHEPROG_DIR:-/tmp/cafs-demo}"

echo "=== Building tools ==="
cd "$REPO_ROOT"
go build -o /tmp/cafs ./cmd/cafs
cd "$SCRIPT_DIR"
go build -o /tmp/gocacheprog .

echo "=== Pull from remote (warm up cache) ==="
/tmp/cafs pull "$REF" --cache-dir "$CACHE_DIR" || echo "No remote data yet"

echo "=== Build with gocacheprog (local only, fast) ==="
export GOCACHEPROG=/tmp/gocacheprog
export GOCACHEPROG_REF="$REF"
export GOCACHEPROG_DIR="$CACHE_DIR"

cd "$REPO_ROOT/examples/01-quickstart"
go build -a .

echo "=== Push to remote ==="
/tmp/cafs push "$REF" --cache-dir "$CACHE_DIR"

echo "=== List entries ==="
/tmp/cafs list "$REF" --cache-dir "$CACHE_DIR" | head -10

echo "=== Done ==="
