# CAFS Examples

## [01-quickstart](./01-quickstart)
Write, read, snapshot. 5 lines of code.

```bash
cd 01-quickstart && go run .
```

## [02-distributed](./02-distributed)
Multi-machine collaboration. Machine A creates, B modifies, C pulls read-only with eager prefetch. Shows auto-pull, content mixing, and prefetching.

```bash
cd 02-distributed && go run .
```

## [03-nested-dirs](./03-nested-dirs)
Nested directories made simple. Create deep paths, list folders, walk trees. Demonstrates eager metadata loading.

```bash
cd 03-nested-dirs && go run .
```
