# Conditional resource writes

AgentGroup, AgentPackage, AgentRemoteConfig and Namespace use an explicit public
revision contract. `metadata.resourceVersion` is a JSON string containing an opaque
server revision. Read it from GET, LIST or a successful mutation; do not manufacture
or reset it.

- **Create (POST):** inserts only if the logical namespace/name is absent. Existing
  resources, concurrent creators and tombstones return 409 without overwriting data.
  A failure to read storage propagates as an error rather than authorizing creation.
- **Update (PUT):** send the revision from the object originally edited in
  `metadata.resourceVersion`. A missing or nonpositive revision returns 400; a
  changed desired state with a stale revision returns 409. The path determines
  identity; a different nonempty name/namespace in the body returns 400. Status,
  creation/deletion timestamps and identity remain server-owned.
- **No-op and response-loss retry:** an update carrying a positive revision whose
  desired fields already equal the current resource returns the current object,
  including its revision, without writing or propagating it. Thus retrying an
  update whose response was lost is safe. A retried create returns 409; read the
  existing object to determine whether it is the intended result.
- **Delete (DELETE):** send `?resourceVersion=<original revision>`. Missing or
  malformed revisions return 400, stale revisions return 409. A successful deletion
  advances the revision and retains a tombstone. Retrying the same delete against
  that tombstone returns 204 without advancing it again.

A tombstone reserves its logical name permanently. POST does not recreate it, and
PUT cannot undelete it. This prevents a stale controller from deleting a replacement
whose revision restarted at one. There is no public purge or recreation operation
in this change; use a new logical name.

Go client updates carry the revision in their resource DTO. The four client delete
methods now require a revision argument. `opampctl delete` reads once before
sending its conditional delete. Web editors retain the original resource, namespace
and revision while open. Conflicts require closing the editor, refreshing and
reopening; the client does not silently rebase an edit onto a newer read.

## Storage upgrade

Stop old server versions before upgrading: they do not honor this contract. MongoDB
startup creates unique logical-key indexes for versioned resources before serving
traffic. Descending unique indexes can coexist with the old ascending non-unique
indexes on MongoDB 4.4, so migration builds the unique replacement before removing
the old index. Concurrent startup never drops the unique replacement. Existing
duplicate logical keys cause startup to fail; migration never chooses a winner or
deletes resource data. Resolve duplicates explicitly before retrying the upgrade.

Startup then sets missing/zero stored revisions to one without changing positive
revisions. Version zero means insert-only in persistence, never legacy migration.
This also applies to the other existing users of the shared CAS helper (endpoints,
remote config schemas, hosts, containers and applications); their public mutation
APIs are outside this four-resource contract.

MongoDB namespace cascades run in a transaction, validate the client revision before
touching children and use CAS for the namespace row. The existing in-memory
transaction runner does not provide multi-resource rollback; that development-mode
limitation remains. Single-resource creates and versioned writes share the same
atomicity/conflict contract in both adapters.
