# CLI Reference

every command and flag in portblock.

## commands

### `portblock serve`

start a mock API server from an OpenAPI spec.

```bash
portblock serve <spec-file> [flags]
```

**arguments:**
- `<spec-file>` — path to your OpenAPI spec (YAML or JSON)

**flags:**

| flag | description | default |
|------|-------------|---------|
| `--port` | port to listen on | `4000` |
| `--seed` | seed for reproducible fake data | random |
| `--delay` | simulate network latency (e.g. `200ms`, `1s`) | `0` |
| `--chaos` | enable chaos mode (random 500s and latency) | `false` |
| `--no-auth` | disable auth simulation | `false` |

**examples:**

```bash
# basic usage
portblock serve api.yaml

# custom port with latency
portblock serve api.yaml --port 8080 --delay 200ms

# reproducible data
portblock serve api.yaml --seed 42

# chaos mode, no auth
portblock serve api.yaml --chaos --no-auth

# everything at once
portblock serve api.yaml --port 3000 --seed 42 --delay 100ms --chaos --no-auth
```

---

### `portblock proxy`

proxy requests to a real API while validating against your spec.

```bash
portblock proxy <spec-file> --target <url> [flags]
```

**arguments:**
- `<spec-file>` — path to your OpenAPI spec

**flags:**

| flag | description | default |
|------|-------------|---------|
| `--target` | URL of the real API to proxy to | required |
| `--port` | port to listen on | `4000` |
| `--record` | record responses to a file | `false` |

**examples:**

```bash
# proxy and validate
portblock proxy api.yaml --target https://api.example.com

# proxy, validate, and record
portblock proxy api.yaml --target https://api.example.com --record

# custom port
portblock proxy api.yaml --target https://api.example.com --port 8080
```

---

### `portblock replay`

replay previously recorded API responses.

```bash
portblock replay <recordings-file> [flags]
```

**arguments:**
- `<recordings-file>` — path to the recordings JSON file

**flags:**

| flag | description | default |
|------|-------------|---------|
| `--port` | port to listen on | `4000` |

**examples:**

```bash
# replay recorded responses
portblock replay recordings.json

# custom port
portblock replay recordings.json --port 9000
```

## global behavior

- all commands bind to `localhost` by default
- JSON and YAML specs are both supported
- portblock auto-detects OpenAPI 2.0 (Swagger) and 3.x specs
- ctrl+c to stop the server gracefully
