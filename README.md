# llmproviders

`llmproviders` contains provider-specific implementations, metadata, pricing, and auth/config helpers for use with `agentapis`.

It intentionally does not define provider-agnostic request, message, conversation, or stream abstractions; those belong to `agentapis`.

## Scope

This repository is intended to hold:

- provider adapters/backends
- model qualification and integration test matrixes
- auth and env/config helpers
- provider capabilities
- model catalogs, aliases, and offerings
- pricing metadata and calculators
- integration/live-provider tests

## Non-goals

This repository should not become a second unified abstraction layer.

It should not define:

- canonical request types
- canonical message types
- canonical conversation/session types
- generic stream event models
- generic usage models

Those belong in `agentapis`.

## Intended relationship

- `agentapis`: canonical provider-agnostic abstraction layer
- `llmproviders`: provider-specific implementations, metadata, and qualification matrixes
- `agentcore`: tool definitions and execution/runtime framework
- `miniagent`: app-level consumer of `agentapis` + `llmproviders`
