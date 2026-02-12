# Smart Fake Data

most mock servers either return `"string"` for every string field or require you to add `example:` blocks to your spec. portblock does neither.

it reads your property names and schema types, then generates realistic data automatically.

## how it works

portblock matches property names against 60+ built-in patterns. if your field is called `email`, you get an email. if it's called `city`, you get a city. simple as that.

```bash
curl localhost:4000/users | jq '.[0]'
```

```json
{
  "id": "6e759b9a-874f-4a43-aaee-c810d8151d86",
  "name": "Johnathon Braun",
  "username": "theKoch",
  "email": "kaleyboyer@garcia.net",
  "company": "Development Seed",
  "job_title": "Hotel Manager",
  "city": "Boston",
  "country": "Brunei Darussalam",
  "bio": "Publish a changelog entry for the child.",
  "avatar": "https://picsum.photos/seed/5204/640/480",
  "status": "active"
}
```

no `examples` in the spec. just field names and types.

## supported patterns

here's a sample of what portblock recognizes:

| pattern | generates | example |
|---------|-----------|---------|
| `name`, `full_name` | person name | "Johnathon Braun" |
| `first_name` | first name | "Sarah" |
| `last_name` | last name | "Rodriguez" |
| `email` | email address | "kaley@garcia.net" |
| `username` | username | "theKoch" |
| `phone` | phone number | "+1-555-0142" |
| `city` | city name | "Boston" |
| `country` | country name | "Canada" |
| `address` | street address | "742 Evergreen Terrace" |
| `zip`, `postal_code` | postal code | "02134" |
| `company` | company name | "Development Seed" |
| `job_title`, `title` | job title | "Hotel Manager" |
| `url`, `website` | URL | "https://example.com" |
| `avatar`, `image` | image URL | picsum.photos link |
| `bio`, `description` | sentence | realistic text |
| `status` | status string | "active" |
| `created_at`, `updated_at` | ISO timestamp | "2024-01-15T..." |
| `id` | UUID | "6e759b9a-..." |
| `price`, `amount` | decimal number | 42.99 |
| `latitude`, `lat` | latitude | 42.3601 |
| `longitude`, `lng` | longitude | -71.0589 |
| `color` | hex color | "#ff6b35" |
| `ip`, `ip_address` | IP address | "192.168.1.42" |

...and many more. 60+ patterns total.

## type-based fallbacks

if portblock doesn't recognize the field name, it falls back to the schema type:

- `string` → random realistic string
- `integer` → random int in a sensible range
- `number` → random float
- `boolean` → true/false
- `string` with `format: date-time` → ISO timestamp
- `string` with `format: email` → email address
- `string` with `format: uri` → URL
- `string` with `enum` → random value from the enum

## reproducible data

want the same data every time? use the `--seed` flag:

```bash
portblock serve api.yaml --seed 42
```

same seed = same data. great for tests and snapshots.

## arrays

portblock generates sensible array sizes too. a list endpoint typically returns 5-10 items. each item gets unique generated data — no copy-paste responses.
