# Patterns to ignore
# Match the CI unit coverage gate; storage coverage comes from optional PostgreSQL integration tests.
IGNORE_PKGS := /cmd|/proto|/pb|/internal/storage
IGNORE_FILES := \.pb\.go|mock_.*\.go|/proto/|/pb/|/cmd/

## help: Show available commands
help:
	@echo ""
	@echo "Usage: make [target]"
	@echo ""
	@echo "Available targets:"
	@echo "  generate-mocks\t\t\t- Generate interface mocks with Mockery"
	@echo "  config					- Copy environment config from .env.example" 
	@echo "  up							- Build and run services with docker" 
	@echo "  down						- Shut down services in docker"
	@echo "  test						- Run tests with race detection"
	@echo "  integration-test			- Run PostgreSQL integration tests"
	@echo "  coverage				- Run tests and show coverage"
	@echo "  open-coverage 	- Run tests and opens coverage in the browser" 
	@echo ""

config:
	cp .env.example .env

test:
	go test ./... -race

generate-mocks:
	go run github.com/vektra/mockery/v2@v2.53.5

integration-test:
	test_url="$${TEST_DATABASE_URL:-$$(sed -n 's/^TEST_DATABASE_URL=//p' .env 2>/dev/null | tail -n 1)}"; \
	if [ -z "$$test_url" ]; then echo "Skipping PostgreSQL integration tests: TEST_DATABASE_URL is not set"; exit 0; fi; \
	TEST_DATABASE_URL="$$test_url" go test ./internal/storage -run TestPostgresBatchInsertAndDuplicateDelivery -count=1

up:
	docker compose up --build

down:
	docker compose down

coverage:
	@echo "==> Running tests (excluding $(IGNORE_PKGS))..."
	@go test -v -coverprofile=cover.out.tmp $$(go list ./... | grep -vE '$(IGNORE_PKGS)')
	@grep -vE '$(IGNORE_FILES)' cover.out.tmp > cover.out
	@rm cover.out.tmp
	@go tool cover -func=cover.out

open-coverage: coverage
	go tool cover -html=cover.out
