UPDATE public.groups
  SET group_name = users.username || 'DeterminedPersonalGroup'
  FROM public.users
  WHERE public.groups.user_id = public.users.id
  AND public.groups.user_id IS NOT NULL;
