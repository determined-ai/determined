### What is db_code?

```db_code``` is a way to avoid needing to make really difficult changes to update database code (views / triggers / functions).

This migration is an example of what we want to avoid. A change adding a field should just add a field.

https://github.com/determined-ai/determined/blob/main/master/static/migrations/20230306115327_add-checkpoint-size.tx.up.sql

### How do I use db_code?

If you need to change views / other db code, just change them and the changes will apply next time master restarts.

If you need to delete views / other db code, just delete them from the file.

If you need to add a view, just make sure you add it into the ```determined_code``` schema (aka do ```CREATE VIEW determined_code.test ...```). Exception are triggers inherit from the table, therefore just create them without the schema name. The procedure being executed should still be in the ```determined_code``` so that way when the ```determined_code``` schema get's dropped the trigger will get cascaded.

### How does db_code work?

On everytime the Determined master starts up

- The Postgres schema ```determined_code``` will be dropped if it exists.
- Migrations run as they did before ```determined_code``` was added.
- The Postgres schema ```determined_code``` will be recreated.
- All SQL files in the ``db_code`` will be run in lexicographically order.

### Limitations

Migrations can't see or use views because migrations happen before ```determined_code``` is created.

In the unlikely event this is an issue, you can track views in regular migrations.