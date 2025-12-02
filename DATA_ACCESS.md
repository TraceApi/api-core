# Data Access & Compliance Strategy

This document defines how TraceApi handles data visibility in compliance with EU Regulations (ESPR, Battery Regulation 2023/1542).

## 1. Access Levels

We categorize all data fields into two levels:

| Level | Audience | Description | Example Fields |
| :--- | :--- | :--- | :--- |
| **Public** | Consumers, General Public | Information required for informed purchase, use, and disposal. | Model, Carbon Footprint, Material Composition, Safety Instructions. |
| **Restricted** | Regulators, Recyclers, Repairers | Sensitive technical data, trade secrets, or safety-critical repair info. | Disassembly Instructions, Part Numbers, Supplier Names, Test Reports. |

## 2. Implementation Strategy: Schema-Driven

We do **not** hardcode field lists in the application code. Instead, we use the JSON Schema as the Single Source of Truth.

### Schema Metadata
Each property in our JSON Schemas (`internal/core/service/schemas/payloads/*.json`) must have an `access` attribute:

```json
"carbonFootprint": {
  "type": "object",
  "access": "public",
  ...
},
"disassemblyInstructions": {
  "type": "object",
  "access": "restricted",
  ...
}
```

### Filtering Logic
The `api-core` service enforces this at runtime:

1.  **Request**: `GET /passports/{id}`
2.  **Check Auth**:
    *   **No Token**: Context is `Public`.
    *   **Valid Token**: Context is `Restricted` (Full Access).
3.  **Filter**:
    *   The service loads the schema for the product category.
    *   It recursively traverses the JSON payload.
    *   If Context is `Public` and a field is marked `"access": "restricted"`, the field is **removed** from the response.

## 3. Regulatory References

*   **EU Battery Regulation (2023/1542)**: Annex XIII defines the 4 levels of access.
*   **ESPR (Ecodesign for Sustainable Products)**: Defines the general framework for the Digital Product Passport (DPP).
