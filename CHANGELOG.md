# changelog

all notable changes to portblock. format loosely based on [keep a changelog](https://keepachangelog.com/).

## [0.3.0] — 2026-02-13

the "it watches you back" release. portblock now reloads when you edit your spec and can diff against live APIs.

### added
- **hot reload** — spec file changes are picked up automatically while serving. edit your spec, portblock reloads the router instantly. store data stays intact. disable with `--watch=false`
- **diff command** — `portblock diff spec.yaml --target https://api.example.com` compares a live API against your spec. checks status codes, response shapes, missing/extra fields, type mismatches. exit code 1 if differences found (CI-friendly). supports `--header` for auth forwarding
- **PROJECT.md** — proper project documentation for future sessions

### changed
- MockServer now uses a read-write mutex for thread-safe hot reload
- extracted route printing into a reusable function

## [0.2.0] — 2026-02-12

the "ok now it's actually useful" release. portblock went from "cool demo" to "you could actually use this at work" territory.

### added
- **request validation** — portblock now validates incoming requests against your spec. bad request? you get a helpful 400 with actual details, not some cryptic error
- **prefer header** — force any response code with `Prefer: code=404`. test your error handling without breaking things
- **query parameters** — pagination (`limit`/`offset`) and filtering actually work now
- **auth simulation** — if your spec has security schemes, portblock enforces them. no token = 401. `--no-auth` to skip
- **proxy mode** — forward to a real API and validate both sides against your spec
- **record/replay** — record real API responses and replay them offline. perfect for flaky CI
- **content negotiation** — respects Accept headers like a well-behaved API should
- **cli glow-up** — switched to charmbracelet libs (lipgloss, bubbles). the terminal output is gorgeous now
- **vitepress docs** — proper documentation site. we're professional now (kinda)

### changed
- cli output is way prettier. colored status codes, formatted tables, the works
- better error messages across the board

## [0.1.0] — 2026-01-28

the "it works on my machine" release. first public version of portblock.

### added
- **basic mock server** — give it an OpenAPI spec, get a working API at localhost:4000
- **smart fake data** — reads schema types AND property names. knows `email` = email, `city` = city. 60+ field patterns built in. no `examples` needed in your spec
- **stateful CRUD** — POST creates, GET returns, DELETE removes. your mock actually remembers things
- **chaos mode** — `--chaos` flag for random 500s and latency spikes. test your app's resilience
- **seed support** — `--seed 42` for reproducible fake data
- **delay simulation** — `--delay 200ms` to simulate network latency
- **custom port** — `--port 8080` because 4000 isn't always available

[0.3.0]: https://github.com/murphships/portblock/compare/v0.2.0...v0.3.0
[0.2.0]: https://github.com/murphships/portblock/compare/v0.1.0...v0.2.0
[0.1.0]: https://github.com/murphships/portblock/releases/tag/v0.1.0
