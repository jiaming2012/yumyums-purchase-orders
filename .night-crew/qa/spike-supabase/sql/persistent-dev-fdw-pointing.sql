-- persistent-dev-fdw-pointing.sql — the PERSISTENT FDW→HQ pointing for the
-- operator's dev environment. Card `sync-live-in-dev-substrate` (Activity 5,
-- run 20260810), leg 3 of the card (the FDW-persistence finding the spike flagged).
--
-- ═══════════════════════════════════════════════════════════════════════════
-- WHAT THIS IS, AND WHY IT IS TWO HALVES
--
-- The substrate's production `submission_responses` SELECT policy resolves the
-- per-user RLS THROUGH the FDW server `hq_pg` to HQ's live source views
-- (migrations 0073/0074: hq_sync_template_assignees, hq_sync_user_roles,
-- hq_sync_field_templates, hq_sync_template_approvers). In the running substrate
-- that server points at `host.docker.internal:5434/hq_test_b2_fdw` — the Go RLS
-- suite's TRANSIENT fixture DB, which is DEAD when the suite is not running, so
-- the policy is INERT (spike ledger §"the FDW entanglement").
--
-- Spike F made the real app read resolve by REPOINTING the server at a scratch HQ
-- and RESTORING it in teardown. That is right for a spike. A PERSISTENT dev
-- environment needs the pointing to STAY: the substrate must resolve real per-user
-- RLS against the operator's LIVE dev HQ, between runs, without a repoint+restore
-- dance. That is what this file arranges, in two halves applied to two databases:
--
--   HALF A (HQ side)  — ALTER ROLE hq_sync_fdw LOGIN. Migration 0073 creates the
--                       role NOLOGIN with no password ON PURPOSE (it runs on every
--                       backend start including prod, and a committed password
--                       would be a shared credential). Enabling it is a deliberate,
--                       per-environment operator step — 0073's own banner says so:
--                         "ALTER ROLE hq_sync_fdw LOGIN PASSWORD '<per env>';"
--                       This half is that step, for the dev environment.
--
--   HALF B (substrate side) — ALTER SERVER hq_pg to point at the operator's live
--                       dev HQ, so the foreign tables resolve against real data.
--
-- ═══════════════════════════════════════════════════════════════════════════
-- 🛑 B-164 — THIS FILE NAMES A DEV COORDINATE. IT MUST NEVER NAME :5433 BLINDLY.
--
-- :5433 (yumyums-dev-pg) is the dev AND production cluster that serves
-- https://hq.yumyums.kitchen — a probe there destroyed the prod database on
-- 2026-08-06 (B-141/B-143, decision 155). This file is APPLIED by
-- `task sync:dev:fdw`, which substitutes the coordinate from environment
-- variables and REFUSES a bare :5433 target unless the operator sets the explicit
-- override HQ_FDW_ALLOW_5433=1 (the dev HQ genuinely lives on :5433 on the Windows
-- box, reached over Tailscale/LAN, so the operator CAN knowingly point at it — but
-- never by default, and never from an unattended run). The proof
-- (`sync-dev-proof.sh`) uses a fresh scratch HQ on an ephemeral port instead, so
-- the MECHANISM is proven without the run ever touching :5433.
--
-- ═══════════════════════════════════════════════════════════════════════════
-- PSQL VARIABLES this file reads (all supplied by `task sync:dev:fdw`):
--
--   :fdw_password   the password hq_sync_fdw logs in with. MUST equal the
--                   substrate's existing user mapping password for hq_pg
--                   (which is the throwaway 'b2-rowvis-suite-throwaway' the Go
--                   RLS suite already uses — so HALF B needs NO user-mapping edit,
--                   only the server host/port/dbname).
--   :hq_host        the host the substrate reaches the dev HQ Postgres at, from
--                   INSIDE the Docker network (typically host.docker.internal
--                   when the dev HQ Postgres is reachable from the box, or a
--                   Tailscale IP for the Windows-box dev cluster).
--   :hq_port        the dev HQ Postgres port.
--   :hq_dbname      the dev HQ database name.
--
-- The file is idempotent: re-applying it re-sets the same options and role state.

