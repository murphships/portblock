# Proxy Mode

portblock can sit between your app and a real API, validating both requests and responses against your spec. think of it as a contract-testing proxy.

## basic proxy

```bash
portblock proxy api.yaml --target https://api.production.com
```

every request goes through portblock to the real API. portblock validates:
- your **request** matches the spec (body, params, types)
- the real API's **response** matches the spec

if either side violates the contract, portblock logs a warning. your request still goes through — it doesn't block traffic, just reports violations.

## proxy + record

```bash
portblock proxy api.yaml --target https://api.production.com --record
```

same as above, but portblock also records every response into a file. this gives you a snapshot of real API behavior that you can replay later.

## use cases

### contract testing

deploy the proxy in your staging environment. if your backend starts returning responses that don't match the spec, you'll know immediately.

### migration validation

switching API providers? proxy through portblock to verify the new API matches your expected contract.

### capturing test fixtures

record real API responses, then use [replay mode](/features/replay) for offline testing. no more flaky tests that depend on external services.

### debugging

see exactly what's going over the wire, validated against your spec. better than reading raw HTTP logs.

## how it works

1. your app sends a request to portblock (running as a proxy)
2. portblock validates the request against your spec
3. portblock forwards the request to the target API
4. the real API responds
5. portblock validates the response against your spec
6. portblock returns the response to your app
7. any violations are logged

your app doesn't need to know portblock exists. just point your base URL at it.
