### What is views_and_triggers?

```views_and_triggers``` is a way to avoid needing to make really difficult changes to update database code (views / triggers / functions). Any stateless database thing can be stored here. The name is not 100% accurate.

This migration is an example of what we want to avoid. A change adding a field should just add a field.

https://github.com/determined-ai/determined/blob/main/master/static/migrations/20230306115327_add-checkpoint-size.tx.up.sql

### How do I use views_and_triggers?

If you need to change views / other db code, just change them and the changes will apply next time master restarts.

If you need to delete views / other db code, just delete them from the file.

If you need to add a view add it in a ```.sql``` file in the ```up``` folder and add a delete statement in the ``down.sql`` in this directory.

### How does dbviews_and_triggers work?

On everytime the Determined master starts up we check if the database views have changed.

- The ``down.sql`` file runs.
- Migrations run as they did before ```determined_code``` was added.
- All SQL files in the ``views_and_triggers`` will be run in lexicographical order.

### Limitations

Migrations can't see or use views because migrations happen before ```determined_code``` is created.

In the unlikely event this is an issue, you can track views in regular migrations.