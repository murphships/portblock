# Stateful CRUD

this is the big one. the thing that makes portblock different from prism and most other mock servers.

POST actually creates something. GET returns what you created. DELETE removes it. your mock behaves like a real API.

## the problem with static mocks

most mock servers return the same canned response every time. you POST a user, then GET the users list, and... it's the same default data. your newly created user is nowhere to be found.

that makes them useless for testing any flow that involves state — which is, you know, most flows.

## how portblock handles it

portblock keeps an in-memory store. when you create something, it's stored. when you read, you get what's stored. when you delete, it's gone.

```bash
# create a user
curl -X POST localhost:4000/users \
  -H "Content-Type: application/json" \
  -d '{"name":"murph","email":"murph@dev.io"}'
# → 201 {"id": "abc-123", "name": "murph", "email": "murph@dev.io"}

# get all users — your created user is there
curl localhost:4000/users
# → [{"id": "abc-123", "name": "murph", "email": "murph@dev.io"}]

# get by id
curl localhost:4000/users/abc-123
# → {"id": "abc-123", "name": "murph", "email": "murph@dev.io"}

# update it
curl -X PUT localhost:4000/users/abc-123 \
  -H "Content-Type: application/json" \
  -d '{"name":"murph v2","email":"murph@dev.io"}'
# → 200

# delete it
curl -X DELETE localhost:4000/users/abc-123
# → 204

# it's gone
curl localhost:4000/users
# → []
```

a real CRUD flow. with a mock server. no config.

## how state works

- **POST** — creates a resource, auto-generates an `id` if not provided, stores it in memory
- **GET** (collection) — returns all stored resources for that path
- **GET** (by id) — returns a specific resource, 404 if not found
- **PUT/PATCH** — updates an existing resource
- **DELETE** — removes it, returns 204

the state lives in memory for the duration of the server process. restart the server and you start fresh.

## default data

when the server starts, portblock generates some initial data for GET endpoints so they're not empty. once you POST something, the collection switches to only returning your explicitly created data.

this means you can:
1. start the server and immediately GET some realistic data
2. POST your own data and work with that instead
3. test the full lifecycle without any setup

## why this matters

- **frontend devs** can build against a mock that actually responds to their actions
- **integration tests** can verify full CRUD flows
- **demos** actually work — create, read, update, delete in real time
- **no more "mock doesn't do that"** conversations with your team
