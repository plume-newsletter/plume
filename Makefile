.PHONY: dev build build-web test migrate sqlc generate

build-web:
	cd web && npm install && npm run build

build: build-web
	go build -o plume ./cmd/plume

test:
	go test ./...

dev:
	@echo "Run in two terminals: 'go run ./cmd/plume' and 'cd web && npm run dev'"

sqlc:
	sqlc generate

generate:
	sqlc generate

migrate:
	go run github.com/pressly/goose/v3/cmd/goose -dir internal/store/migrations postgres "$$PLUME_DATABASE_URL" up
