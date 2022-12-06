# Database Migrations

We use `go-pg/migrations`: <https://github.com/go-pg/migrations>

## Running migrations manually

```bash
determined-master [MASTER_ARGS] migrate [MIGRATION_ARGS]
```

where `MASTER_ARGS` are the normal determined-master command flags,
and `MIGRATION_ARGS` are `go-pg/migrations` args.

For example, to migrate down to a specific version, run

```bash
determined-master --config-file /path/to/master.yaml migrate down 20210917133742
```

## Creating new migrations

We use timestamps instead of sequential numbers, standard for `go-pg/migrations`.
When creating a new migration, either write it manually, or try:

```bash
./migration-create.sh my-migration-name
```

If there is a chance another migration has landed between when you created
yours and when your PR lands, you should update your filename so the migrations
land in-order:

```bash
./migration-move-to-top.sh my-migration-name
```
