WITH w AS (
  INSERT INTO webhooks (url)
  VALUES ($1)
  RETURNING id, url
)
SELECT w.id, w.url
FROM w

