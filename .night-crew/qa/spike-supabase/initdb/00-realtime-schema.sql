-- Runs once, at first `initdb`, inside the supabase/postgres container.
--
-- Realtime's boot sequence is `/app/bin/migrate` -> seeds -> server. The
-- migrator connects with `search_path = _realtime` (DB_AFTER_CONNECT_QUERY in
-- docker-compose.supabase.yml) and expects that schema to already exist; it
-- does not create it. Without this file the realtime container dies on boot
-- with a search_path/relation error and never gets as far as seeding a tenant.
--
-- This mirrors `volumes/db/realtime.sql` in Supabase's own self-hosted compose.
create schema if not exists _realtime;
alter schema _realtime owner to supabase_admin;
