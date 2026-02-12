# Query Parameters

portblock handles pagination and filtering out of the box. no config needed.

## pagination

```bash
# get the first 10
curl "localhost:4000/users?limit=10"

# skip the first 5, get the next 10
curl "localhost:4000/users?limit=10&offset=5"
```

works with both generated default data and your manually created (POST'd) resources.

## filtering

filter by any field on the resource:

```bash
# only active users
curl "localhost:4000/users?status=active"

# users in Boston
curl "localhost:4000/users?city=Boston"

# combine filters
curl "localhost:4000/users?status=active&city=Boston"
```

portblock does case-sensitive string matching against stored resources.

## how it works

- `limit` — max number of items to return (default varies by dataset size)
- `offset` — number of items to skip from the start
- any other query param is treated as a filter on the response data

this means you can paginate through generated data and filter your POSTed resources — all with zero configuration.

## why this matters

most mock servers ignore query parameters entirely. you send `?limit=10` and get... the same full response. portblock actually respects pagination and filtering, so your frontend code works correctly against the mock.
