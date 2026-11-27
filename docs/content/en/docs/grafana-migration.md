---
title: Move standalone web workflows to Grafana
description: Choose a replacement for each standalone UI workflow and verify it before retiring the old service
---

Grafana is the graphical entry point for Commander fleet visibility. Use
Kubernetes for configuration and permission changes. Review the workflows in
use before removing the standalone web Deployment.

| Standalone workflow | Replacement or status |
| --- | --- |
| Agent inventory, connection and reported health | Grafana Fleet overview for the data source's configured namespace. |
| Agent report age | Grafana's last-reported field. Refresh after a reconnect to see current state. |
| Agent detail links, current desired/reported config and rollout progress | Track readiness in the Grafana migration issue. Preserve access to the old workflow until the required read views are verified. |
| AgentGroup membership and configuration | Inspect and change namespaced AgentGroups with native kubectl. Use a singleton selector for one Agent. |
| Remote configuration, schemas, endpoints and packages | Manage their configuration CRs with native kubectl. Validate manifests with `kubectl opamp validate`. |
| Connection settings and certificates | Store sensitive inputs in Kubernetes Secrets. Review the installation's certificate rotation procedure before moving enrollment and delivery. |
| Namespaces, users, roles and bindings | Manage native Namespace, Role and RoleBinding resources and use your Kubernetes identity provider. |
| Application, connection and server inventories | Track projected views or retirement in the Grafana migration issue. Retain operational access where these workflows are still required. |

Follow the [Helm Grafana setup](https://github.com/minuk-dev/opampcommander/tree/main/deploy/charts/opampcommander#optional-grafana-preview) for the deployed plugin image and
data source. Use a private Commander upstream or an access-controlled proxy.
Choose a namespace explicitly and verify that access to the Grafana data source
matches the intended audience for that namespace.

## Validate the migration

- Check that Grafana opens Fleet overview and shows a known Agent's health,
  connection and last-report timestamp with the standalone Service disabled in
  a rehearsal deployment.
- Refresh after an Agent reconnects and verify that another namespace's Agents
  stay outside the data source's results.
- Verify every workflow your team uses, including any Agent detail links and
  configuration status. Record the workflows that require additional migration
  work with [the Grafana migration](https://github.com/minuk-dev/opampcommander/issues/909).
- Keep the previous values and image tags for rollback. Once the required
  workflows pass, set `web.enabled: false` in the release values and apply the
  Helm upgrade. Remove any independently managed standalone ingress or
  Deployment in the same maintenance window.
