# VNTU Timetable

## Local setup

```sh
cp .env.example .env
```

Export variables into current shell:

```sh
set -a
. ./.env
set +a
```

Create or update the database:

```sh
go run . migrate
```

## Development

Go 1.26.6.

Checks:

- `gofmt -w .`
- `go vet ./...`
- `go test ./...`
- `go test -race ./...`

Build:

- `go build -o dist/vntu-timetable .`

Enable pre-commit hook:

- `git config core.hooksPath .githooks`
