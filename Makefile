## help: Show available commands
help:
	@echo ""
	@echo "Usage: make [target]"
	@echo ""
	@echo "Available targets:"
	@echo "  config  - Copy environment config from .env.example" 
	@echo "  up      - Build and run services with docker" 
	@echo "  down    - Shut down services in docker"
	@echo "  test    - Run tests with race detection" 
	@echo ""

config:
	cp .env.example .env

test:
	go test ./... -race

up:
	docker compose up --build

down:
	docker compose down