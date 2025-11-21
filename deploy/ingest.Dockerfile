# --- Stage 1: Builder ---
FROM golang:1.25-alpine AS builder

WORKDIR /app

# Cache dependencies first (Optimization)
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the binary
# -o /ingest-api : Output filename
# ./cmd/api-ingest : Input entry point
# CGO_ENABLED=0 : Static binary (no external C dependencies)
RUN CGO_ENABLED=0 GOOS=linux go build -o /ingest-api ./cmd/api-ingest

# --- Stage 2: Runtime ---
FROM alpine:3.19

WORKDIR /root/

# Add certificates so we can make HTTPS calls (Validation/S3)
RUN apk --no-cache add ca-certificates

# Copy the binary from the builder stage
COPY --from=builder /ingest-api .

# Copy schemas? NO. 
# They are embedded in the binary via go:embed, so we don't need to copy files!

# Expose the port
EXPOSE 8080

# Run
CMD ["./ingest-api"]