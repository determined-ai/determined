UPDATE public.models
    SET name = REPLACE(name, '--', '/');
