#!/usr/bin/env bash
# K1(c) flaky repeats, K3 legs + mutations, K4 whole spec + mutations + permissive-override probe.
# FINAL tree worktree, port 8393, --retries=0 throughout. :5434 only. Every mutation restored by git checkout.
set -u
W=/home/jcole/projects/hq-worktrees/triage-adv-20261007
L=/home/jcole/projects/hq/.night-crew/runs/2026-10-07-autonomous/logs/triage-adversarial
PW=$L/pw-leg.sh
DBN=hq_test_triage_adv_20261007_e2e
cd $W
rep() { python3 - "$1" "$2" "$3" <<'EOF'
import sys
p,old,new=sys.argv[1],sys.argv[2],sys.argv[3]
s=open(p).read(); assert s.count(old)==1, ('target not unique/found', p, s.count(old))
open(p,'w').write(s.replace(old,new,1))
EOF
}
h4=.night-crew/runs/2026-10-02-autonomous/logs/h4/states; h5=.night-crew/runs/2026-10-02-autonomous/logs/h5/states

echo "## K1(c) three previously-flaky tests, 5x each alone"
bash $PW $W 8393 $DBN k1c-2186-x5 tests/inventory.spec.js -g "Setup item editor shows alias chips, adds and removes nicknames" --repeat-each=5 --retries=0
bash $PW $W 8393 $DBN k1c-2919-x5 tests/inventory.spec.js -g "create item without group shows alert" --repeat-each=5 --retries=0
bash $PW $W 8393 $DBN k1c-2931-x5 tests/inventory.spec.js -g "creating item opens edit form with store location dropdown" --repeat-each=5 --retries=0

echo "## K3: each states spec alone"
bash $PW $W 8393 $DBN k3-stats-alone tests/states-marketing-stats.spec.js --retries=0
git diff --stat -- $h4 $h5 | tail -1; echo "k3-stats diffstat lines=$(git diff --stat -- $h4 $h5 | wc -l)"
bash $PW $W 8393 $DBN k3-subscribers-alone tests/states-marketing-subscribers.spec.js --retries=0
git diff --stat -- $h4 $h5 | tail -1; echo "k3-subs diffstat lines=$(git diff --stat -- $h4 $h5 | wc -l)"
echo "## K3: md5 of tracked PNGs vs HEAD"
for f in $(git ls-files $h4 $h5); do if ! git diff --quiet -- "$f"; then echo "CHANGED $f"; fi; done | tee $L/k3-png-changed.txt; echo "changed_pngs=$(wc -l < $L/k3-png-changed.txt)"
ls test-screenshots/marketing-stats | wc -l; ls test-screenshots/marketing-subscribers | wc -l

echo "## K3 mutation: stats default back to the tracked dir -> [SS-01]"
rep tests/states-marketing-stats.spec.js "const DEFAULT_SHOT_DIR = path.join(__dirname, '..', 'test-screenshots', 'marketing-stats');" "const DEFAULT_SHOT_DIR = path.join(__dirname, '..', '.night-crew', 'runs', '2026-10-02-autonomous', 'logs', 'h4', 'states');"
git diff -- tests/states-marketing-stats.spec.js > $L/k3-m1.diff
bash $PW $W 8393 $DBN k3-m1-stats-default-tracked tests/states-marketing-stats.spec.js -g "SS-01" --retries=0
git checkout -- tests/states-marketing-stats.spec.js $h4
echo "## K3 mutation: subscribers default back to the tracked dir -> [SS-02]"
rep tests/states-marketing-subscribers.spec.js "const DEFAULT_SHOT_DIR = path.join(__dirname, '..', 'test-screenshots', 'marketing-subscribers');" "const DEFAULT_SHOT_DIR = path.join(__dirname, '..', '.night-crew', 'runs', '2026-10-02-autonomous', 'logs', 'h5', 'states');"
git diff -- tests/states-marketing-subscribers.spec.js > $L/k3-m2.diff
bash $PW $W 8393 $DBN k3-m2-subs-default-tracked tests/states-marketing-subscribers.spec.js -g "SS-02" --retries=0
git checkout -- tests/states-marketing-subscribers.spec.js $h5
echo "## K3: STATES_SHOT_DIR honoured"
SD=/tmp/claude-1000/-home-jcole-projects-hq/d35260ba-2f67-4dc1-9824-435e818e0e51/scratchpad/adv-shots; rm -rf $SD
STATES_SHOT_DIR=$SD bash $PW $W 8393 $DBN k3-shot-dir-env tests/states-marketing-stats.spec.js --retries=0
echo "files in STATES_SHOT_DIR: $(ls $SD 2>/dev/null | wc -l)"; ls $SD | head -3

