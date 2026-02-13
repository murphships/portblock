---
layout: home

hero:
  name: portblock
  text: mock APIs that actually behave like real ones
  tagline: One binary. One command. Stateful CRUD, smart fake data, contract testing, and more.
  actions:
    - theme: brand
      text: Get Started
      link: /getting-started
    - theme: alt
      text: View on GitHub
      link: https://github.com/murphships/portblock

features:
  - icon: 🧠
    title: Smart Fake Data
    details: Reads property names AND schema types. Knows "email" = email, "city" = city. 60+ field patterns. No examples needed in your spec.
  - icon: 💾
    title: Stateful CRUD
    details: POST creates. GET returns what you created. DELETE removes it. Your mock behaves like a real API, not a static stub.
  - icon: ✅
    title: Contract Testing
    details: Write tests in YAML, run them against your mock or a live API. Variable interpolation, regex matching, CI-ready exit codes.
  - icon: 🔄
    title: Hot Reload
    details: Edit your spec, save it, and the mock updates instantly. No restart needed. Your stored data survives reloads.
  - icon: 🔌
    title: Proxy & Diff
    details: Forward to a real API and validate both sides. Diff a live API against your spec. Record responses and replay them offline.
  - icon: ⚡
    title: Single Binary
    details: Written in Go. No Node.js, no JVM, no runtime. Download, run, done. Available via Homebrew, apt, scoop, Docker.
---

## Quick Start

```bash
# install
brew install murphships/tap/portblock

# mock an API
portblock serve api.yaml

# test contracts
portblock test api.yaml tests.yaml

# test against a live API
portblock test api.yaml tests.yaml --target https://api.staging.com
```

## Why portblock?

**prism** returns static responses. **wiremock** needs XML config files. **mockoon** needs a GUI.

portblock gives you a **working API** from an OpenAPI spec — with state, smart data, and contract testing — in **one command**.

[Get started →](/getting-started)
