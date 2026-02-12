# Prefer Header

want to test what happens when your API returns a 404? a 500? a 429? just ask.

portblock supports the `Prefer` header to force any response code defined in your spec.

## usage

```bash
# force a 404
curl -H "Prefer: code=404" localhost:4000/users

# force a 500
curl -H "Prefer: code=500" localhost:4000/users

# force a 429 (rate limited)
curl -H "Prefer: code=429" localhost:4000/users
```

portblock looks up that status code in your spec and returns the matching response schema. if you've defined what a 404 looks like, that's what you'll get.

## why it's useful

- **test error handling** — does your app show the right error message on 404?
- **test loading states** — combine with `--delay` for slow + failing responses
- **test retry logic** — force 503s and see if your retry kicks in
- **frontend dev** — quickly toggle between success and error states without changing your backend

## how it works

1. you send a request with `Prefer: code=XXX`
2. portblock looks up that status code in your spec for that endpoint
3. if found, it generates a response matching that status code's schema
4. if not found (the code isn't defined in your spec), it returns a generic response with that status code

## combining with other features

```bash
# slow 500 — test timeout + error handling together
curl -H "Prefer: code=500" localhost:4000/users
# (with --delay 2s on the server)

# 401 — test auth error flow
curl -H "Prefer: code=401" localhost:4000/users
```

it's like having a remote control for your API's behavior.