echo "## K4: whole tests/marketing.spec.js --retries=0"
bash $PW $W 8393 $DBN k4-marketing-whole tests/marketing.spec.js --retries=0

echo "## K4 mutation 1: override read removed -> [SV-12]"
rep marketing/scan-page.js "const policyOverride = window.__MARKETING_POLICY_SOURCE__;" "const policyOverride = undefined; // ADV MUTATION"
git diff -- marketing/scan-page.js > $L/k4-m1.diff
bash $PW $W 8393 $DBN k4-m1-no-override-read tests/marketing.spec.js -g "SV-12" --retries=0
git checkout -- marketing/scan-page.js
echo "## K4 mutation 2: submit-flow.js catch arms permissive -> [SV-12]"
rep marketing/submit-flow.js "  } catch (e) {
    // A throwing source is a source that resolved nothing — same arm.
    return failClosed(campaignId);
  }" "  } catch (e) {
    return false; // ADV MUTATION: permissive
  }"
rep marketing/submit-flow.js "      return !!p.unresolved;
    } catch (e) { return true; }" "      return !!p.unresolved;
    } catch (e) { return false; } // ADV MUTATION"
git diff -- marketing/submit-flow.js > $L/k4-m2.diff
bash $PW $W 8393 $DBN k4-m2-permissive-catch tests/marketing.spec.js -g "SV-12" --retries=0
git checkout -- marketing/submit-flow.js

echo "## K4 probe: PERMISSIVE override on the normal test server, no fixture beyond the init script"
cat >> tests/marketing.spec.js <<'EOF'

// THROWAWAY triage-adversarial probe (run 20261007) — never committed.
test.describe('ADV probe', () => {
  for (const withOverride of [true, false]) {
    test(`[ADV-K4] requires-online $40 code scanned offline, permissive boot override=${withOverride}`, async ({ page }) => {
      if (withOverride) {
        await page.addInitScript(() => {
          window.__MARKETING_POLICY_SOURCE__ = { policyFor: () => ({ requiresOnline: false, unresolved: false }) };
        });
      }
      await openProvisionedScanner(page);
      const calls = await mockRedeem(page);
      await seedLocal(page, { offers: [fixture5HighRow()], codes: [fixture5HighRow()], campaigns: [campaignHighRow()] });
      await page.context().setOffline(true);
      await page.evaluate(() => window.MarketingSubmit.probeNow());
      await expect(page.locator('#scan-conn')).toHaveAttribute('data-conn', 'offline');
      await scanText(page, FIXTURE_5_PAYLOAD);
      await expect(page.locator('#ms-flow')).toHaveAttribute('data-mstate', 'offerReady');
      await page.waitForTimeout(500);
      const branch = await page.locator('#ms-gate').getAttribute('data-branch').catch(() => null);
      const orderCount = await page.locator('#ms-order').count();
      const submitCount = await page.locator('[data-action="ms-submit"]').count();
      const origin = await page.evaluate(() => location.origin);
      console.log(`ADV-K4 RESULT override=${withOverride}: origin=${origin} gate-branch=${branch} order-field=${orderCount} submit-button=${submitCount} posts=${calls.length}`);
      if (withOverride) {
        expect(orderCount, 'FINDING REPRODUCED if this fails: the page OFFERS a requires-online code offline under a permissive same-origin override').toBe(0);
      } else {
        expect(branch).toBe('requires-online');
        expect(orderCount).toBe(0);
      }
    });
  }
});
EOF
bash $PW $W 8393 $DBN k4-probe-permissive-override tests/marketing.spec.js -g "ADV-K4" --retries=0
grep 'ADV-K4 RESULT' $L/k4-probe-permissive-override.log
git checkout -- tests/marketing.spec.js
git status --porcelain
echo POST_K1_DONE
