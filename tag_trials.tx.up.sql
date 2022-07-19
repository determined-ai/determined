ALTER table trials ADD COLUMN IF NOT EXISTS tags jsonb;
CREATE INDEX trials_tags_index ON trials USING GIN (tags);
update trials set tags = '{}';

CREATE TABLE trials_collections (
-- table to store a set of filters as defined in QueryFilters in api/trial.proto
  id integer PRIMARY KEY
	user_id integer REFERENCES public.users(id) NOT NULL, --want index
	name text NOT NULL,
	filters jsonb,
);