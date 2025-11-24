# Distributed Workflow Example

Demonstrates multi-machine collaboration with auto-pull and content mixing.

## Workflow

**Machine A** → Creates initial files and pushes
**Machine B** → Pulls, modifies, adds new files, and pushes
**Machine C** → Pulls final version (read-only access)

## What it demonstrates

- **Auto-pull on open** - `WithAutoPullIfMissing()` automatically syncs
- **Content mixing** - Old files (config.json) + new files (guide.md) coexist
- **Read-only mode** - Machine C uses `WithReadOnly()` for safe access
- **Eager prefetch** - Machine C uses `WithPrefetch(["/"])` to load all content upfront
- **Version tracking** - All machines see consistent content hashes

## Run it

```bash
go run .
```

## Output

```
=== Machine A: Creating initial files ===
Pushed snapshot: f305bde879b2ee8d

=== Machine B: Pull, modify, and push ===
Pulled README: # Project Docs
Modified and pushed: 74c43211d8aa7575

=== Machine C: Pull with prefetch (read-only) ===
All content eagerly loaded via prefetch
README: # Project Docs - Updated by B
Guide: # Quick Start Guide
Config: {"version": "1.0"}

Final hash matches B: true
All files accessible (old + new): true
```
