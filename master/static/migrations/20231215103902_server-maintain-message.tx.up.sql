CREATE TABLE maintenance_messages (
  id SERIAL PRIMARY KEY,
  message TEXT NOT NULL,
  user_id INT REFERENCES public.users(id),
  start_time TIMESTAMP with time zone NOT NULL DEFAULT NOW(),
  end_time TIMESTAMP with time zone NOT NULL DEFAULT NOW()
);
