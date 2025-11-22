# Repository Guidelines

## Project Structure & Module Organization
- `cmd/chatbotgate/`: CLI entrypoint for the reverse proxy.
- `pkg/middleware/`: Auth, authz, session, rules, forwarding, and config logic.
- `pkg/proxy/`: Reverse-proxy implementation and config helpers.
- `pkg/shared/`: Logging, i18n, kvs, filewatcher, factory utilities.
- `web/`: Frontend assets (Yarn build); `email/`: email templates; `examples/`: deployment configs.
- Tests live alongside packages, plus `e2e/` (Playwright + Docker) and fixtures in `test/`.

## Build, Test, and Development Commands
- `make build` (or `make build-go` / `make build-web`): compile `bin/chatbotgate` and bundle web assets.
- `./bin/chatbotgate -c config.example.yaml`: run locally with the sample config after building.
- `make test` / `go test ./...`: Go unit/integration suite; `make test-coverage` writes `coverage.out`.
- `make lint`: run `golangci-lint`; `make fmt` or `make fmt-check` enforce `gofmt`.
- `make dev-web` or `cd web && yarn dev`: frontend dev server; `cd e2e && make test` for Playwright e2e (Docker, set `HEADLESS=false` to watch).

## Coding Style & Naming Conventions
- Go code is `gofmt`-first; prefer table-driven tests and wrapped errors for context.
- Keep exported names Go-idiomatic (`NewX`, `Config`, `Handler`); keep config/provider IDs kebab- or snake-case per examples.
- Use structured logging via `pkg/shared/logging`; avoid ad-hoc `fmt.Printf` in production paths.
- Frontend uses Yarn; keep TypeScript/JS linted by the web toolchain when touching `web/`.

## Testing Guidelines
- Name tests `*_test.go` with descriptive functions (`TestComponent_Scenario`); keep assertions minimal and focused.
- For configuration-heavy changes, run `./bin/chatbotgate test-config -c <file>` to validate YAML.
- Capture coverage when altering core middleware/proxy paths (`make test-coverage`) and update fixtures under `test/` or `e2e/testdata` when behavior changes.
- E2E: `cd e2e && make dev` to spin up the stack for manual checks, `make test` for Playwright runs; clean with `make test-down` or `make clean-e2e`.

## Commit & Pull Request Guidelines
- Use short, imperative commit subjects; keep one logical change per commit and add body notes for behavior or config shifts.
- PRs should summarize intent, list user-facing changes (UI/auth flows), note config migrations, and link issues. Add screenshots/GIFs for UI or email template adjustments.
- Mention test coverage run (e.g., `make test`, `cd e2e && make test HEADLESS=true`) in the PR description.

## Security & Configuration Tips
- Never commit secrets; rely on `${VAR}` or `${VAR:-default}` expansion in configs. Prefer env vars for client secrets and cookie keys.
- Validate new configs against `config.example.yaml`/`config.example.json` patterns and document defaults in the PR when introducing new fields.
