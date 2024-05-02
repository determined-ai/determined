### What is db_code?

```db_code``` is a way to avoid needing to make really difficult changes to update database code (views / triggers / functions).

### How does db_code work?

On everytime the Determined master starts up

- The Postgres schema ```determined_code``` will be dropped if it exists.
- Migrations run as they did before ```determined_code``` was added.
- The Postgres schema ```determined_code``` will be recreated.
- All SQL files in the ``db_code`` will be run in lexicographically order.

### Limitations

Migrations can't see or use views because they are created after ```determined_code``` is created.

In the unlikely event this is an issue, you can track views in regular migrations.