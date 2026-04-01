# Project Guidelines

## Code Style
- Keep changes small and localized; avoid broad refactors unless requested.
- Follow existing Go style in the touched package; prefer clear names over abbreviations.
- For Gin handlers, use the wrapper helpers from h/wrappers.go:
  - h.G for basic handlers
  - h.B for JSON body binding
  - h.Q for query binding
- Preserve the API response contract used by wrappers:
  - success: {"data": ...}
  - error: {"error": ...}

## Architecture
- Entry and wiring are in main.go.
- Core boundaries:
  - routes/: route registration and HTTP handlers
  - mid/: auth and authorization middleware
  - h/: handler wrappers and HTTP error mapping
  - store/: DB init + GORM models
  - tasks/: task definitions, execution lifecycle, and SSE integration
  - sse/: event broker for task output streaming
  - aliyun/: Alibaba Cloud client integrations
  - casbin/: RBAC model and policy
- Initialization order is intentional: load configs first, then initialize modules.

## Build and Test
- Generate config:
  - make gen-config
- Run unit tests:
  - make test
  - equivalent: go test -v ./...
- Build binary:
  - make build
- Run in dev mode:
  - make dev
- API integration tests (requires running server and hurl installed):
  - cd hurl/users && make test-login USER=<name>
  - cd hurl/users && make test-create USER=<name>
  - cd hurl/tasks && make test-create-test-task USER=<name>

## Conventions
- Config files live under configs/*.toml and are validated during startup.
- If you change config structs/tags/defaults, update config.example.toml and keep cmd/configgen output aligned.
- Authenticated routes depend on session middleware and user context set by mid/auth.go.
- Task execution updates status and streams output via SSE broker; keep lifecycle transitions consistent.
- Prefer existing patterns from:
  - routes/user_routes/user_routes.go
  - routes/user_routes/user_routes_test.go
  - tasks/executor.go

## Pitfalls
- Session key material must be present in config; otherwise startup fails.
- In dev mode, auto-migration can modify schema; be cautious in shared DBs.
- hurl scripts assume env values from hurl/env/dev.env and a reachable local server.
- Some tests rely on real store initialization; avoid introducing global state coupling where possible.
