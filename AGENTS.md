# portblock — project reference

## what it is
CLI tool that mocks APIs from OpenAPI specs. give it a spec file, get a working mock server with fake data, stateful CRUD, auth simulation, and more.

**current version:** 0.4.0

## architecture
multi-file Go project:
- **main.go** — core: cobra commands, HTTP server, Store, MockServer, fake data generation, proxy/replay/diff logic
- **ui.go** — charmbracelet/lipgloss styling: banners, route tables, request logging, colored output
- **config.go** — config file loading (.portblock.yaml/.json), `init` command
- **strict.go** — strict mode: spec validation, response schema validation, aggressive request checking
- **webhooks.go** — webhook/callback dispatching with retry logic
- **generate.go** — reverse-engineer OpenAPI specs from live APIs

### key abstractions
- **Store** — thread-safe in-memory CRUD (`map[string]map[string]interface{}`) with per-resource write tracking
- **MockServer** — holds the parsed OpenAPI doc, Store, router, webhook manager, and handles all request routing/response generation. uses a mutex for hot reload safety
- **WebhookManager** — fires webhooks on mutations with retry logic and exponential backoff
- **fake data generation** — `gofakeit` + property name heuristics (60+ patterns like `email`→email, `city`→city). seeded RNG for reproducibility
- **content negotiation** — JSON and XML response support via Accept header
- **auth simulation** — enforces security schemes from the spec (bearer, apiKey, oauth2)

## commands
| command | status | description |
|---------|--------|-------------|
| `serve` | ✅ implemented | mock server from spec, with hot reload (`--watch`) |
| `proxy` | ✅ implemented | reverse proxy with spec validation |
| `replay` | ✅ implemented | replay recorded responses |
| `diff` | ✅ implemented | compare live API against spec |
| `init` | ✅ implemented | scaffold a `.portblock.yaml` config file |
| `generate` | ✅ implemented | reverse-engineer OpenAPI spec from live API |

## features
| feature | status |
|---------|--------|
| fake data generation (schema + name heuristics) | ✅ |
| stateful CRUD | ✅ |
| request validation | ✅ |
| Prefer header (force status codes) | ✅ |
| query params (filter/paginate) | ✅ |
| auth simulation | ✅ |
| chaos mode | ✅ |
| seed/delay flags | ✅ |
| content negotiation (JSON/XML) | ✅ |
| proxy mode with validation | ✅ |
| record/replay | ✅ |
| hot reload (fsnotify) | ✅ |
| diff against live API | ✅ |
| vitepress docs site | ✅ (in docs/) |
| config file (.portblock.yaml) | ✅ |
| strict mode (--strict) | ✅ |
| webhooks/callbacks | ✅ |
| spec generation (reverse-engineer) | ✅ |
| Dockerfile + docker-compose | ✅ |

## dependencies
- `github.com/spf13/cobra` — CLI framework
- `github.com/getkin/kin-openapi` — OpenAPI parsing, validation, routing
- `github.com/brianvoe/gofakeit/v7` — fake data
- `github.com/charmbracelet/lipgloss` — terminal styling
- `github.com/fsnotify/fsnotify` — file watching for hot reload

## build
```bash
go build -o portblock .
```

## file structure
```
├── main.go              # core server, commands, store, fake data
├── ui.go                # lipgloss styles, banners, logging
├── config.go            # config file loading, init command
├── strict.go            # strict mode validation
├── webhooks.go          # webhook/callback dispatching
├── generate.go          # spec generation from live APIs
├── go.mod / go.sum      # dependencies
├── Dockerfile           # multi-stage docker build
├── docker-compose.yml   # docker compose example
├── CHANGELOG.md         # release notes
├── README.md            # user-facing docs
├── docs/                # vitepress documentation site
├── examples/            # example spec files
└── index.ts             # npm wrapper (not the main project)
```
