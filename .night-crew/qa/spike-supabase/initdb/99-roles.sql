-- Runs once, at first `initdb`, AFTER the supabase/postgres image's own
-- migrations have created the role set. The `99-` prefix is load-bearing:
-- migrate.sh runs /docker-entrypoint-initdb.d/init-scripts/* in lexical order,
-- and `authenticator` does not exist until the image's 00000000000003-post-setup
-- script has run.
--
-- SHARP EDGE (cost us bring-up attempt 2): the supabase/postgres image creates
-- `authenticator`, `pgbouncer`, `supabase_auth_admin` and `supabase_storage_admin`
-- with NO PASSWORD. Only the superuser (`supabase_admin`) gets POSTGRES_PASSWORD.
-- PostgREST connects as `authenticator`, so without this file it loops on
--   FATAL: password authentication failed for user "authenticator"
-- and then exits(1) with `postgrest: thread killed`. Supabase's own self-hosted
-- compose ships the equivalent file as volumes/db/roles.sql; it is NOT part of
-- the image.
--
-- Passwords here are the same throwaway POSTGRES_PASSWORD as everything else in
-- this spike. See the banner in docker-compose.supabase.yml.
\set pgpass `echo "$POSTGRES_PASSWORD"`

ALTER USER authenticator          WITH PASSWORD :'pgpass';
ALTER USER pgbouncer              WITH PASSWORD :'pgpass';
ALTER USER supabase_auth_admin    WITH PASSWORD :'pgpass';
ALTER USER supabase_storage_admin WITH PASSWORD :'pgpass';
