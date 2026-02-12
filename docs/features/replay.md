# Replay Mode

recorded some real API responses with [proxy mode](/features/proxy-mode)? now you can replay them offline.

## usage

```bash
# first, record responses from a real API
portblock proxy api.yaml --target https://api.production.com --record

# this creates a recordings file
# now replay those exact responses, no internet needed
portblock replay recordings.json
```

your app gets the exact same responses the real API gave, served locally.

## why replay?

### offline testing

CI/CD pipelines shouldn't depend on external APIs. record once, replay forever.

### deterministic tests

real APIs can return different data each time. replay gives you the same response every time. no flaky tests.

### fast tests

no network roundtrip. responses are served from a local file. your test suite runs faster.

### snapshot testing

record a known-good state of the API, then run your tests against it. if your app breaks, you know it's your code — not the API that changed.

## workflow

a typical workflow looks like this:

1. **record** — run your test suite against the real API through portblock proxy with `--record`
2. **commit** — check the recordings file into your repo
3. **replay** — in CI, use `portblock replay recordings.json` instead of hitting the real API
4. **update** — periodically re-record to catch API changes

it's like VCR for your API tests, but without the Ruby.
