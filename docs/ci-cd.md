# CI/CD Integration

## GitHub Action

portblock provides a GitHub Action for testing API contracts in your CI pipeline.

### Basic Usage

```yaml
name: API Tests
on: [push, pull_request]

jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: murphships/portblock/.github/actions/portblock@main
        with:
          spec: api.yaml
          tests: tests.yaml
```

This spins up a mock server from your spec, runs the tests, and fails the build if any test fails.

### Test Against a Live API

```yaml
      - uses: murphships/portblock/.github/actions/portblock@main
        with:
          spec: api.yaml
          tests: tests.yaml
          command: test
          target: https://staging-api.example.com
```

### Diff Against Production

```yaml
      - uses: murphships/portblock/.github/actions/portblock@main
        with:
          spec: api.yaml
          command: diff
          target: https://api.production.com
```

### Action Inputs

| Input | Required | Default | Description |
|-------|----------|---------|-------------|
| `spec` | ✅ | — | Path to OpenAPI spec file |
| `tests` | — | — | Path to test file |
| `command` | — | `test` | Command to run (`test` or `diff`) |
| `target` | — | — | Target URL (default: internal mock) |
| `version` | — | `latest` | portblock version to install |

## Generic CI

portblock is a single binary. Install it in any CI:

```bash
curl -sL https://github.com/murphships/portblock/releases/latest/download/portblock_linux_amd64.tar.gz | tar xz
sudo mv portblock /usr/local/bin/
portblock test api.yaml tests.yaml
```

### GitLab CI

```yaml
api-test:
  image: golang:1.22
  script:
    - curl -sL https://github.com/murphships/portblock/releases/latest/download/portblock_linux_amd64.tar.gz | tar xz
    - chmod +x portblock
    - ./portblock test api.yaml tests.yaml
```

### Exit Codes

- `0` — all tests pass / no diffs found
- `1` — test failures / diffs found

Use this for build gates, PR checks, and deployment pipelines.
