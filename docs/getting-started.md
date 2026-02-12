# Getting Started

## install

### go install (recommended)

```bash
go install github.com/murphships/portblock@latest
```

### build from source

```bash
git clone https://github.com/murphships/portblock.git
cd portblock
go build -o portblock .
```

single binary. no runtime dependencies. no node_modules. just vibes.

## your first mock in 30 seconds

you need an OpenAPI spec. if you don't have one, grab one of the [examples](/examples).

```bash
portblock serve api.yaml
```

that's it. you now have a working API at `localhost:4000`.

```bash
curl localhost:4000/users | jq
```

portblock reads your spec, generates realistic fake data based on field names and types, and serves it up. no `examples` blocks needed in your YAML.

## basic usage

### start a mock server

```bash
portblock serve api.yaml
```

### custom port

```bash
portblock serve api.yaml --port 8080
```

### reproducible data

want the same fake data every time? use a seed:

```bash
portblock serve api.yaml --seed 42
```

### simulate latency

```bash
portblock serve api.yaml --delay 200ms
```

### skip auth checks

if your spec has security schemes but you don't want to deal with tokens right now:

```bash
portblock serve api.yaml --no-auth
```

### go full chaos

```bash
portblock serve api.yaml --chaos
```

10% of requests return 500. random latency spikes. your frontend will hate it (that's the point).

## what happens under the hood

1. portblock parses your OpenAPI spec
2. for each endpoint, it builds a handler that generates realistic responses
3. field names like `email`, `name`, `city` get matched to smart generators (60+ patterns)
4. POST/PUT/PATCH requests store data in memory
5. GET requests return what you've stored (or generated defaults)
6. DELETE actually deletes things
7. all requests are validated against your spec

it's a real API. it just doesn't have a database.

## next steps

- [Smart Fake Data](/features/smart-fake-data) — how field inference works
- [Stateful CRUD](/features/stateful-crud) — POST→GET→DELETE flow
- [CLI Reference](/cli-reference) — all the flags
- [Examples](/examples) — sample specs to try
