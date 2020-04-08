# Database Migrations

## Install migrate CLI

https://github.com/golang-migrate/migrate/tree/master/cmd/migrate

## Example Commands

```bash

export DET_DB_PASSWORD=my-postgres-password
export CONNECTION_STRING=postgres://postgres:${DET_DB_PASSWORD}@localhost:5432/determined'?'sslmode=disable

# Create template migration files.
migrate \
  -database ${CONNECTION_STRING} \
  -verbose \
  create -ext sql -dir $(pwd) title-of-my-migation

# Edit template sql files.

# Run master.
```

## If things go wrong

```
# To manually migrate...
migrate \
  -database ${CONNECTION_STRING} \
  -path . \
  -verbose \
  up

# To fix broken migrations
migrate \
  -database ${CONNECTION_STRING} \
  -path . \
  -verbose \
  version
migrate
  -database ${CONNECTION_STRING} \
  -path . \
  -verbose \
  force version-number
migrate
  -database ${CONNECTION_STRING} \
  -path . \
  -verbose \
  down 1
```
