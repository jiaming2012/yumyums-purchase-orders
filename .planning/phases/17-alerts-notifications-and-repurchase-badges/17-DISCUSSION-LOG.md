# Phase 17: Alerts, Notifications, and Repurchase Badges - Discussion Log

> **Audit trail only.** Do not use as input to planning, research, or execution agents.
> Decisions are captured in CONTEXT.md — this log preserves the alternatives considered.

**Date:** 2026-04-22
**Phase:** 17-alerts-notifications-and-repurchase-badges
**Areas discussed:** Alert delivery channel, Notification preferences UI, Cutoff reminder timing, Repurchase badge behavior
**Mode:** auto (all areas auto-selected with recommended defaults)

---

## Alert Delivery Channel

| Option | Description | Selected |
|--------|-------------|----------|
| Zoho Cliq webhook + email fallback | Primary channel via incoming webhook, email as alternative | ✓ |
| Email only | Simple SMTP delivery, no chat integration | |
| Push notifications | PWA push API — requires notification permission | |

**User's choice:** [auto] Zoho Cliq webhook + email fallback (recommended default per ALRT-03/ALRT-05)
**Notes:** Matches requirements — Zoho Cliq is the team's existing communication tool.

---

## Notification Preferences UI

| Option | Description | Selected |
|--------|-------------|----------|
| Users tab per-user setting | Admin/self-edit in existing Users management | ✓ |
| Separate settings page | Dedicated notification settings route | |
| System-wide only | Single admin-configured channel for everyone | |

**User's choice:** [auto] Users tab per-user setting (recommended default per ALRT-04)
**Notes:** Leverages existing Users tab — no new page needed.

---

## Cutoff Reminder Timing

| Option | Description | Selected |
|--------|-------------|----------|
| 24 hours before | Single reminder, one day ahead | ✓ |
| 1 hour before | Last-minute nudge | |
| Multiple (24h + 1h) | Two reminders at different intervals | |

**User's choice:** [auto] 24 hours before (recommended default)
**Notes:** Single reminder avoids alert fatigue for small crew.

---

## Repurchase Badge Behavior

| Option | Description | Selected |
|--------|-------------|----------|
| Badge on stock items after shopping completion, weekly reset | Shows +Qty, admin-configurable reset date | ✓ |
| Badge that auto-clears after N days | Time-based expiry | |
| Permanent badge until next PO cycle | Clears when new PO is created | |

**User's choice:** [auto] Badge on stock items after shopping completion, weekly reset (recommended default per REP-01/REP-02)
**Notes:** Matches weekly PO cycle — badge shows what was restocked this week.

---

## Claude's Discretion

- Alert queue implementation approach
- Email template formatting
- Zoho Cliq message card formatting
- Database schema for alert log

## Deferred Ideas

None
