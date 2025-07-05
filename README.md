# PostgreSQL Streaming Replication Demo

This repository contains a Docker Compose setup for learning how PostgreSQL streaming replication works. The environment runs a primary server, a standby server initialised by `pg_basebackup` and a pgAdmin instance for administration.  It also includes a small Go application that demonstrates read/write splitting and provides basic tests.

## Directory overview

- `docker-compose.yml` – orchestrates the containers
- `primary/` and `standby/` – PostgreSQL configuration files
- `scripts/` – setup scripts executed when the containers initialise
- `app/` – Go sample application and tests
- `*.md` – additional documentation

## Prerequisites

- Docker and Docker Compose
- Go (for running the demo application)

## Quick start

1. Copy the environment template and edit the passwords:
   ```bash
   cp .env.example .env
   # open .env and set secure values
   ```
2. Start the containers:
   ```bash
   docker compose up -d
   ```
3. Check container status and logs:
   ```bash
   docker compose ps
   docker compose logs postgres-primary
   docker compose logs postgres-standby
   ```
4. Confirm replication:
   ```bash
   docker exec -it postgres-primary psql -U postgres -d testdb -c "SELECT * FROM pg_stat_replication;"
   docker exec -it postgres-standby psql -U postgres -d testdb -c "SELECT * FROM pg_stat_wal_receiver;"
   ```
5. Stop the environment:
   ```bash
   docker compose down
   ```

pgAdmin will be available on <http://localhost:8080> (credentials are defined in `.env`).

## Go demo application

Inside `app/` you will find a simple program that writes data to the primary server using `docker exec` and reads from the standby using a direct connection. The code also includes a test suite.

Run the tests with:

```bash
cd app
go mod tidy
go test -v
```

## Security notes

This project is for demonstration purposes. Default passwords are provided only for convenience – **always** change them in your `.env` file. In production you should further restrict network access and consider enabling SSL/TLS as outlined in `SECURITY.md`.

## Further reading

More detailed explanations of the experiment, troubleshooting steps and development notes can be found in the other Markdown files in this repository.