\set ON_ERROR_STOP on

-- ---------------------------------------------------------------------------
-- Dispatch: this ONE file is applied to BOTH databases, and each half guards
-- itself so applying the wrong half to the wrong database is a no-op, not a
-- corruption. `task sync:dev:fdw` applies it to the HQ Postgres (HALF A fires)
-- and to the substrate Postgres (HALF B fires). The guard is the presence of
-- the objects each half owns.
-- ---------------------------------------------------------------------------

-- ── HALF A — HQ side: give hq_sync_fdw LOGIN (migration 0073 made it NOLOGIN) ──
-- Fires only where the role hq_sync_fdw EXISTS (i.e. an HQ database with 0073
-- applied). On the substrate the role does not exist, so this half is skipped.
DO $$
BEGIN
  IF EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'hq_sync_fdw') THEN
    -- LOGIN + password. Idempotent: ALTER ROLE ... LOGIN PASSWORD re-sets both.
    -- The password comes from the psql variable so it is never committed here.
    EXECUTE format('ALTER ROLE hq_sync_fdw LOGIN PASSWORD %L', :'fdw_password');
    -- CONNECT on the current database so the FDW can actually open a session.
    EXECUTE format('GRANT CONNECT ON DATABASE %I TO hq_sync_fdw', current_database());
    RAISE NOTICE 'HALF A applied: hq_sync_fdw is now LOGIN on database %', current_database();
  ELSE
    RAISE NOTICE 'HALF A skipped: role hq_sync_fdw does not exist here (not an HQ database) — this is the substrate half''s target';
  END IF;
END
$$;

-- ── HALF B — substrate side: repoint the hq_pg foreign server at the dev HQ ──
-- Fires only where the foreign server hq_pg EXISTS (i.e. the substrate with
-- sync-schema/sql/0002_hq_fdw.sql applied). On an HQ database the server does not
-- exist, so this half is skipped.
DO $$
DECLARE
  v_have_host bool;
  v_have_port bool;
  v_have_db   bool;
BEGIN
  IF EXISTS (SELECT 1 FROM pg_foreign_server WHERE srvname = 'hq_pg') THEN
    -- ALTER SERVER's SET/ADD asymmetry: SET fails if the option is absent, ADD
    -- fails if it is present. The substrate seeds hq_pg WITH host/port/dbname
    -- (0002_hq_fdw.sql), so SET is correct here — but we compute per-option to
    -- stay idempotent even if a future substrate change drops one.
    SELECT
      bool_or(o LIKE 'host=%'),
      bool_or(o LIKE 'port=%'),
      bool_or(o LIKE 'dbname=%')
      INTO v_have_host, v_have_port, v_have_db
      FROM pg_foreign_server, unnest(srvoptions) o
     WHERE srvname = 'hq_pg';

    EXECUTE format('ALTER SERVER hq_pg OPTIONS (%s host %L)',
                   CASE WHEN v_have_host THEN 'SET' ELSE 'ADD' END, :'hq_host');
    EXECUTE format('ALTER SERVER hq_pg OPTIONS (%s port %L)',
                   CASE WHEN v_have_port THEN 'SET' ELSE 'ADD' END, :'hq_port');
    EXECUTE format('ALTER SERVER hq_pg OPTIONS (%s dbname %L)',
                   CASE WHEN v_have_db THEN 'SET' ELSE 'ADD' END, :'hq_dbname');

    -- Drop cached connections so the next read reconnects to the new target.
    PERFORM postgres_fdw_disconnect_all();
    RAISE NOTICE 'HALF B applied: foreign server hq_pg -> %:%/%',
      :'hq_host', :'hq_port', :'hq_dbname';
  ELSE
    RAISE NOTICE 'HALF B skipped: foreign server hq_pg does not exist here (not the substrate) — this is the HQ half''s target';
  END IF;
END
$$;
