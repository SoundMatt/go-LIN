# Coding Standard — go-LIN

## Language and version

- Go 1.25 or later.
- No CGo. All packages use the Go standard library only (except `transport/` which may use `golang.org/x/sys/unix` for serial I/O on Linux).

## Formatting

- All code must pass `gofmt -w ./...` without changes.
- All code must pass `go vet ./...` with zero warnings.

## Error handling

- Never discard errors silently (`_ = err` is forbidden except in deferred Close calls).
- Return errors; do not panic in library code.
- Wrap errors with `fmt.Errorf("pkg: operation: %w", err)`.

## Safety annotations

- Every exported function implementing a safety requirement must carry a `//fusa:req REQ-*` annotation.
- Every test that verifies a safety requirement must carry a `//fusa:test REQ-*` annotation.
- Annotation format: `//fusa:req REQ-<GROUP>-<NNN>` (no trailing space).

## Concurrency

- All exported types that hold state must be safe for concurrent use unless explicitly documented otherwise.
- Use `sync.Mutex` / `sync.RWMutex` for shared state; prefer channels for signalling.
- Never hold a mutex across a channel send or blocking I/O.

## Comments

- Package doc must state the purpose and list any platform requirements.
- Do not add comments that repeat what the code already says.
- Add a comment only when the WHY is non-obvious.

## Tests

- Every new exported symbol must have at least one test.
- Tests that require real serial hardware must call a helper that skips gracefully when no device is present.
- Fuzz targets must compile and run with `go test -fuzz=<Target> -fuzztime=10s`.

## Dependencies

- The root module and all packages except `transport/` must have zero external dependencies (Go standard library only).
- `transport/` may depend on `golang.org/x/sys/unix` for serial UART access on Linux.
- Bridge packages are welcome; follow the same rules.
