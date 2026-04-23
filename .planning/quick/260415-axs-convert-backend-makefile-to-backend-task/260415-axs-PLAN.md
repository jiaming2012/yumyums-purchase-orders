---
phase: quick
plan: 260415-axs
type: execute
wave: 1
depends_on: []
files_modified:
  - backend/Taskfile.yml
  - backend/Makefile
autonomous: true
requirements: []
must_haves:
  truths:
    - "All Makefile targets are available as Taskfile tasks with identical behavior"
    - "Makefile is deleted after Taskfile.yml is verified"
  artifacts:
    - path: "backend/Taskfile.yml"
      provides: "go-task equivalent of all Makefile targets"
  key_links: []
---

<objective>
Convert backend/Makefile to backend/Taskfile.yml using go-task format, preserving all targets and behavior. Delete the Makefile after.

Purpose: Migrate from Make to go-task for the backend build system.
Output: backend/Taskfile.yml replaces backend/Makefile
</objective>

<execution_context>
@$HOME/.claude/get-shit-done/workflows/execute-plan.md
@$HOME/.claude/get-shit-done/templates/summary.md
</execution_context>

<context>
@backend/Makefile
</context>

<tasks>

<task type="auto">
  <name>Task 1: Create Taskfile.yml and delete Makefile</name>
  <files>backend/Taskfile.yml, backend/Makefile</files>
  <action>
Create backend/Taskfile.yml with go-task v3 format. Convert all Makefile targets:

**Variables (top-level `vars` block):**
- DB_URL: `postgres://yumyums:yumyums@localhost:5432/yumyums?sslmode=disable`
- PORT: `8080`
- STATIC_DIR: `..`

All three should be overridable via environment variables (go-task does this automatically when using `{{.DB_URL}}` with a default in vars).

**Tasks to create (preserve exact shell commands):**

1. `dev` — runs `go run ./cmd/server` with STATIC_DIR, PORT, DB_URL, SUPERADMIN_CONFIG env vars
2. `build` — rm -rf cmd/server/public/*, cp files, go build -o bin/server ./cmd/server
3. `db-start` — docker run with all the same flags (postgres:16, volume, env vars, port)
4. `db-stop` — docker stop yumyums-pg && docker rm yumyums-pg
5. `db-reset` — depends on db-stop, then removes volume, then calls db-start. Use `deps: [db-stop]` for the stop, then cmds for volume rm and calling db-start via `task db-start`
6. `seed` — runs `go run ./cmd/seed` with DB_URL env var
7. `db-seed` — echo + docker exec psql with the INSERT statement + echo. Use a multi-line cmd block.

For the `build` task, use a `cmds` list with each shell line as a separate entry. For `db-start`, use a single cmd with line continuation or a multi-line string.

For `db-reset`: use `cmds` with three entries: `docker volume rm yumyums-pgdata`, then `task db-start`. Note: do NOT use deps for db-stop here because deps run in parallel with cmds in go-task — instead put `task db-stop` as the first cmd.

After creating Taskfile.yml, delete backend/Makefile.
  </action>
  <verify>
    <automated>cd /Users/jamal/projects/yumyums/hq/backend && cat Taskfile.yml && test ! -f Makefile && echo "OK: Makefile deleted"</automated>
  </verify>
  <done>Taskfile.yml exists with all 7 tasks matching Makefile behavior. Makefile is deleted.</done>
</task>

</tasks>

<verification>
- `cat backend/Taskfile.yml` shows all 7 tasks: dev, build, db-start, db-stop, db-reset, seed, db-seed
- `test ! -f backend/Makefile` confirms Makefile is deleted
- If go-task is installed: `cd backend && task --list` shows all tasks
</verification>

<success_criteria>
- backend/Taskfile.yml contains all targets from the original Makefile with identical commands
- backend/Makefile no longer exists
- Variable defaults (DB_URL, PORT, STATIC_DIR) are preserved and overridable
</success_criteria>

<output>
After completion, create `.planning/quick/260415-axs-convert-backend-makefile-to-backend-task/260415-axs-SUMMARY.md`
</output>
