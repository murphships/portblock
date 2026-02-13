# Test Runner

`portblock test` validates API contracts with sequential, stateful tests. Write tests in YAML, run them against your mock or a live API.

## Quick Start

```bash
# test against the built-in mock server
portblock test api.yaml tests.yaml

# test against a live API
portblock test api.yaml tests.yaml --target https://api.example.com
```

## Test File Format

```yaml
tests:
  - name: "create user"
    method: POST
    path: /users
    headers:
      Content-Type: application/json
    body:
      name: "test user"
      email: "test@example.com"
    expect:
      status: 201
      body:
        - field: "id"
          exists: true
        - field: "name"
          equals: "test user"
        - field: "email"
          matches: ".*@.*"
    save:
      user_id: "id"

  - name: "get created user"
    method: GET
    path: "/users/{{user_id}}"
    expect:
      status: 200
      body:
        - field: "name"
          equals: "test user"
```

## Features

### Variable Interpolation

Save response fields and use them in later tests:

```yaml
save:
  user_id: "id"
  token: "data.auth.token"
```

Use with `{{variable_name}}` in paths, headers, and body values.

### JSON Path

Access nested fields with dot notation:

```yaml
body:
  - field: "data.user.id"
    exists: true
  - field: "meta.pagination.total"
    type: number
```

### Assertions

| Assertion | Description | Example |
|-----------|-------------|---------|
| `status` | HTTP status code | `status: 201` |
| `exists` | Field exists (or not) | `exists: true` |
| `equals` | Exact value match | `equals: "test user"` |
| `matches` | Regex match | `matches: ".*@.*"` |
| `type` | Type check | `type: string` (string/number/boolean/array/object) |
| `is_array` | Response is an array | `is_array: true` |
| `min_length` | Minimum array length | `min_length: 1` |
| `max_length` | Maximum array length | `max_length: 100` |

### Verbose Mode

See full request/response details:

```bash
portblock test api.yaml tests.yaml --verbose
```

### Exit Codes

- `0` — all tests passed
- `1` — one or more tests failed

Perfect for CI pipelines.

## Internal Mock Server

When no `--target` is specified, portblock spins up a temporary mock server from your spec, runs the tests against it, then shuts it down. This validates that your spec supports the workflows your tests describe.

## Example

See `examples/todo-tests.yaml` for a complete example testing the todo API.
