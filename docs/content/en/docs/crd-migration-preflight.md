---
title: CRD migration preflight
description: Record the current ownership and rehearse a reversible cutover in an isolated environment
---

Rehearse the migration on a restored database and an isolated Kubernetes
installation. Keep one configuration writer active for an installation at a
time. Use this procedure to collect evidence before changing the writer.

## Record the boundary

1. Record the source image and chart versions, MongoDB database and backup
   identifier, and Kubernetes contexts. Record the installation ConfigMap and
   Namespace UIDs at the source where they exist. Record the expected target
   installation and Namespace UIDs alongside each source identity. Include
   existing nondefault namespaces and at least one recreated namespace in the
   rehearsal.
2. Inventory legacy desired resources by kind, namespace and name. Separately
   inventory Agent instance UIDs, the current application and report
   revisions, and the connection credential issuer. Associate every mapping
   with its source UID or legacy identity.
3. Write down collisions and their resolutions before generating manifests.
   Check that name normalization preserves equal-priority AgentGroup order.
   Treat numeric MongoDB resource versions as source metadata. Kubernetes
   assigns its own object UIDs and resource versions.
4. Store credentials and their export separately from the inventory. Use a
   protected Secret manifest for private keys, connection credentials and
   request headers. Collect names, counts, and revision identifiers in the
   ordinary report.

## Inspect the target without applying configuration

Use an explicit target context throughout. Run the commands from a protected
directory containing the generated configuration CRs:

```sh
kubectl opamp runtime-scope --context cutover-rehearsal --namespace collectors \
  --installation-namespace opampcommander-operator-system \
  --installation-name opampcommander-operator-opamp-installation -o json

kubectl opamp validate --context cutover-rehearsal -n collectors -f collectors-crs.yaml
```

Compare the returned installation and namespace UIDs with the target entries
in the mapping. A recreated Namespace or installation ConfigMap has a new UID
and requires an explicit migration mapping before reusing any legacy records.
Record whether each referenced Secret and catalog exists in the intended
namespace. Check source-to-target counts and compare each cohort Agent's
resolved configuration keys and intended content hashes before enabling
delivery. API dry-run only validates against the admission rules installed in
the target cluster. Verify delivery and acknowledgment separately with live
cohort Agents.

## Record the writer transition and recovery gates

| Gate | Evidence to capture |
| --- | --- |
| Before writer freeze | Restorable MongoDB backup, resolved name mapping, credential rotation and rollback plan, and current reported Agent hashes |
| With legacy writes fenced | A failed legacy configuration write and a verified inventory of queued or pending Agent work |
| Before expanding from the cohort | Matching expected desired and reported hashes, healthy Agent reconnect, and a known outcome for every previously pending operation |
| Before rollback | The new writer is fenced, the backup corresponds to the recorded UID mapping, and the previous credential issuer remains usable |
| After rollback | The previous writer is active alone, reports resume, and the cohort returns to the recorded baseline |

Record failed preflight checks and stop the rehearsal before enabling the new
writer. Preserve the original backup for rollback. Schedule data deletion as
an explicit operation after retention and cutover review.

The [control-plane design](../crd-control-plane/) describes the remaining
migration gates. The rehearsal requires a conversion and import path for the
legacy resources you use.
