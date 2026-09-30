-- +goose Up
BEGIN;

-- ===========================================================================
-- 0082 — Trends and Cost move out of Inventory into the BI app; one grant.
--
-- Until B-455 / WO-2b the two owner-only reports were gated as PER-TAB rows
-- of hq_apps (`inventory-trends`, `inventory-cost`, seeded by SeedHQApps since
-- design `prove-surface-gating-and-endpoints.md` §1.4) with the whole-app
-- `inventory` grant as umbrella. They now live on bi.html behind the `bi`
-- launcher app, so the grant that opens them is the `bi` grant and nothing
-- else — an `inventory` grant no longer implies them; that IS the
-- consolidation the mockup signed ("no BI grant, no tile").
--
-- Copy, don't rename (operator's preference for rollback safety):
--   1. make sure the `bi` row exists (the seed adds it on startup, but a
--      migration must not depend on the seed having run);
--   2. copy every grant on either tab row onto `bi` — role grants and user
--      grants alike — skipping any (app, role) / (app, user) pair `bi`
--      already holds (app_permissions_role_idx / app_permissions_user_idx are
--      PARTIAL unique indexes, so NOT EXISTS is the idempotent shape, and
--      DISTINCT keeps the two tab rows from inserting the same pair twice);
--   3. disable the two tab rows. auth.RequirePermission joins on
--      `a.enabled = true`, so a disabled row grants nothing — the grants stay
--      on disk for the Down path, and SeedHQApps no longer seeds the rows, so
--      a fresh database never gets them.
-- ===========================================================================

INSERT INTO hq_apps (slug, name, icon) VALUES ('bi', 'BI', '📊')
ON CONFLICT (slug) DO NOTHING;

INSERT INTO app_permissions (app_id, role)
SELECT DISTINCT bi.id, p.role
FROM app_permissions p
JOIN hq_apps tab ON tab.id = p.app_id
JOIN hq_apps bi  ON bi.slug = 'bi'
WHERE tab.slug IN ('inventory-trends', 'inventory-cost')
  AND p.role IS NOT NULL
  AND NOT EXISTS (
    SELECT 1 FROM app_permissions q WHERE q.app_id = bi.id AND q.role = p.role
  );

INSERT INTO app_permissions (app_id, user_id)
SELECT DISTINCT bi.id, p.user_id
FROM app_permissions p
JOIN hq_apps tab ON tab.id = p.app_id
JOIN hq_apps bi  ON bi.slug = 'bi'
WHERE tab.slug IN ('inventory-trends', 'inventory-cost')
  AND p.user_id IS NOT NULL
  AND NOT EXISTS (
    SELECT 1 FROM app_permissions q WHERE q.app_id = bi.id AND q.user_id = p.user_id
  );

UPDATE hq_apps SET enabled = false
WHERE slug IN ('inventory-trends', 'inventory-cost');

COMMIT;

-- +goose Down
BEGIN;
-- Re-open the tab rows; the copied `bi` grants are left in place (a grant an
-- operator may since have edited is not a migration's to take back).
-- NOTE: Down is only a rollback together with the code that mounted these two
-- slugs (pre-WO-2b main.go and seed). On current code the re-enabled rows gate
-- nothing and merely reappear in Users › Access; the `bi` grants keep working.
UPDATE hq_apps SET enabled = true
WHERE slug IN ('inventory-trends', 'inventory-cost');
COMMIT;
