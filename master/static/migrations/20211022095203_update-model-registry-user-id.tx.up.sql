WITH det_id as (SELECT id from public.users where username LIKE 'determined' LIMIT 1) UPDATE public.models SET user_id = det_id.id FROM det_id WHERE user_id is NULL;
ALTER TABLE public.models ALTER COLUMN user_id SET NOT NULL;
