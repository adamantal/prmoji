## prmoji Helm chart

This chart deploys **prmoji** as a Kubernetes **Deployment** with a **Service** and optional **Ingress**.
It also creates a **suspended Job** that can be manually unsuspended to trigger `POST /cleanup/`.

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

### Configuration

Runtime config maps to the app env vars:

- `config.port` → `PORT`
- `config.logLevel` → `LOG_LEVEL`
- `config.retentionDays` → `RETENTION_DAYS`
- `DB_PATH` is set automatically to `<persistence.mountPath>/prmoji.db`
- `config.ignoredCommenters` → `IGNORED_COMMENTERS`

Secrets:

This chart requires an **existing Kubernetes Secret**. Set `secret.existingSecret` to its name and ensure it contains:

- `SLACK_TOKEN` (**required**)

### Persistence (SQLite)

Persistence is **enabled by default** (a PVC is created and mounted).
The SQLite file path is always `<persistence.mountPath>/prmoji.db`.

It is not recommended to disable persistence, as it will lose all data on pod restart.

The container runs as a **non-root** user; this chart sets `podSecurityContext.fsGroup=10001` by default so the mounted volume is writable across common storage classes.

### Cleanup Job

A Job named `<release>-prmoji-cleanup` is created with `spec.suspend: true` by default.
To run it once, unsuspend it:

```bash
kubectl patch job <release>-prmoji-cleanup -n <ns> -p '{"spec":{"suspend":false}}'
```

You can configure whether it calls the in-cluster Service or the public Ingress:

- `cleanupJob.useIngress=false` (default) → calls `http://<service>/cleanup/`
- `cleanupJob.useIngress=true` → calls `https://<ingress-host>/cleanup/`
