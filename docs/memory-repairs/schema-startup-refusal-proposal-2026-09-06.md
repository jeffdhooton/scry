# Refuse incompatible memory stores without mutation

Private prevention experiment only. No schema bump, migration, format change,
deployed binary change or live store mutation is authorized by this file.

Existing ensureSchema deletes all database content whenever a nonzero integer
version differs from1. Existing tests explicitly expect this loss. Replace
that policy with refusal. Initialize a schema marker only in a genuinely
empty database. Missing-marker nonempty stores, null/zero/negative/future
versions and malformed markers must fail without changing any raw key/value.
Keep normal schema1 startup byte-preserving. Check/read/init in one transaction.

Required fixtures: all raw families including unknown/empty payloads preserved
through refused Open; supported1 opens unchanged; empty new store initializes;
concurrent/repeated opens do not erase or reinitialize; full live-backup direct
restore plus candidate Open remains raw-equal. Full no-CGO source tests and a
fresh independent adversarial reviewer before shared promotion.

Limit: this does not retroactively protect old installed or retained binaries.
They can still erase a different numeric version. A future format transition
needs its own backwards-refusal encoding/bridge and rollback gate. Do not bump
the live schema merely because the new source itself would refuse it safely.

Restore separately calls DropAll BEFORE reading its input. This experiment does
not fix that destructive restore design or approve using it on a live/populated
store. Production restore safety requires staged validation and a reviewed
atomic replacement/recovery plan; all current proof restores use direct Badger
Load into newly created private directories, not overwrite of existing data.
