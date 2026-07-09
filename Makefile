FRONTEND := npm --prefix frontend
GO_TEST := go test

.PHONY: dev test test-integration e2e examples frontend-test frontend-build migrate

dev:
	mprocs

migrate:
	go run ./cmd/migrate up

test:
	$(GO_TEST) ./...
	$(FRONTEND) run typecheck
	$(FRONTEND) test

test-integration:
	$(GO_TEST) -tags=integration ./...

e2e:
	./e2e/full-flow.sh

examples:
	go build ./examples/...

frontend-test:
	$(FRONTEND) test

frontend-build:
	$(FRONTEND) run build
