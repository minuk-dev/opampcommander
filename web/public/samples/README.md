# Editor samples

These YAML files back the **Load sample** dropdown in the web UI's resource
editors. Each file is a YAML list of `{ label, description, value }` entries —
add, remove, or rewrite samples here without touching TypeScript.

The shape is identical to the resource body the editor saves, so a sample is
just a valid request payload pre-filled.

## Tokens

Place-holders are substituted at fetch time:

| Token           | Meaning                                                    |
|-----------------|------------------------------------------------------------|
| `{{namespace}}` | The currently selected namespace (for namespace-scoped UIs) |
| `{{now}}`       | Current ISO-8601 timestamp                                 |

## Files

| File | Used by |
|---|---|
| `roles.yaml` | `/roles` create + edit |
| `rolebindings.yaml` | `/rolebindings` create + edit |
| `certificates.yaml` | `/certificates` create + edit |
| `users.yaml` | `/users` create |
| `agentpackages.yaml` | `/agentpackages` create + edit |
| `agentremoteconfigs.yaml` | `/agentremoteconfigs` create + edit |
| `agentspecs.yaml` | Agent detail "Edit spec" dialog (spec subtree only) |
| `agentgroups.yaml` | `/agentgroups` create + edit (entries include `name`, `attributes`, `spec`) |

## Adding a sample

```yaml
- label: Short human-readable name shown in the menu
  description: One-liner shown as secondary text
  value:
    # ...the payload, exactly as you would type it in the editor...
```

Save the file — the next time the dialog is opened the new entry appears.
