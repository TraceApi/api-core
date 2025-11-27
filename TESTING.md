# Testing Guide

Follow these steps to validate the TraceApi Core implementation.

## 1. Start Infrastructure

Start the PostgreSQL and Redis containers:

```bash
make up
```

Wait a few seconds for the database to be ready.

## 2. Initialize Database

Apply the initial schema to the database:

```bash
make db-init
```

## 3. Run the Applications

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

## 4. Test Scenarios

### Scenario A: Create a Battery Passport (Success)

**Request:**
```bash
curl -X POST "http://localhost:8080/v1/passports?category=BATTERY_INDUSTRIAL" \
  -H "Content-Type: application/json" \
  -H "X-Manufacturer-ID: tesla-gigafactory-berlin" \
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
