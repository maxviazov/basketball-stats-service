# Architecture

This service follows a clean architecture layout with clear separation of concerns:

- HTTP handlers (Gin): routing, DTO binding, basic syntax validation and error mapping using pkg/response.
- Services: business rules, input normalization, domain validation, existence checks, and orchestrating repositories.
- Repositories: persistence access (pgxpool), SQL <-> domain mapping, and mapping PG errors to domain errors.
- Postgres: durable store with migrations (goose), indexes/constraints for integrity and performance.

## Data flow

Write (Upsert stat line):
1. Handler receives JSON (machine-to-machine input), binds DTO.
2. Service validates: IDs > 0, integers >= 0, Fouls ∈ [0..6], Minutes Played ∈ [0..48.0]; normalizes.
3. Service checks existence (player and game). No pre-check uniqueness—leave it for DB.
4. Repository upserts using ON CONFLICT, returning the current row state.
5. Handler returns 200 with the updated line.

Read (Aggregates):
1. Handler parses query (season or career=true) and enforces mutual exclusivity.
2. Service validates IDs and season format, normalizes.
3. Repository runs aggregate queries (GROUP BY / SUM / AVG etc.), returns typed results.
4. Handler returns 200 with JSON.

## Error model
- Service returns domain errors:
  - ErrInvalidInput (with field_errors)
  - ErrNotFound, ErrAlreadyExists, ErrConflict
- Handlers map them to HTTP via pkg/response.

## Health & readiness
- /live returns process liveness.
- /ready pings DB via a minimal repository interface (Pinger) to validate critical dependencies.

## Testing strategy
- Unit tests: service validation and business rules, using fakes.
- Contract tests: repository interfaces against Postgres (migrations applied beforehand), verifying behavior and error mapping.
- Handler tests: httptest for endpoints, error mapping, and corner cases.

## Scalability considerations
- Stateless app containers; horizontal scale behind a load balancer.
- Postgres: connection pool sizing (min/max), health checks, and timeouts tuned in config.
- Write-read path is designed to surface new data immediately after write (upsert), ensuring aggregates reflect latest persisted stats.

## Extensibility
- Repositories are interfaces; adding a cache layer or an alternative store is possible by implementing the same contracts.
- Validation lives in services, enabling new rules without touching I/O layers.
