## prmoji Helm chart

This chart deploys **prmoji** as a Kubernetes **Deployment** with a **Service** and optional **Ingress** / **HTTPRoute**.

### Install

```bash
helm upgrade --install prmoji ./charts/prmoji \
  --namespace prmoji --create-namespace \
  --set secret.slackToken='xoxb-...'
```

### Ingress

Enable ingress and set your host:

```bash
helm upgrade --install prmoji ./charts/prmoji \
  --set ingress.enabled=true \
  --set ingress.hosts[0].host=prmoji.example.com
```

### HTTPRoute (Gateway API)

Enable HTTPRoute and set your Gateway `parentRefs`:

```bash
helm upgrade --install prmoji ./charts/prmoji \
  --set httpRoute.enabled=true \
  --set httpRoute.parentRefs[0].name=your-gateway \
  --set httpRoute.parentRefs[0].namespace=default
```

### Configuration

Runtime config maps to the app env vars:

- `config.port` → `PORT`
- `config.logLevel` → `LOG_LEVEL`
- `config.retentionDays` → `RETENTION_DAYS`
- `DB_PATH` is set automatically to `<persistence.mountPath>/prmoji.db`
- `config.ignoredCommenters` → `IGNORED_COMMENTERS`
- `config.suppressAuthorComments` → `SUPPRESS_AUTHOR_COMMENTS` (default `true`)
- `config.emojiPools.*` → `EMOJI_POOL_*` (optional; each pool is a list of **9** Slack emoji names, comma-joined in the ConfigMap)

#### Custom emoji pools (multi-PR)

Override the nine reactions used per PR slot (link order in the Slack message). Omitted or empty lists use the app defaults.

```bash
helm upgrade --install prmoji ./charts/prmoji \
  --set secret.existingSecret=prmoji-slack \
  --set 'config.emojiPools.changesRequested={no_entry,x,negative_squared_cross_mark,warning,no_entry_sign,stop_sign,imp,rage,cursing_face}'
```

Or in `values.yaml`:

```yaml
config:
  emojiPools:
    changesRequested:
      - prmoji-changes-1
      - prmoji-changes-2
      # ... exactly 9 entries total
      - prmoji-changes-9
```

Environment variables (also work outside Kubernetes): `EMOJI_POOL_COMMENTED`, `EMOJI_POOL_APPROVED`, `EMOJI_POOL_CHANGES_REQUESTED`, `EMOJI_POOL_MERGED`, `EMOJI_POOL_CLOSED`.

Secrets:

This chart requires an **existing Kubernetes Secret**. Set `secret.existingSecret` to its name and ensure it contains:

- `SLACK_TOKEN` (**required**)

### Persistence (SQLite)

Persistence is **enabled by default** (a PVC is created and mounted).
The SQLite file path is always `<persistence.mountPath>/prmoji.db`.

It is not recommended to disable persistence, as it will lose all data on pod restart.

The container runs as a **non-root** user; this chart sets `podSecurityContext.fsGroup=10001` by default so the mounted volume is writable across common storage classes.
