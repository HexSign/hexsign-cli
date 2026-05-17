<p align="center">
  <a href="https://hexsign.io">
    <img src="https://hexsign.io/logo.png" alt="HexSign" height="64" />
  </a>
</p>

<h1 align="center">HexSign CLI</h1>

<p align="center">
  Manage Apple signing material — certificates, provisioning profiles, devices, and identifiers — from your terminal or your CI pipeline.
</p>

<p align="center">
  <a href="https://hexsign.io">hexsign.io</a>
  &nbsp;·&nbsp;
  <a href="https://dashboard.hexsign.net">Dashboard</a>
  &nbsp;·&nbsp;
  <a href="LICENSE">MIT License</a>
  &nbsp;·&nbsp;
  <a href="SECURITY.md">Security</a>
</p>

---

## Install

```sh
brew install hexsign
# or download a pre-built binary from the releases page
```

The released binaries are zero-config — log in and you're done. Building
from source requires one extra step (the OAuth client ID is injected at
build time); see [Development](#development).

## Authentication

The CLI runs in one of two modes, picked automatically from the environment:

| Mode | When | Source of credentials |
|------|------|----------------------|
| **user** (default) | local development | OAuth2 Authorization Code + PKCE through your browser; long-lived refresh token kept in the OS keychain |
| **machine** | CI | OAuth2 Client Credentials grant; access token cached in memory only |

### Local: `hexsign login`

```sh
hexsign login
```

This opens a browser to `identity.hexsign.net`, captures the authorization code on `http://localhost:53682/callback`, exchanges it for tokens, and stashes the refresh token in your OS keychain (`hexsign-cli` / `refresh_token`). On subsequent calls the CLI silently refreshes the ID token.

### CI: client credentials

Service credentials are provisioned per-organization and bound to a fixed scope set (`read`, `write`). The org owner creates one in the dashboard under **Settings → CLI Tokens**; the `client_secret` is shown exactly once.

Store them as encrypted secrets in your pipeline:

```yaml
env:
  HEXSIGN_CLIENT_ID:     ${{ secrets.HEXSIGN_CLIENT_ID }}
  HEXSIGN_CLIENT_SECRET: ${{ secrets.HEXSIGN_CLIENT_SECRET }}
  HEXSIGN_CLIENT_SCOPES: hexsign-api/read hexsign-api/write   # optional
```

When both `HEXSIGN_CLIENT_ID` and `HEXSIGN_CLIENT_SECRET` are present, the CLI fetches a fresh access token from `identity.hexsign.net/oauth2/token` on each invocation. The API enforces:

- `GET` requests require `hexsign-api/read`.
- All other methods require `hexsign-api/write`.
- Routes under `/users`, `/subscriptions`, `/contact-form`, `/service-credentials` are blocked for machine tokens (those manage humans / billing).
- The `last_used_at` timestamp is updated on every successful call so you can spot stale or compromised credentials in the dashboard.

Revocation is immediate (dashboard → Settings → CLI Tokens → Revoke).

## Configuration

Released binaries are zero-config. The only customer-settable knob is the
local OAuth callback port, which you'll only need to touch if `53682` is
already in use on your machine:

```sh
hexsign config show
hexsign config set callback_port 53683
```

<details>
<summary>Advanced / contributor overrides</summary>

Internal protocol endpoints (`api_base_url`, `cognito_domain`, `origin`,
`user_client_id`, `scopes`) are baked in at build time and overridable
for staging only via env vars: `HEXSIGN_API_BASE_URL`,
`HEXSIGN_COGNITO_DOMAIN`, `HEXSIGN_ORIGIN`, `HEXSIGN_CLI_CLIENT_ID`.
They are intentionally not in `hexsign config set`.

</details>

## Commands

```sh
hexsign login | logout | whoami
hexsign config (show | set <key> <value>)

hexsign apple-accounts list
hexsign apple-accounts delete <id|team_id>

hexsign certificates list [--type <t>] [--status <s>] [--team-id <id>] [--page N --limit N]
hexsign certificates get <id>
hexsign certificates download <id> [--output-dir DIR] [--filename NAME]
hexsign certificates download --type <t> --team-id <id> [--output-dir DIR]
hexsign certificates revoke <id>
hexsign certificates expiring

hexsign profiles list [--type <t>] [--status <s>] [--bundle-id <id>] [--team-id <id>] [--page N --limit N]
hexsign profiles get <id>
hexsign profiles download <id> [--output-dir DIR] [--filename NAME]
hexsign profiles download --bundle-id <id> [--team-id <id>] [--output-dir DIR]
hexsign profiles regenerate <id>
hexsign profiles delete <id>
hexsign profiles expiring

hexsign identifiers list [--bundle-id <id>] [--team-id <id>] [--page N --limit N]
hexsign identifiers get    <id>
hexsign identifiers create --apple-account-id <id> --bundle-id <id> --name <name> [--platform IOS|MAC_OS] [--type APP_IDS]
hexsign identifiers delete <id>

hexsign devices list [--device-class <c>] [--status <s>] [--team-id <id>] [--page N --limit N]
hexsign devices get    <id>
hexsign devices create --apple-account-id <id> --name <name> --udid <udid> [--platform IOS|MAC_OS]

hexsign summary
```

All commands accept `-o table|json` (default `table`).

### Bulk downloads

Both `certificates download` and `profiles download` accept a filter
instead of a single ID for fetching every matching artefact in one go:

- `hexsign certs download --type IOS_DISTRIBUTION --team-id ABCDE12345` —
  downloads every distribution certificate for the given Apple Developer team
  as `.p12` + `.password` pairs. `--team-id` is required so you don't
  accidentally pull certs across multiple linked Apple accounts.
- `hexsign profiles download --bundle-id com.example.app [--team-id ABCDE12345]`
  — downloads every `.mobileprovision` linked to that bundle identifier.
  Pass `--team-id` when the same bundle id exists in more than one linked
  Apple account.

Run `hexsign certs list --help` or `hexsign profiles list --help` for the
full list of accepted `--type` values.

## CI example: fetch signing material before xcodebuild

```yaml
# .github/workflows/release.yml
- name: Fetch signing material
  env:
    HEXSIGN_CLIENT_ID:     ${{ secrets.HEXSIGN_CLIENT_ID }}
    HEXSIGN_CLIENT_SECRET: ${{ secrets.HEXSIGN_CLIENT_SECRET }}
    PROFILE_ID:            ${{ vars.HEXSIGN_PROFILE_ID }}
    CERT_ID:               ${{ vars.HEXSIGN_CERT_ID }}
  run: |
    hexsign certificates download "$CERT_ID" --output-dir build/sign
    hexsign profiles     download "$PROFILE_ID" --output-dir build/sign
    # then security import / xcrun fastlane / xcodebuild as usual
```

## Development

The OAuth client ID is injected via `-ldflags` at build time so distributed
binaries don't require any env vars. From source:

```sh
make tidy
make build HEXSIGN_CLI_CLIENT_ID=<terraform output cli_user_client_id>
# or, set the env var once and forget about it:
export HEXSIGN_CLI_CLIENT_ID=<id>
make build
make test
```

Without an injected client ID, `hexsign login` falls back to the
`HEXSIGN_CLI_CLIENT_ID` env var at runtime and prints a clear error if
neither is set.

Verbose HTTP errors include the JSON body envelope from the API
(`error.code`, `error.message`).

## Contributing & security

- Bugs / feature requests: open a GitHub issue.
- Security vulnerabilities: see [SECURITY.md](SECURITY.md). Please do **not**
  open a public issue.

## License

[MIT](LICENSE).

## Security notes

* Refresh tokens never touch disk in plain text — they live in the OS keychain (Keychain on macOS, Secret Service on Linux, Credential Manager on Windows).
* Short-lived ID/access tokens are cached in `~/.../hexsign/tokens.json` with `0600` perms.
* Downloaded `.p12` and `.password` files are written `0600`.
* Keep CI client secrets in encrypted secrets stores; rotate periodically.