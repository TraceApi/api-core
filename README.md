# TraceApi


#### Project structure

```
.
├── cmd/
│   ├── api-ingest/
│   └── api-resolver/
├── deploy/
├── internal/
│   ├── core/
│   │   ├── domain/
│   │   ├── ports/
│   │   └── service/
│   ├── platform/
│   │   ├── cache/
│   │   └── storage/
│   └── transport/
│       └── rest/
├── schemas/
├── docker-compose.yml
├── go.mod
├── LICENSE
├── Makefile
└── README.md
```


## License

This project is licensed under the **Business Source License 1.1 (BSL)**.

* **Free Use:** You can use this code for development, testing, and personal projects.
* **Commercial Use:** Running this in production (making money from it) requires a commercial license.
* **Open Source Promise:** On **November 21, 2029**, this code automatically converts to **AGPLv3** (Open Source).

For commercial licensing, contact: license@traceapi.eu

## Running

To start the entire system (Infrastructure + Applications):

```bash
docker-compose -f docker-compose.yml -f docker-compose.apps.yml up --build
```

*   **Infrastructure**: Postgres (5432), Redis (6379), MinIO (9000/9001)
*   **Applications**:
    *   `api-ingest`: http://localhost:8080
    *   `api-resolver`: http://localhost:8081