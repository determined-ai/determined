# Optional Database Migrations

This directory contains optional database migrations. While they may be needed to view historical 
data for some product features, they are time-consuming and not necessary for all use cases.  
Unlike the migrations in `master/static/migrations/`, these migrations will not run automatically 
on cluster startup. Instead, they can be run manually with the following command:

```bash
psql -d $DET_DB_NAME --single-transaction -f $PATH_TO_MIGRATION
```
