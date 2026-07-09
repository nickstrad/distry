FRONTEND := npm --prefix frontend
GO_TEST := go test

.PHONY: dev test test-integration examples frontend-test frontend-build migrate

dev:
	$(FRONTEND) run build
	go run ./cmd/server

migrate:
	go run ./cmd/migrate up

test:
	$(GO_TEST) ./...
	$(FRONTEND) run typecheck
	$(FRONTEND) test

test-integration:
	$(GO_TEST) -tags=integration ./...

examples:
	go build ./examples/...

frontend-test:
	$(FRONTEND) test

frontend-build:
	$(FRONTEND) run build
