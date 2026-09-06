# Private ordered candidate registration contract

Baselinee13f7ae. Implements design19474eeb only with complete reviewcd07b5b4
corrections. No production callers, support/ownership/lifecycle policy, B evidence,
durable observations/current result, cleanup or deployment approval.

One private root wrapper takes canonical parsed revision key/raw and a private
registry BODY callback. It uses the reviewed serialized owner and initializes its
OWN copied revision, complete identity mutation ledger, complete fact mutation
ledger and registry before invoking the callback. No caller baseline/ledger/handle/
support choice/finalizer argument. Fixed internal finalizing verifies both ledgers
and registration coverage. Return a fully owned report ONLY after actual commit;
error gives a zero report; a body panic propagates with rollback. As in the existing
coordinator, an observer panic occurs after commit and cannot roll it back; the
wrapper does not return a report through that panic. Body-returned/prior errors
keep outer precedence. Local structural/storage errors are static registration
sentinel plus allowlisted TxnTooBig/Conflict and poison even when caught.

Register uses one full canonical primary observation matched to the owned revision.
Derive birth tuple internally from original declaration Name or original src/dst,
not a resolved/flipped side or Supersedes. Same complete first observation/role
retry returns the same issued handle even after materialization; same birth tuple
with different first observation is NOT an exact retry. Different mentions require
the explicit mechanical link operation, which never grants routing/support.

Validate genuine phase/owner/facade and prior poison, then canonical input before
any result. For an unregistered derived slug, baseline en presence yields a typed
existing/not-new observation without a handle or lifecycle certificate. Otherwise
baseline and current ig:/il:/il-consumed:/rs:/rt: exact birth-name/slug control keys
must be absent; current en must remain absent; ANY earlier owned identity writer
entry with that actor refuses first registration, even a reverted alias/entity
mutation. Old aliases/attestations never establish identity authority. This does
not detect a prior memory-only proposal or untracked transient raw writer; future
all-producer routing must gate those entry points with issued handles.

After structural/lifecycle/history checks, any current OR historical baseline fact
reference gives normal deferred-preexisting-reference disposition, no error/poison
or handle/birth/generation authority. Store the complete matched observation and
role. Baseline references remain authoritative even if body deletes them. Do not
mask poison/malformed data/occupied controls with cached or deferred results.
Exact duplicate normal existing/deferred observations are deduplicated only in the
report, not used to bypass checks. Different observations remain distinct.

First successful registration owns the full birth, first observation and current
identity writer position. All private identity-writing facade methods require an
issued registration for baseline-absent actors before forwarding actual writes.
Freeze ALSO audits the owned history, so calling the lower-level owned ledger
cannot retroactively evade ordering or creation coverage. Every baseline-absent
actor's en/al entry must have prior registration (entry sequence >= position).
Every actual absent-to-present en creation must match exact stable slug/name/time.
All later en after-images must keep that tuple; deletion is allowed but recreation
after deletion refuses. At most one recorded creation per candidate. Include
created-then-deleted transitions; unmaterialized candidates are allowed and remain
descriptive proposals. Existing baseline actors are not newly registered identities.
This does not authorize their updates or certify alias ownership/post-undo closure.

Mention links validate issued pointer membership, same owner/registry and complete
revision-matching observations. Closed roles are declaration, primary-src/dst and
supersedes-src/dst. Declaration/primary role matches original observation side;
hint roles require a nonnil original Supersedes and retain its carrying primary
observation. Deduplicate only exact key/role; the same parsed observation may be a
proposed link to multiple candidates, NOT a routing/alias certificate. Birth first
origin never changes. Nil/foreign/copied/closed handles and early/late use refuse.

Registry operations themselves write no store rows/events. Explicit wrapper writer
methods delegate owned en/al/fa actual ledgers and retain their mechanical scope;
the ordinary facade can stage episode/queue data, still transactional. Fact ordinal
labels and counts remain descriptive, not assertion permission/support. The whole
report owns revision bytes, candidate first/link data, normal dispositions, complete
identity accounting and fact accounting; no buffer aliases internal state or other
returned projections. Freeze is private and automatic, once; caller cannot supply
a result, omit it, or continue callback work afterward. Low-level forged private
state, concurrent/escaped facades and raw transient/coordinator exclusions remain.

No EP provenance promise: parsed revision is pure during body because ep: is normally
last. Future immutable writers must prove actual EP before persisting observations.
No Force merge: earlier selected birth inventory must be retained via the separate
complete current-result policy, never inferred from this transaction's candidates.
A registered unsupported entity can STILL commit under this uncalled accounting
unit; eventual fixed support/undo/lifecycle policy is required before routing Apply.
