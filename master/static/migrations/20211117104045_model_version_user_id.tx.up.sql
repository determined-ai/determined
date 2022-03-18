UPDATE public.model_versions
SET user_id = (SELECT user_id FROM public.models WHERE model_id = public.models.id)
WHERE user_id IS NULL;
