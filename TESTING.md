# Testing Guide

Follow these steps to validate the TraceApi Core implementation.

## 1. Automated Tests (Unit & Integration)

Before running the application manually, you should run the automated test suite.

### Run All Tests
```bash
go test ./...
```

### Run Integration Tests
Integration tests require the Docker infrastructure to be running (`make up`).
```bash
go test -tags=integration ./tests/integration/... -v
```

### Run Specific Layers
**Service Layer (Business Logic):**
```bash
go test ./internal/core/service/... -v
```

**Handler Layer (HTTP Logic):**
```bash
go test ./internal/transport/rest/... -v
```

## 2. Manual Testing (Local Dev)

### Authentication (New!)

The Ingest API is now protected by JWT Authentication. You must generate a token to make requests.

**Generate a Test Token:**
```bash
go run scripts/generate_token.go
```
This will output a valid `Bearer` token and a ready-to-use `curl` command.

### Start Infrastructure

Start the PostgreSQL and Redis containers:

```bash
make up
```

Wait a few seconds for the database to be ready.

### Initialize Database

Apply the initial schema to the database using the migration tool:

```bash
make db-migrate-up
```

### Run the Applications

You will need two terminal windows.

**Terminal 1 (Ingest API):**
```bash
make run-ingest
```
*Expected Output:* `{"time":"...","level":"INFO","msg":"TraceApi Ingest Server starting","port":"8080"}`

**Terminal 2 (Resolver API):**
```bash
make run-resolver
```
*Expected Output:* `{"time":"...","level":"INFO","msg":"TraceApi Resolver Server starting","port":"8081"}`

## 3. Test Scenarios

### Scenario A: Create a Battery Passport (Success)

**Prerequisite:** Generate a token first.
```bash
TOKEN=$(go run scripts/generate_token.go | grep "eyJ" | head -n 1)
```

**Request:**
```bash
curl -X POST "http://localhost:8080/v1/passports?category=BATTERY_INDUSTRIAL" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "batteryModel": "Model Y Structural Pack",
    "chemistry": "LITHIUM_ION",
    "ratedCapacity": 75.0,
    "carbonFootprint": {
      "totalCarbonFootprint": 45.2,
      "shareOfRenewables": 80
    },
    "materialComposition": [
      { "material": "Lithium", "massPercentage": 2.5, "recycledContentPercentage": 0 }
    ]
  }'
```

**Expected Response (201 Created):**
```json
{
  "id": "some-uuid-...",
  "product_category": "BATTERY_INDUSTRIAL",
  "status": "DRAFT",
  ...
}
```

### Scenario B: Invalid Data (Error Handling)

**Request:**
```bash
curl -X POST "http://localhost:8080/v1/passports?category=BATTERY_INDUSTRIAL" \
  -H "Content-Type: application/json" \
  -d '{
    "batteryModel": "Bad Battery",
    "chemistry": "UNKNOWN_CHEMISTRY" 
  }'
```

**Expected Response (400 Bad Request):**
```text
invalid input: schema validation failed
```
*Check Terminal 1 logs to see the detailed validation error.*

### Scenario C: Resolve Passport (JSON)

Copy the `id` from Scenario A.

**Request:**
```bash
# Replace <UUID> with the ID from Scenario A
curl -v "http://localhost:8081/r/<UUID>"
```

**Expected Response (200 OK):**
Returns the full passport JSON.

### Scenario D: Resolve Passport (HTML)

**Request:**
```bash
# Replace <UUID> with the ID from Scenario A
curl -v -H "Accept: text/html" "http://localhost:8081/r/<UUID>"
```

**Expected Response (200 OK):**
Returns an HTML page displaying the passport data.

### Scenario E: Get QR Code

**Request:**
```bash
# Replace <UUID> with the ID from Scenario A
curl -v "http://localhost:8081/r/<UUID>/qr" --output passport_qr.png
```

**Expected Response (200 OK):**
Downloads a `passport_qr.png` file. Open it to verify it scans to `http://localhost:8081/r/<UUID>`.
