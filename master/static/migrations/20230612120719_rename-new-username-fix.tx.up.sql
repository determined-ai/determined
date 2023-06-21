UPDATE public.groups
  SET group_name = user_id || 'DeterminedPersonalGroup'
  WHERE user_id IS NOT NULL;
