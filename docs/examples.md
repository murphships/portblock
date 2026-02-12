# Examples

sample specs and common patterns to get you started.

## simple todo API

a minimal CRUD API — great for testing portblock's stateful features.

```yaml
openapi: "3.0.0"
info:
  title: Todo API
  version: "1.0"
paths:
  /todos:
    get:
      summary: List all todos
      responses:
        "200":
          description: A list of todos
          content:
            application/json:
              schema:
                type: array
                items:
                  $ref: "#/components/schemas/Todo"
    post:
      summary: Create a todo
      requestBody:
        required: true
        content:
          application/json:
            schema:
              type: object
              required: [title]
              properties:
                title:
                  type: string
                completed:
                  type: boolean
      responses:
        "201":
          description: Created
          content:
            application/json:
              schema:
                $ref: "#/components/schemas/Todo"
  /todos/{id}:
    get:
      summary: Get a todo
      parameters:
        - name: id
          in: path
          required: true
          schema:
            type: string
      responses:
        "200":
          description: A todo
          content:
            application/json:
              schema:
                $ref: "#/components/schemas/Todo"
        "404":
          description: Not found
    delete:
      summary: Delete a todo
      parameters:
        - name: id
          in: path
          required: true
          schema:
            type: string
      responses:
        "204":
          description: Deleted
components:
  schemas:
    Todo:
      type: object
      properties:
        id:
          type: string
          format: uuid
        title:
          type: string
        completed:
          type: boolean
        created_at:
          type: string
          format: date-time
```

try it:

```bash
portblock serve todo-api.yaml

# create a todo
curl -X POST localhost:4000/todos \
  -H "Content-Type: application/json" \
  -d '{"title":"ship portblock docs"}'

# list todos
curl localhost:4000/todos | jq

# delete it
curl -X DELETE localhost:4000/todos/<id>
```

## user API with auth

a more complex spec with security schemes and lots of field types.

```yaml
openapi: "3.0.0"
info:
  title: User API
  version: "1.0"
security:
  - BearerAuth: []
paths:
  /users:
    get:
      summary: List users
      parameters:
        - name: limit
          in: query
          schema:
            type: integer
        - name: offset
          in: query
          schema:
            type: integer
        - name: status
          in: query
          schema:
            type: string
      responses:
        "200":
          description: A list of users
          content:
            application/json:
              schema:
                type: array
                items:
                  $ref: "#/components/schemas/User"
        "401":
          description: Unauthorized
    post:
      summary: Create a user
      requestBody:
        required: true
        content:
          application/json:
            schema:
              type: object
              required: [name, email]
              properties:
                name:
                  type: string
                email:
                  type: string
                  format: email
                company:
                  type: string
                city:
                  type: string
      responses:
        "201":
          description: Created
          content:
            application/json:
              schema:
                $ref: "#/components/schemas/User"
  /users/{id}:
    get:
      summary: Get a user
      parameters:
        - name: id
          in: path
          required: true
          schema:
            type: string
      responses:
        "200":
          description: A user
          content:
            application/json:
              schema:
                $ref: "#/components/schemas/User"
        "404":
          description: Not found
    delete:
      summary: Delete a user
      parameters:
        - name: id
          in: path
          required: true
          schema:
            type: string
      responses:
        "204":
          description: Deleted
components:
  securitySchemes:
    BearerAuth:
      type: http
      scheme: bearer
  schemas:
    User:
      type: object
      properties:
        id:
          type: string
          format: uuid
        name:
          type: string
        email:
          type: string
          format: email
        username:
          type: string
        company:
          type: string
        job_title:
          type: string
        city:
          type: string
        country:
          type: string
        bio:
          type: string
        avatar:
          type: string
          format: uri
        status:
          type: string
          enum: [active, inactive, pending]
        created_at:
          type: string
          format: date-time
```

try it:

```bash
# without auth — gets rejected
portblock serve user-api.yaml
curl localhost:4000/users
# → 401

# with auth
curl -H "Authorization: Bearer test" localhost:4000/users | jq

# or skip auth entirely
portblock serve user-api.yaml --no-auth
```

## common patterns

### testing error handling

```bash
# force specific error codes
curl -H "Prefer: code=404" localhost:4000/users/123
curl -H "Prefer: code=500" localhost:4000/users
curl -H "Prefer: code=429" localhost:4000/users
```

### pagination testing

```bash
curl "localhost:4000/users?limit=5&offset=0"
curl "localhost:4000/users?limit=5&offset=5"
curl "localhost:4000/users?limit=5&offset=10"
```

### chaos testing

```bash
# start with chaos
portblock serve api.yaml --chaos

# hammer it and see what breaks
for i in $(seq 1 100); do
  curl -s -o /dev/null -w "%{http_code}\n" localhost:4000/users
done
# → mix of 200s and 500s
```

### recording and replaying

```bash
# record from production
portblock proxy api.yaml --target https://api.prod.com --record

# replay in CI
portblock replay recordings.json
```

### deterministic test data

```bash
# same seed = same data, every time
portblock serve api.yaml --seed 12345

# great for snapshot tests
curl localhost:4000/users | jq > expected.json
```
