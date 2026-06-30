.PHONY: dev build build-web test fmt fmt-check migrate sqlc generate

build-web:
	cd web && npm install && npm run build

build: build-web
	go build -o plume ./cmd/plume

# fmt rewrites any unformatted Go files in place.
fmt:
	gofmt -w .

# fmt-check fails (like CI) if any Go file is not gofmt-clean.
fmt-check:
	@test -z "$$(gofmt -l .)" || (gofmt -l . && echo 'Run: make fmt' && exit 1)

test: fmt-check
	go test ./...

dev:
	@echo "Run in two terminals: 'go run ./cmd/plume' and 'cd web && npm run dev'"

sqlc:
	sqlc generate

generate:
	sqlc generate

migrate:
	go run github.com/pressly/goose/v3/cmd/goose -dir internal/store/migrations postgres "$$PLUME_DATABASE_URL" up
