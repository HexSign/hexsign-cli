# Security policy

## Reporting a vulnerability

If you've found a security issue in the HexSign CLI, please report it
**privately** — do not open a public GitHub issue.

Preferred channel: open a [GitHub Security Advisory](https://github.com/hexsign/hexsign-cli/security/advisories/new).
This keeps the report private until a fix is released and lets us
collaborate on a CVE if one is warranted.

If GitHub Security Advisories aren't an option, email
**support@hexsign.io** instead. Please include:

- A description of the issue and its impact.
- Steps to reproduce, ideally a minimal proof of concept.
- The CLI version (`hexsign --version`) and your OS / arch.
- Whether the issue is also exploitable through the dashboard or API
  directly — that affects how we triage.

## What to expect

- We acknowledge new reports within **3 business days**.
- Triage and fix targets follow severity (CVSS-style):
  - Critical / High: patch within 7 days.
  - Medium: patch within 30 days.
  - Low: rolled into the next regular release.
- We coordinate disclosure with you. By default we publish an advisory
  and CVE (if applicable) once a fix has shipped to the latest release.
- We're happy to credit you in the advisory if you'd like; let us know.

## Scope

In scope:

- The CLI binary and its source code (this repository).
- The way it stores tokens (OS keychain + on-disk caches).
- How it handles `.p12` / `.mobileprovision` material on disk.
- Authorization logic against the HexSign API.

Out of scope (please report through the appropriate channel):

- HexSign API / dashboard issues — open a security advisory on the
  hexsign-api or hexsign-dashboard repository instead, or email
  support@hexsign.io.
- Issues that require physical access to an unlocked machine running the
  CLI (those are addressed at the OS / disk-encryption layer).
- Findings against very old releases — please verify against the latest
  release first.

## Supported versions

Only the latest minor release receives security fixes. Older releases
will not be back-ported.

## Verifying release artifacts

Every release archive and the `checksums.txt` file are signed with
[cosign](https://docs.sigstore.dev/) keyless signing. The signature
(`*.sig`) and the short-lived signing certificate (`*.pem`) are uploaded
alongside each artifact, and a transparency log entry is published to
[Rekor](https://rekor.sigstore.dev/).

To verify a download:

```sh
VER=v1.2.3                                # the release tag
ART=hexsign_${VER#v}_darwin_arm64.tar.gz  # or whichever artifact

cosign verify-blob \
  --certificate "${ART}.pem" \
  --signature   "${ART}.sig" \
  --certificate-identity-regexp 'https://github.com/hexsign/hexsign-cli/.+' \
  --certificate-oidc-issuer 'https://token.actions.githubusercontent.com' \
  "${ART}"
```

The same command works for `checksums.txt`. If the verification succeeds
you have proof that this exact file was produced by our release workflow
running on the tagged commit.

## Hardening reminders for users

- Treat `.p12` / `.password` files as secrets — they're written `0600`
  by default; keep them off shared storage.
- Rotate CI client secrets periodically (Settings → CLI Tokens in the
  dashboard). Revocation is immediate.
- Run the CLI on machines with full-disk encryption enabled. The OS
  keychain protects the refresh token, but cached short-lived tokens
  live in `~/.config/hexsign/tokens.json`.
