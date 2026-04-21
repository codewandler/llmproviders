# Testing Strategy

## Goals

- `go test ./...` should be deterministic and safe to run without live credentials.
- Unit tests are the default safety net.
- Live provider tests remain available as opt-in smoke coverage.

## Test classes

### Unit tests

Unit tests must:
- avoid live network calls
- avoid real credentials
- be deterministic
- run by default with `go test ./...`

### Integration / smoke tests

Integration tests may:
- call live provider APIs
- require local credentials or local model runtimes
- fail for environmental reasons such as quota, rate limits, or outages

These tests are opt-in.

## How to run tests

### Default test suite

```bash
go test ./...
```

### Live provider integration tests

```bash
RUN_INTEGRATION=1 go test ./integration/...
```

### Local provider integration tests

```bash
TEST_INTEGRATION_LOCAL=1 go test ./integration/...
```

### Both live and local integration tests

```bash
RUN_INTEGRATION=1 TEST_INTEGRATION_LOCAL=1 go test ./integration/...
```

## Policy

- New CLI and provider behavior should get unit tests first where practical.
- Live integration tests should focus on smoke coverage of critical paths.
- Expensive defaults such as very large token budgets should be avoided in integration tests.
