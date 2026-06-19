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

#### Automatic setup for orgs (recommended)

The recommended way to configure GitHub webhooks — especially in larger organizations — is to use the built-in `prmoji` CLI. This lets you manage webhooks across many repositories at once using a GitHub Personal Access Token (PAT).

1. Create a PAT with scopes that allow managing repository webhooks in your org (for GitHub.com, `admin:repo_hook` plus appropriate `repo` access is sufficient for private repos).
2. Export the token:

```bash
export GITHUB_TOKEN=ghp_...
```

3. Run the setup command (dry-run first):

```bash
prmoji github-webhook-setup \
  --org YOUR-ORG \
  --url YOUR_HOST \
  --dry-run
```

4. If the output looks good, run without `--dry-run`:

```bash
prmoji github-webhook-setup \
  --org YOUR-ORG \
  --url YOUR_HOST
```

By default this:

- Targets all **non-archived, non-fork** repositories in the org that the token can see.
- Creates or updates a webhook with:
  - Payload URL `https://YOUR_HOST/event/github`
  - Content type `application/json`
  - Events: **Issue comments**, **Pull requests**, **Pull request reviews**, and **Pull request review comments**.

Additional useful flags:

- `--include-pattern` / `--exclude-pattern`: Go regexes to include/exclude repositories by name.
- `--include-forks`: Also manage webhooks for forked repositories.
- `--include-archived`: Also manage webhooks for archived repositories.
- `--secret`: Set a webhook secret (handy if you later add signature verification).

If you prefer to configure a single repository manually instead, or you only need a one-off setup, follow the steps below.

#### Manual per-repo setup

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
SLACK_TOKEN='xoxb-...' ./prmoji run
```

## Other

### Emoji mapping

For a **single PR** in a Slack message (first link, slot 0):

- **commented** → `speech_balloon`
- **commented by GitHub Copilot** → `robot_face`
- **approved** → `white_check_mark`
- **changes requested** → `no_entry`
- **merged** → `pr-merged` *(custom emoji may be required in your Slack workspace)*
- **closed (not merged)** → `wastebasket`

#### Multiple PRs in one message

- Up to **9** PR URLs per message are tracked (link order in the text = PR #1 … #9).
- Each PR gets a stable **slot** (0-based in storage; **#1–#9** when labeling custom art).
- The same GitHub action uses a **different emoji per slot** so Slack does not reject duplicate reaction names
- Slot 0 uses the emojis above; slots 1–8 use the next entries in each action’s nine-emoji pool (see `internal/util/emoji.go`).
- Numbered custom workspace emojis (e.g. `prmoji-approved-3` for the 3rd PR’s approval) can replace the interim standard names using the pattern `prmoji-{action}-{N}` where `N` is the 1-based slot number.

#### Configuring emoji pools

Each action has a pool of **9** comma-separated Slack emoji names (one per PR slot). If unset, built-in defaults are used.

| Environment variable | Action |
|------------------------|--------|
| `EMOJI_POOL_COMMENTED` | commented |
| `EMOJI_POOL_APPROVED` | approved |
| `EMOJI_POOL_CHANGES_REQUESTED` | changes requested |
| `EMOJI_POOL_MERGED` | merged |
| `EMOJI_POOL_CLOSED` | closed (not merged) |

Example:

```bash
export EMOJI_POOL_CHANGES_REQUESTED='prmoji-changes-1,prmoji-changes-2,prmoji-changes-3,prmoji-changes-4,prmoji-changes-5,prmoji-changes-6,prmoji-changes-7,prmoji-changes-8,prmoji-changes-9'
```

With Helm, set `config.emojiPools.changesRequested` in [`charts/prmoji/values.yaml`](charts/prmoji/values.yaml) (see the chart README).

### Endpoints

- `GET /` → `OK`
- `GET /healthz` → `OK`
- `POST /event/slack` → Slack Events API callback (also handles Slack URL verification challenges)
- `POST /event/github` → GitHub webhook callback
- `POST /cleanup/` → deletes old rows (also runs automatically once per day)

## Notes / limitations

- **No signature verification**: Slack/GitHub request signature verification is not implemented. Deploy behind HTTPS and consider restricting ingress to Slack/GitHub IP ranges and/or a private network.
- **PR URL matching**: only matches URLs of the form `https://github.com/<owner>/<repo>/pull/<number>`.
- **Multiple PRs**: at most 9 PR URLs per Slack message; additional links in the same message are ignored.