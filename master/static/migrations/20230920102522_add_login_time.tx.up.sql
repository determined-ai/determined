ALTER TABLE public.users ADD COLUMN last_login timestamptz NULL;

-- backfill with sessions
UPDATE public.users u
SET last_login = s.last_login
FROM (
    -- 7 days is the current hardcoded session duration
    SELECT user_id, MAX(expiry) - interval '7 days' as last_login
    FROM public.user_sessions
    GROUP BY user_id
) s
where s.user_id = u.id;
