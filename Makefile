APP_NAME=tracestack
VERSION=$(shell git describe --tags --always --dirty)
BUILD_DATE=$(shell date -u +'%Y-%m-%dT%H:%M:%SZ')

# Go commands
.PHONY: all build run test clean

up: ## Start the dev infrastructure (DB, Redis, S3)
	docker-compose up -d

down: ## Stop the dev infrastructure
	docker-compose down

run-ingest: ## Run the Ingest API locally
	go run cmd/api-ingest/main.go

run-resolver: ## Run the Resolver API locally
	go run cmd/api-resolver/main.go

db-init: ## Initialize the database schema (Manual migration for dev)
	@echo "Initializing database..."
	docker exec -i trace_db psql -U trace_user -d trace_core < internal/platform/storage/postgres/migrations/001_init_passports.sql

db-migrate: ## Run database migrations (We will add the tool later)
	@echo "Running migrations..."
	# migrate -path internal/platform/db/migrations -database "postgresql://trace_user:trace_password@localhost:5432/trace_core?sslmode=disable" up

# The License Header Enforcer (Crucial for your BSL strategy)
add-license:
	@echo "Adding BSL License headers to new files..."
# To be defined: a script or tool to add license headers