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

Start the PostgreSQL, Redis, and Minio containers:

```bash
make up
```

Wait for the `trace_init_buckets` container to exit successfully (this ensures the S3 bucket is created).

### Initialize Database

Run the migrations to set up the schema:

```bash
make db-init
```

### Start the APIs

**Terminal 1 (Ingest API):**
```bash
make run-ingest
```

**Terminal 2 (Resolver API):**
```bash
make run-resolver
```

## 3. End-to-End Workflow (Immutability Phase)

### Step 1: Create a Draft Passport
Use the token generated above.

```bash
TOKEN=$(go run scripts/generate_token.go | grep "Bearer" | cut -d ' ' -f 2)

curl -X POST "http://localhost:8080/passports?category=BATTERY_INDUSTRIAL" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "batteryModel": "PowerCell X1",
    "serialNumber": "SN-998877",
    "chemistry": "LFP",
    "ratedCapacity": 105.5,
    "weight": 12.4,
    "manufacturingDate": "2025-05-20",
    "manufacturingPlace": "Berlin, DE",
    "warrantyDurationYears": 5,
    "cycleLife": 3000,
    "carbonFootprint": {
      "totalCarbonFootprint": 150.5,
      "shareOfRenewables": 85.0
    },
    "materialComposition": [
      {"material": "Lithium", "percentage": 12.5},
      {"material": "Iron", "percentage": 22.0},
      {"material": "Phosphate", "percentage": 30.0}
    ]
  }'
```
**Response:** Note the `passportId` (e.g., `550e8400-e29b-41d4-a716-446655440000`).

### Step 2: Publish the Passport (Lock It)
This uploads the data to Minio (S3) with Object Locking and updates the status to `PUBLISHED`.

```bash
PASSPORT_ID="<YOUR_UUID_FROM_STEP_1>"

curl -X POST "http://localhost:8080/passports/$PASSPORT_ID/publish" \
  -H "Authorization: Bearer $TOKEN"
```
**Response:** Check that `status` is `PUBLISHED`, `storageLocation` is set, and `immutabilityHash` is present.

### Step 3: Verify Storage (Minio)
You can inspect the uploaded file in the Minio Console.

1. Open [http://localhost:9001](http://localhost:9001) in your browser.
2. Login with `minio_admin` / `minio_password`.
3. Navigate to the `passports` bucket.
4. You should see a file named `passports/<UUID>.json`.
5. Check the "Object Locking" status (Governance Mode).

### Step 4: Resolve the Passport (Public Read)
The Resolver API can now fetch the published passport.

```bash
curl -X GET "http://localhost:8081/r/$PASSPORT_ID"
```
