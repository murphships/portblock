# Chaos Mode

your API works when everything goes right. but what about when it doesn't?

chaos mode injects random failures into your mock server so you can test your app's resilience.

## usage

```bash
portblock serve api.yaml --chaos
```

that's it. one flag.

## what happens

with chaos mode enabled:

- **~10% of requests return 500** — random internal server errors
- **random latency spikes** — some requests take up to 2 seconds longer than normal
- **the rest work normally** — so your app gets a realistic mix of success and failure

## why chaos?

### error handling

does your frontend show a nice error message when the API returns 500? or does it just... crash? chaos mode will tell you.

### retry logic

if your app retries failed requests, chaos mode tests that. does it back off? does it give up after N attempts? does it retry things it shouldn't?

### loading states

random latency means some requests are fast and some are slow. does your UI handle that gracefully? skeleton loaders? spinners? or does it flash and flicker?

### resilience

in production, APIs fail. networks are unreliable. servers go down. testing against a perfect mock gives you false confidence. chaos mode gives you realistic confidence.

## combining with other flags

```bash
# chaos + slow baseline latency = extra painful
portblock serve api.yaml --chaos --delay 500ms

# chaos + specific port
portblock serve api.yaml --chaos --port 8080

# chaos + no auth (just test resilience, not auth)
portblock serve api.yaml --chaos --no-auth
```

## the philosophy

your app should handle failure gracefully. chaos mode is the easiest way to prove that it does — or find out that it doesn't. better to find out with a mock than in production at 3am.
