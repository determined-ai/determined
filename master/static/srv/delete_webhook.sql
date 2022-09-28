DELETE FROM webhooks
  WHERE id = $1
RETURNING webhooks.id;
