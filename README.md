# Hezzl Goods Management System
A RESTful API service for managing goods with prioritization, built with Go, PostgreSQL, Redis, NATS, and ClickHouse.

Features
Goods Management:
- Create, read, update, and delete goods
- Reprioritize goods with automatic reordering
- Paginated listing with filtering

Infrastructure:
- PostgreSQL for primary data storage
- Redis for caching
- NATS for event streaming
- ClickHouse for logging

Monitoring:
- Structured logging with Zerolog
- Comprehensive test coverage

## Getting Started

1. Clone the Repository
```bash
git clone https://github.com/romanpitatelev/hezzl-goods.git
cd hezzl-goods
```

2. Run with Docker Compose
```bash
make up
```

3. Build and Run the Application
```bash
make build
make run
```

4. Run Tests
```bash
make test
```

5. Create Docker image
```bash
make image
```
