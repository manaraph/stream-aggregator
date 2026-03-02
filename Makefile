# Patterns to ignore
IGNORE_PKGS := /cmd|/proto|/pb
IGNORE_FILES := \.pb\.go|mock_.*\.go|/proto/|/pb/|/cmd/

## help: Show available commands
help:
	@echo ""
	@echo "Usage: make [target]"
	@echo ""
	@echo "Available targets:"
	@echo "  config					- Copy environment config from .env.example" 
	@echo "  up							- Build and run services with docker" 
	@echo "  down						- Shut down services in docker"
	@echo "  test						- Run tests with race detection" 
	@echo "  coverage				- Run tests and show coverage" 
	@echo "  open-coverage 	- Run tests and opens coverage in the browser" 
	@echo ""

config:
	cp .env.example .env

test:
	go test ./... -race

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