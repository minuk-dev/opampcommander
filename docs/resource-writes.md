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

Stop old server versions before upgrading: they do not honor this contract. With
`database.ddlAuto=true`, MongoDB startup creates unique logical-key indexes for
versioned resources before serving traffic. With `ddlAuto=false`, startup only
validates the required full unique indexes and positive stored revisions, and
refuses to start if either prerequisite is missing. Run the upgrade with DDL
enabled or provision the same schema offline before disabling it. Sparse and
partial unique indexes do not satisfy this check. Descending unique indexes can
coexist with the old ascending non-unique
indexes on MongoDB 4.4, so migration builds the unique replacement before removing
the old index. Concurrent startup never drops the unique replacement. Existing
duplicate logical keys cause startup to fail; migration never chooses a winner or
deletes resource data. Resolve duplicates explicitly before retrying the upgrade.

The DDL upgrade then sets missing/zero stored revisions to one without changing positive
revisions. Version zero means insert-only in persistence, never legacy migration.
This also applies to the other existing users of the shared CAS helper (endpoints,
remote config schemas, hosts, containers and applications); their public mutation
APIs are outside this four-resource contract.

Concurrent servers applying the same bootstrap namespace manifest re-read and
retry a bounded number of times on version conflicts. A matching manifest is a
no-op, so both create and update races converge without aborting startup.

MongoDB namespace cascades run in a transaction, validate the client revision before
touching children and use CAS for the namespace row. The existing in-memory
transaction runner does not provide multi-resource rollback; that development-mode
limitation remains. Single-resource creates and versioned writes share the same
atomicity/conflict contract in both adapters.

## Partial updates (PATCH)

AgentGroup, AgentPackage, AgentRemoteConfig and Namespace accept `PATCH` on their
single-resource URL with `Content-Type: application/merge-patch+json` ([RFC 7396](https://www.rfc-editor.org/rfc/rfc7396)).
PATCH requires the same UPDATE permission as PUT, with no additional GET permission.
It updates existing live resources only: missing names and tombstones return 404.

- PUT replaces the desired resource and requires the revision from the original read.
- PATCH with `metadata.resourceVersion` (a positive decimal string) keeps that
  original precondition. A changed desired state with a stale revision returns 409;
  the server never refreshes the supplied revision to replay the edit.
- PATCH without `metadata.resourceVersion` applies to current state. The server
  still writes with CAS, recomputing the patch from a fresh read on conflicts, for
  at most five attempts; exhaustion returns 409. Unrelated concurrent changes are
  preserved. Later patches to the **same field overwrite earlier values**. This
  is not field ownership, Server-Side Apply, or same-field lost-update protection.

Omitted fields are preserved. Objects merge recursively; null deletes map entries
or clears nullable map, slice, and pointer fields. Non-nullable scalars and required
objects cannot be removed. Arrays replace the entire array (including an empty
array), never merge by index. Unknown fields are rejected. Identity fields, when
supplied, must match the resource at the path. Status and creation/deletion timestamps
are server-owned and may not appear in a patch. Revision is a precondition, not a
client-assigned next revision. Existing update validation applies to the merged result.
An already-satisfied patch, including a retry after a lost response, returns the
current object/revision without a write, revision increment, or propagation.

```sh
# No GET required; other attributes and all omitted spec fields are preserved.
opampctl patch agentpackage collector -n default \
  -p '{"metadata":{"attributes":{"channel":"stable","obsolete":null}}}'
# Conditional edit using the revision retained from the original read.
opampctl patch agentgroup production -n default \
  -p '{"metadata":{"resourceVersion":"12"},"spec":{"priority":20}}'
# A JSON patch document can also come from a file or stdin.
opampctl patch namespace production -f patch.json
cat patch.json | opampctl patch namespace production -f -
```

Go clients expose `PatchAgentGroup`, `PatchAgentPackage`, `PatchAgentRemoteConfig`
and `PatchNamespace`, accepting raw JSON bytes and explicit path identity. They
send one PATCH and forward supplied revisions unchanged. Web editors continue to
use conditional PUT and retain the revision from when the editor was opened.
