CREATE TABLE cluster_messages (
  message TEXT NOT NULL,
  created_by INT REFERENCES public.users(id),
  start_time TIMESTAMP with time zone NOT NULL DEFAULT NOW(),
  end_time TIMESTAMP with time zone DEFAULT NULL,
  created_time TIMESTAMP NOT NULL DEFAULT NOW()
);
