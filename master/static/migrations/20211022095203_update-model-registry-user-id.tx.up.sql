WITH det_id AS (
    SELECT id FROM public.users WHERE username LIKE 'determined' LIMIT 1
)

UPDATE public.models SET user_id = det_id.id
FROM det_id
WHERE user_id IS NULL
;
ALTER TABLE public.models ALTER COLUMN user_id SET NOT NULL;
