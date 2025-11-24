# Nested Directories Example

Clean, simple demonstration of nested directory operations.

## What it shows

- **Automatic directory creation** - Write to `/src/handlers/users.go` and all parents are created
- **Reading files** - Access any file by path
- **Listing directories** - See what's in a specific folder
- **Walking the tree** - Traverse entire structure with `fs.WalkDir`
- **Eager metadata loading** - All file sizes loaded upfront for fast walks

## Run it

```bash
go run .
```

## Directory Structure

```
/
├── README.md (16 bytes)
├── src/
│   ├── main.go (28 bytes)
│   └── handlers/
│       ├── users.go (36 bytes)
│       └── orders.go (37 bytes)
├── config/
│   └── app.yaml (26 bytes)
└── docs/
    └── api.md (19 bytes)
```

All 6 files across 4 directories created with simple `WriteFile()` calls!
