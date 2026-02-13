# portblock — project reference

## what it is
CLI tool that mocks APIs from OpenAPI specs. give it a spec file, get a working mock server with fake data, stateful CRUD, auth simulation, and more.

**current version:** 0.3.0

## architecture
two-file Go project:
- **main.go** — everything: cobra commands, HTTP server, Store, MockServer, fake data generation, proxy/replay/diff logic
- **ui.go** — charmbracelet/lipgloss styling: banners, route tables, request logging, colored output

### key abstractions
- **Store** — thread-safe in-memory CRUD (`map[string]map[string]interface{}`) with per-resource write tracking
- **MockServer** — holds the parsed OpenAPI doc, Store, router, and handles all request routing/response generation. uses a mutex for hot reload safety
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
├── main.go          # core server, all commands, store, fake data
├── ui.go            # lipgloss styles, banners, logging
├── go.mod / go.sum  # dependencies
├── CHANGELOG.md     # release notes
├── README.md        # user-facing docs
├── docs/            # vitepress documentation site
├── examples/        # example spec files
└── index.ts         # npm wrapper (not the main project)
```
