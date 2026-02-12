# ⬛ portblock

mock APIs that actually behave like real ones.

you know prism? cool tool. but your mock forgets everything the moment you POST something. portblock doesn't. it's a mock server with memory.

give it an OpenAPI spec, get a server that does real CRUD — POST creates a resource, GET retrieves it, PUT updates it, DELETE removes it. like a real API. because your frontend shouldn't have to pretend.

## install

```bash
go install github.com/murphships/portblock@latest
```

or clone and build:

```bash
git clone https://github.com/murphships/portblock.git
cd portblock
go build -o portblock .
```

## usage

```bash
portblock serve api.yaml
```

that's it. zero config. it reads your spec, spins up a server on `:4000`, and starts serving realistic fake data.

```
  ⬛ portblock v0.1.0
  spec:  api.yaml
  port:  4000
  seed:  42

  ready at http://localhost:4000

  GET,POST /todos
  GET,PUT,DELETE /todos/{id}
```

## stateful CRUD

this is the whole point. your mock actually remembers things.

```bash
# create a todo
curl -X POST http://localhost:4000/todos \
  -H 'Content-Type: application/json' \
  -d '{"title":"ship portblock","description":"make it work"}'

# → {"id":"3b1351bf-...","title":"ship portblock","description":"make it work"}

# get it back
curl http://localhost:4000/todos/3b1351bf-...

# → same object. because it remembers.

# update it
curl -X PUT http://localhost:4000/todos/3b1351bf-... \
  -H 'Content-Type: application/json' \
  -d '{"completed":true}'

# delete it
curl -X DELETE http://localhost:4000/todos/3b1351bf-...
# → 204. gone.
```

## smart fake data

portblock reads your schema types and generates realistic data. no `examples` needed.

- `string` with `format: email` → `angelobarnes@swift.org`
- `string` with `format: date-time` → `2024-03-15T10:30:00Z`
- `string` with `format: uuid` → proper UUID
- `integer` → sensible random number (respects min/max)
- `boolean` → random true/false
- `$ref` → resolves and generates
- `array` → 2-5 items
- `enum` → picks from your values

## flags

```bash
portblock serve api.yaml --port 8080        # custom port
portblock serve api.yaml --seed 42          # reproducible data
portblock serve api.yaml --delay 200ms      # simulate latency
portblock serve api.yaml --chaos            # random 500s and latency spikes 💥
```

| flag | default | what it does |
|------|---------|-------------|
| `--port, -p` | 4000 | port to listen on |
| `--seed` | random | seed for reproducible fake data |
| `--delay` | 0 | simulated latency per request |
| `--chaos` | false | 10% chance of 500, random latency spikes |

## vs prism

| feature | prism | portblock |
|---------|-------|-----------|
| reads OpenAPI spec | ✅ | ✅ |
| generates fake data | ✅ | ✅ |
| stateful CRUD | ❌ | ✅ |
| consistent responses | ❌ | ✅ (seeded) |
| chaos mode | ❌ | ✅ |
| single binary | ❌ (node) | ✅ (go) |
| zero config | ✅ | ✅ |

prism is great for "does my spec look right?" portblock is great for "i need to actually build a frontend against this."

## how it works

1. reads your OpenAPI 3.x spec with [kin-openapi](https://github.com/getkin/kin-openapi)
2. registers routes for every path in your spec
3. POST/PUT → stores in memory, GET → retrieves, DELETE → removes
4. if no stored data exists for a GET, generates fake data from your schema
5. fake data is seeded per-path so the same GET returns the same data
6. CORS is enabled by default because we're not animals

## license

MIT

---

built by [murph](https://twitter.com/murphships) at mass o'clock in the morning. ship > perfect.
