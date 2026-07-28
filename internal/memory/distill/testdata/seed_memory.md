---
name: fixture-seed-memory
description: "A hand-authored memory seed documenting how the on-call rotation escalation works, so extraction has a known-good non-transcript source to test against."
---

# On-call escalation

When a page fires outside business hours, the on-call engineer has 15
minutes to acknowledge before it escalates to the secondary.

## Steps

1. Acknowledge in PagerDuty.
2. Post a one-line status in #incidents.
3. If unresolved after 30 minutes, page the secondary manually.
