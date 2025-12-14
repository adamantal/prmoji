# prmoji

A tiny web service that adds emoji reactions to Slack messages when the GitHub Pull Requests mentioned in those messages get reviewed/commented/merged/closed.

- Influenced by [`endreymarcell/prmoji`](https://github.com/endreymarcell/prmoji)
- Redesigned in Go with **zero external service dependencies**: a single binary + a local **SQLite** DB file (SQLite should comfortably handle up to **dozens of requests/sec** for this use case)

## How does it work?

1. Invite the `prmoji` Slack bot to a channel.
2. When someone posts a GitHub Pull Request URL in that channel, `prmoji` stores a mapping:
   - PR URL → (Slack channel ID, Slack message timestamp)
3. When GitHub sends webhook events for that PR, `prmoji` looks up the stored Slack message(s) and adds a matching emoji reaction.

## Deploying (Kubernetes)

The recommended way to deploy `prmoji` to Kubernetes is via the **Helm chart**.

1) Create a Secret (must contain `SLACK_TOKEN`):

```bash
kubectl create namespace prmoji
kubectl -n prmoji create secret generic prmoji-secrets \
  --from-literal=SLACK_TOKEN='xoxb-...'
```

2) Install the chart (OCI on GHCR):

```bash
helm upgrade --install prmoji oci://ghcr.io/adamantal/charts/prmoji \
  --namespace prmoji \
  --set secret.existingSecret=prmoji-secrets
```

Notes:

- **Persistence**: enabled by default (a PVC is created and mounted). The SQLite DB file is stored at `/data/prmoji.db`.
- **Ingress**: enable and configure host(s) via chart values:

```bash
helm upgrade --install prmoji oci://ghcr.io/adamantal/charts/prmoji \
  --namespace prmoji \
  --set secret.existingSecret=prmoji-secrets \
  --set ingress.enabled=true \
  --set ingress.hosts[0].host=prmoji.example.com
```

- **Cleanup Job**: the chart includes a suspended Job you can manually unsuspend to trigger `POST /cleanup/`.

## Setup

### Slack

This only has to be done once per Slack workspace.

- Go to `https://api.slack.com/apps/`
- Click **Your Apps** → **Create New App**
- Create an app (e.g. “prmoji”) and select your workspace
- Create/enable a **Bot User**
- Under **OAuth & Permissions**, add the bot token scope:
  - `reactions:write`
- Under **Event Subscriptions**:
  - Enable events
  - Set **Request URL** to `https://YOUR_HOST/event/slack`
  - Under **Subscribe to bot events**, add:
    - `message.channels`
    - `message.groups`
- **Install App** to your workspace
- Copy the **Bot User OAuth Token** and set it as `SLACK_TOKEN`
- Invite the bot to any channel where it should listen

### GitHub

This has to be done for every repository you want to watch.

- Go to `https://github.com/YOUR-ORG/YOUR-REPO/settings/hooks`
- Click **Add webhook**
- Set **Payload URL** to `https://YOUR_HOST/event/github`
- Set **Content type** to `application/json`
- Click **Let me select individual events**
- Select:
  - **Issue comments**
  - **Pull requests**
  - **Pull request reviews**
  - *(Optional)* **Pull request review comments**
- Click **Add webhook**

## Configuration

Environment variables:

- **Required**
  - `SLACK_TOKEN`: Slack bot token used for Slack Web API calls (`reactions.add`)
- **Optional**
  - `PORT`: HTTP listen port (default `5000`)
  - `LOG_LEVEL`: log level (default `info`)
  - `DB_PATH`: path to SQLite database file (default `./prmoji.db`)
  - `RETENTION_DAYS`: delete mappings older than N days (default `90`)
  - `IGNORED_COMMENTERS`: comma-separated GitHub usernames to suppress *comment* reactions for (default empty)

## Run locally

Build:

```bash
go build ./cmd/prmoji
```

Run:

```bash
SLACK_TOKEN='xoxb-...' ./prmoji
```

## Other

### Emoji mapping

- **commented** → `speech_balloon`
- **approved** → `white_check_mark`
- **changes requested** → `no_entry`
- **merged** → `pr-merged` *(custom emoji may be required in your Slack workspace)*
- **closed (not merged)** → `wastebasket`

### Endpoints

- `GET /` → `OK`
- `GET /healthz` → `OK`
- `POST /event/slack` → Slack Events API callback (also handles Slack URL verification challenges)
- `POST /event/github` → GitHub webhook callback
- `POST /cleanup/` → deletes old rows (also runs automatically once per day)

## Notes / limitations

- **No signature verification**: Slack/GitHub request signature verification is not implemented. Deploy behind HTTPS and consider restricting ingress to Slack/GitHub IP ranges and/or a private network.
- **PR URL matching**: only matches URLs of the form `https://github.com/<owner>/<repo>/pull/<number>`.