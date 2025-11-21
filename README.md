# TraceApi


#### Project structure

api-core/
├── cmd/                    # Entry points (Main applications)
│   ├── api-ingest/         # The write-heavy API for manufacturers
│   └── api-resolver/       # The read-heavy API for QR codes/Consumers
├── internal/               # Private application code (The Business Logic)
│   ├── core/               # Domain entities (Passport, Event, Scan)
│   ├── platform/           # Infrastructure (Postgres, Redis, S3)
│   └── service/            # Application logic (PassportService, Auth)
├── pkg/                    # Public libraries (Safe to import by others)
│   ├── schema/             # The JSON Schemas we just defined
│   └── validator/          # Shared validation logic
├── deploy/                 # IaC (Docker, Terraform, K8s charts)
├── web/                    # The Next.js Frontend (The Resolver UI)
├── go.mod                  # Go Module definition
├── Makefile                # The Command Center
└── docker-compose.yml      # Local Dev Environment