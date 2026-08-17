# VNTU Timetable

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


