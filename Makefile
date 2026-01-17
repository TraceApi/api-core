APP_NAME=tracestack
VERSION=$(shell git describe --tags --always --dirty)
BUILD_DATE=$(shell date -u +'%Y-%m-%dT%H:%M:%SZ')

# Docker Configuration
IMAGE_INGEST=ghcr.io/traceapi/api-core-ingest:latest
IMAGE_RESOLVER=ghcr.io/traceapi/api-core-resolver:latest
BUILD_CONTEXT=.

# Go commands
.PHONY: all build run test clean build-ingest build-resolver push push-ingest push-resolver deploy

build: build-ingest build-resolver

build-ingest:
	@echo "Building Ingest API Docker Image..."
	docker build \
		-f deploy/ingest.Dockerfile \
		-t $(IMAGE_INGEST) \
		$(BUILD_CONTEXT)

build-resolver:
	@echo "Building Resolver API Docker Image..."
	docker build \
		-f deploy/resolver.Dockerfile \
		-t $(IMAGE_RESOLVER) \
		$(BUILD_CONTEXT)

push: push-ingest push-resolver

push-ingest:
	docker push $(IMAGE_INGEST)

push-resolver:
	docker push $(IMAGE_RESOLVER)

deploy: build push

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
	docker exec -i trace_db psql -U trace_user -d trace_core < internal/platform/storage/postgres/migrations/000001_init_passports.up.sql

db-reset: ## Reset the database (DROP and Re-init)
	@echo "Resetting database..."
	docker exec -i trace_db psql -U trace_user -d trace_core -c "DROP SCHEMA public CASCADE; CREATE SCHEMA public;"
	$(MAKE) db-init

db-migrate-up: ## Run database migrations UP
	@echo "Running migrations UP..."
	migrate -path internal/platform/storage/postgres/migrations -database "postgres://trace_user:trace_password@localhost:5432/trace_core?sslmode=disable" up

db-migrate-down: ## Rollback database migrations
	@echo "Running migrations DOWN..."
	migrate -path internal/platform/storage/postgres/migrations -database "postgres://trace_user:trace_password@localhost:5432/trace_core?sslmode=disable" down

gen-jwt: ## Generate a test JWT token
	go run cmd/gen-jwt-token/main.go

gen-key: ## Generate a new API Key (usage: make gen-key tenant=my-tenant)
	go run cmd/gen-api-key/main.go -tenant=$(or $(tenant),manufacturer-001)

# The License Header Enforcer (Crucial for your BSL strategy)
add-license:
	@echo "Adding BSL License headers to new files..."
# To be defined: a script or tool to add license headers