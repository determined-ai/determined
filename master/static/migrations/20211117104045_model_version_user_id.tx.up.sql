UPDATE public.model_versions
SET
    user_id
    = (
        SELECT models.user_id
        FROM public.models
        WHERE models.model_id = public.models.id
    )
WHERE user_id IS NULL;
